package code

import (
	"context"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"
)

// ServiceOperation selects exactly one result payload and its output budgets.
type ServiceOperation string

const (
	ServiceDiagnostics ServiceOperation = "diagnostics"
	ServiceSymbols     ServiceOperation = "symbols"
	ServiceCompletion  ServiceOperation = "completion"
	ServiceDefinition  ServiceOperation = "definition"
	ServiceReferences  ServiceOperation = "references"
	ServiceInfo        ServiceOperation = "info"
	ServiceRename      ServiceOperation = "rename"
)

type ServiceStatus string

const (
	ServiceComplete    ServiceStatus = "complete"
	ServiceUnsupported ServiceStatus = "unsupported"
	ServiceAmbiguous   ServiceStatus = "ambiguous"
	ServiceInvalid     ServiceStatus = "invalid"
	ServiceLimit       ServiceStatus = "limit"
	ServiceCanceled    ServiceStatus = "canceled"
)

// ServiceLimits bounds one request. All fields must be positive and values
// above their respective hard ceilings are clamped. TextBytes counts retained
// source-derived text, including both strings in each planned edit.
type ServiceLimits struct {
	Structure                                  StructureLimits
	Symbols, Completion, References, RenameEdits int
	TextBytes                                  int
}

const (
	serviceFieldBytes = 16 << 10
	serviceASTDepth   = 512

	serviceSymbolsResource     StructureResource = "symbols"
	serviceCompletionResource  StructureResource = "completion"
	serviceReferencesResource  StructureResource = "references"
	serviceRenameEditsResource StructureResource = "rename-edits"
	serviceTextResource        StructureResource = "text-bytes"
	serviceFieldResource       StructureResource = "field-bytes"
	serviceASTResource         StructureResource = "go-ast-depth"
)

func DefaultServiceLimits() ServiceLimits {
	return ServiceLimits{
		Structure: DefaultStructureLimits(), Symbols: 4096, Completion: 512,
		References: 4096, RenameEdits: 1024, TextBytes: 512 << 10,
	}
}

// ServiceRequest carries a single immutable source snapshot. ByteOffset is an
// absolute byte position for semantic operations. Completion filters the
// vocabulary by the caller's literal Prefix; it has no replacement range.
type ServiceRequest struct {
	Source          string
	Profile         StructureProfile
	Operation       ServiceOperation
	ByteOffset      int
	Prefix, NewName string
	Limits          ServiceLimits
}

type ServiceSymbol struct {
	Name, Kind, Path string
	Span             StructureSpan
}

type ServiceCompletionItem struct{ Text, Kind string }

type ServiceSymbolInfo struct {
	Symbol                   ServiceSymbol
	Signature, Documentation string
}

// ServiceEdit shares the original snapshot's coordinates. OldText must match
// those exact bytes before a host can apply the complete plan.
type ServiceEdit struct {
	Span             StructureSpan
	OldText, NewText string
}

// ServiceResult owns its slices. Edits and lexical references (use sites
// only) are ordered by increasing source byte offset. Only ServiceComplete
// carries successful payloads. ServiceInvalid may carry syntax diagnostics.
type ServiceResult struct {
	Status      ServiceStatus
	Reason      string
	Resource    StructureResource
	Limit       int
	Diagnostics []StructureDiagnostic
	Symbols     []ServiceSymbol
	Completion  []ServiceCompletionItem
	Definition  *ServiceSymbol
	References  []StructureSpan
	Info        *ServiceSymbolInfo
	Edits       []ServiceEdit
	NoOp        bool
}

func serviceStatus(status ServiceStatus, reason string) ServiceResult {
	return ServiceResult{Status: status, Reason: reason}
}

func serviceLimit(resource StructureResource, limit int) ServiceResult {
	return ServiceResult{Status: ServiceLimit, Reason: "resource-limit", Resource: resource, Limit: limit}
}

type serviceRun struct {
	ctx       context.Context
	request   ServiceRequest
	limits    ServiceLimits
	failure   ServiceResult
	textBytes int
	points    []structurePoint
}

// AnalyzeService synchronously analyzes one supplied string in memory. Each
// invocation owns its transient state. T1 validation and the standard Go
// parser are synchronous phases, so cancellation discards their result when
// they return; service-owned walks check the context as they progress.
func AnalyzeService(ctx context.Context, req ServiceRequest) ServiceResult {
	if ctx == nil {
		return serviceStatus(ServiceInvalid, "missing-context")
	}
	run := serviceRun{ctx: ctx, request: req}
	if run.stop() {
		return run.failure
	}
	limits := req.Limits
	var err error
	limits.Structure, err = normalizeStructureLimits(limits.Structure)
	if err != nil {
		return serviceStatus(ServiceInvalid, "invalid-limits")
	}
	hard := DefaultServiceLimits()
	for _, field := range [...]struct {
		value *int
		max   int
	}{
		{&limits.Symbols, hard.Symbols}, {&limits.Completion, hard.Completion},
		{&limits.References, hard.References}, {&limits.RenameEdits, hard.RenameEdits},
		{&limits.TextBytes, hard.TextBytes},
	} {
		if *field.value <= 0 {
			return serviceStatus(ServiceInvalid, "invalid-limits")
		}
		*field.value = min(*field.value, field.max)
	}
	run.limits = limits
	if req.Profile != StructureGo && req.Profile != StructureJSON {
		return serviceStatus(ServiceUnsupported, "profile")
	}
	switch req.Operation {
	case ServiceDiagnostics, ServiceSymbols, ServiceCompletion:
	case ServiceDefinition, ServiceReferences, ServiceInfo, ServiceRename:
		if req.Profile != StructureGo {
			return serviceStatus(ServiceUnsupported, "operation-for-profile")
		}
	default:
		return serviceStatus(ServiceUnsupported, "operation")
	}
	if req.Operation == ServiceCompletion {
		if len(req.Prefix) > serviceFieldBytes {
			return serviceLimit(serviceFieldResource, serviceFieldBytes)
		}
		if !utf8.ValidString(req.Prefix) {
			return serviceStatus(ServiceInvalid, "invalid-prefix")
		}
	}
	checked, err := AnalyzeStructure(req.Source, req.Profile, limits.Structure)
	if run.stop() {
		return run.failure
	}
	if err != nil {
		var budget *StructureBudgetError
		switch {
		case errors.As(err, &budget):
			return serviceLimit(budget.Resource, budget.Limit)
		case errors.Is(err, ErrStructureValidationLimit):
			return serviceStatus(ServiceLimit, "go-validator-nesting")
		default:
			return serviceStatus(ServiceUnsupported, "validator-failed")
		}
	}
	if !checked.Complete {
		for _, diagnostic := range checked.Diagnostics {
			if !run.keepText(diagnostic.Message) {
				return run.failure
			}
		}
		return ServiceResult{Status: ServiceInvalid, Reason: "source", Diagnostics: checked.Diagnostics}
	}
	if req.Operation == ServiceDiagnostics {
		return ServiceResult{Status: ServiceComplete}
	}
	if req.Operation == ServiceDefinition || req.Operation == ServiceReferences || req.Operation == ServiceInfo || req.Operation == ServiceRename {
		if req.ByteOffset < 0 || req.ByteOffset > len(req.Source) ||
			(req.ByteOffset < len(req.Source) && !utf8.RuneStart(req.Source[req.ByteOffset])) {
			return serviceStatus(ServiceInvalid, "invalid-offset")
		}
	}
	var result ServiceResult
	if req.Profile == StructureGo {
		result = run.goResult()
	} else {
		result = run.jsonResult()
	}
	if run.stop() {
		return run.failure
	}
	return result
}

func (s *serviceRun) stop() bool {
	if s.ctx.Err() != nil {
		s.failure = serviceStatus(ServiceCanceled, "context")
	}
	return s.failure.Status != ""
}

func (s *serviceRun) count(size, limit int, resource StructureResource) bool {
	if size >= limit {
		s.failure = serviceLimit(resource, limit)
		return false
	}
	return true
}

func (s *serviceRun) keepText(fields ...string) bool {
	for _, field := range fields {
		if !s.keepBytes(len(field)) {
			return false
		}
	}
	return true
}

func (s *serviceRun) keepBytes(size int) bool {
	if size > serviceFieldBytes {
		s.failure = serviceLimit(serviceFieldResource, serviceFieldBytes)
		return false
	}
	if size > s.limits.TextBytes-s.textBytes {
		s.failure = serviceLimit(serviceTextResource, s.limits.TextBytes)
		return false
	}
	s.textBytes += size
	return true
}

// Sparse checkpoints let arbitrary token ranges use the T1 coordinate rules.
// Building them is linear and translating an endpoint visits at most one
// short interval, even on a multi-megabyte physical line.
const servicePointStride = 256

func (s *serviceRun) loadPoints() bool {
	if s.points != nil {
		return true
	}
	source := s.request.Source
	var point structurePoint
	s.points = append(s.points, point)
	for point.offset < len(source) {
		r, width := utf8.DecodeRuneInString(source[point.offset:])
		point.step(r, width)
		if point.offset-s.points[len(s.points)-1].offset >= servicePointStride {
			if s.stop() {
				return false
			}
			s.points = append(s.points, point)
		}
	}
	return !s.stop()
}

func (s *serviceRun) span(start, end int) StructureSpan {
	return s.pointAt(start).span(s.pointAt(end))
}

func (s *serviceRun) pointAt(offset int) structurePoint {
	i, found := slices.BinarySearchFunc(s.points, offset, func(p structurePoint, offset int) int {
		return p.offset - offset
	})
	if !found {
		i--
	}
	point := s.points[i]
	point.advance(s.request.Source, offset)
	return point
}

type serviceVocabulary struct {
	run   *serviceRun
	seen  map[string]struct{}
	items []ServiceCompletionItem
}

func (v *serviceVocabulary) add(word, kind string) bool {
	if !strings.HasPrefix(word, v.run.request.Prefix) {
		return true
	}
	if _, exists := v.seen[word]; exists {
		return true
	}
	if !v.run.count(len(v.items), v.run.limits.Completion, serviceCompletionResource) || !v.run.keepText(word) {
		return false
	}
	word = strings.Clone(word)
	v.seen[word] = struct{}{}
	v.items = append(v.items, ServiceCompletionItem{Text: word, Kind: kind})
	return true
}

func (v *serviceVocabulary) result() ServiceResult {
	slices.SortFunc(v.items, func(a, b ServiceCompletionItem) int { return strings.Compare(a.Text, b.Text) })
	return ServiceResult{Status: ServiceComplete, Completion: v.items}
}
