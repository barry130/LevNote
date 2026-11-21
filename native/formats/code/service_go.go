package code

import (
	"cmp"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"
)

func (s *serviceRun) goResult() ServiceResult {
	return s.goResultWithWork(&goServiceWork{limit: serviceGoWorkLimit})
}

// The private seam permits small deterministic budgets in package tests. Public
// requests always receive a fresh hard budget, never a shared or caller limit.
func (s *serviceRun) goResultWithWork(work *goServiceWork) (result ServiceResult) {
	work.run = s
	defer func() {
		// Cancellation overrides any competing failure, as in AnalyzeService.
		// Discard every payload, including prefixes retained before exhaustion.
		if s.stop() {
			result = s.failure
		}
	}()
	// T1 has already admitted and validated this source. Its API doesn't
	// expose an AST, so obtain a transient one, still with object resolution
	// disabled. The non-nil string input selects the memory path.
	files := token.NewFileSet()
	mode := parser.SkipObjectResolution
	if s.request.Operation == ServiceInfo {
		mode |= parser.ParseComments
	}
	file, err := parser.ParseFile(files, "", s.request.Source, mode)
	if s.stop() {
		return s.failure
	}
	if err != nil {
		return serviceStatus(ServiceUnsupported, "go-ast-unavailable")
	}
	if !s.checkGoDepth(file, work) {
		return s.failure
	}
	if s.request.Operation == ServiceCompletion {
		return s.goCompletion(file, work)
	}
	binding := goServiceBinding{run: s, work: work, file: files.File(file.Pos())}
	binding.bind(file)
	if s.stop() || !s.loadPoints() {
		return s.failure
	}
	if s.request.Operation == ServiceSymbols {
		return binding.symbols()
	}
	decl, failure := binding.target()
	if decl == nil {
		return failure
	}
	switch s.request.Operation {
	case ServiceDefinition:
		symbol := binding.symbol(decl)
		return ServiceResult{Status: ServiceComplete, Definition: &symbol}
	case ServiceReferences:
		return binding.references(decl)
	case ServiceInfo:
		return binding.info(decl)
	case ServiceRename:
		return binding.rename(decl)
	default:
		return serviceStatus(ServiceUnsupported, "operation")
	}
}

func (s *serviceRun) checkGoDepth(file *ast.File, work *goServiceWork) bool {
	// ast.Inspect can decline children before reaching a depth dangerous to
	// subsequent recursive walks. T1 delimiter depth alone doesn't bound a
	// deep delimiter-free expression or chain of labels.
	depth := 0
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			depth--
			return false
		}
		if !work.step() || !s.count(depth, serviceASTDepth, serviceASTResource) {
			return false
		}
		depth++
		return true
	})
	return !s.stop()
}

func (s *serviceRun) goCompletion(file *ast.File, work *goServiceWork) ServiceResult {
	vocab := serviceVocabulary{run: s, seen: make(map[string]struct{})}
	for keyword := token.BREAK; keyword <= token.VAR; keyword++ {
		if !work.step() || !vocab.add(keyword.String(), "keyword") {
			return s.failure
		}
	}
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		if !work.step() {
			return false
		}
		if id, ok := n.(*ast.Ident); ok && id.Name != "_" && id.Name != "." {
			return vocab.add(id.Name, "file-word")
		}
		return true
	})
	if s.stop() {
		return s.failure
	}
	return vocab.result()
}

func (b *goServiceBinding) target() (*goServiceDecl, ServiceResult) {
	item := b.at(b.run.request.ByteOffset)
	if item == nil {
		return nil, serviceStatus(ServiceUnsupported, "no-identifier")
	}
	switch item.role {
	case goServiceExcluded:
		return nil, serviceStatus(ServiceUnsupported, "identifier-role")
	case goServiceMaybeKey:
		return nil, serviceStatus(ServiceAmbiguous, "composite-key")
	case goServiceBadShort:
		return nil, serviceStatus(ServiceUnsupported, "short-or-type-switch-declaration")
	}
	decl := item.decl
	if decl == nil {
		decl, _ = item.lookup(item.id.Name, nil, nil)
		if b.run.stop() {
			return nil, b.run.failure
		}
		for probe := item.inherited; probe != nil; probe = probe.parent {
			if !b.work.step() {
				return nil, b.run.failure
			}
			alternate, _ := item.lookup(item.id.Name, probe, nil)
			if b.run.stop() {
				return nil, b.run.failure
			}
			if alternate != decl {
				return nil, serviceStatus(ServiceAmbiguous, "inherited-constant-binding")
			}
		}
	}
	if decl == nil || decl.unsupported || (decl.scope != nil && decl.scope.opaque) {
		return nil, serviceStatus(ServiceUnsupported, "unresolved-or-external")
	}
	if decl.ambiguous {
		return nil, serviceStatus(ServiceAmbiguous, "duplicate-declaration")
	}
	return decl, ServiceResult{}
}

func (b *goServiceBinding) identSpan(id *ast.Ident) StructureSpan {
	start := b.file.Offset(id.Pos())
	return b.run.span(start, start+len(id.Name))
}

func (b *goServiceBinding) symbol(d *goServiceDecl) ServiceSymbol {
	if !b.run.keepText(d.id.Name) {
		return ServiceSymbol{}
	}
	return ServiceSymbol{Name: strings.Clone(d.id.Name), Kind: d.kind, Span: b.identSpan(d.id)}
}

func (b *goServiceBinding) symbols() ServiceResult {
	out := make([]ServiceSymbol, 0, min(len(b.decls), b.run.limits.Symbols))
	for _, d := range b.decls {
		if !b.work.step() || !b.run.count(len(out), b.run.limits.Symbols, serviceSymbolsResource) {
			return b.run.failure
		}
		out = append(out, b.symbol(d))
	}
	if b.run.stop() {
		return b.run.failure
	}
	slices.SortFunc(out, func(a, b ServiceSymbol) int { return cmp.Compare(a.Span.Bytes.Start, b.Span.Bytes.Start) })
	return ServiceResult{Status: ServiceComplete, Symbols: out}
}

// matchingUse resolves a same-spelled occurrence. Unknown scopes and syntax
// roles that might use the target are surfaced so a complete reference list
// (or edit plan) cannot silently omit them.
func (b *goServiceBinding) matchingUse(item goServiceOccurrence, target *goServiceDecl) (match, uncertain bool) {
	if item.decl != nil || item.role == goServiceExcluded || item.id.Name != target.id.Name {
		return false, false
	}
	decl, opaque := item.lookup(item.id.Name, nil, nil)
	if b.run.stop() {
		return false, false
	}
	if opaque {
		for scope := item.scope; scope != nil; scope = scope.parent {
			if !b.work.step() {
				return false, false
			}
			if scope == target.scope {
				return false, true
			}
		}
		return false, false
	}
	for probe := item.inherited; probe != nil; probe = probe.parent {
		if !b.work.step() {
			return false, false
		}
		alternate, _ := item.lookup(item.id.Name, probe, nil)
		if b.run.stop() {
			return false, false
		}
		if alternate != decl && (alternate == target || decl == target) {
			return false, true
		}
	}
	if decl != target {
		return false, false
	}
	if item.role == goServiceMaybeKey || item.role == goServiceBadShort {
		return false, true
	}
	return true, false
}

func (b *goServiceBinding) references(d *goServiceDecl) ServiceResult {
	var refs []StructureSpan
	for _, item := range b.uses {
		if !b.work.step() {
			return b.run.failure
		}
		match, uncertain := b.matchingUse(item, d)
		if b.run.stop() {
			return b.run.failure
		}
		if uncertain {
			return serviceStatus(ServiceAmbiguous, "possible-reference")
		}
		if match {
			if !b.run.count(len(refs), b.run.limits.References, serviceReferencesResource) {
				return b.run.failure
			}
			refs = append(refs, b.identSpan(item.id))
		}
	}
	slices.SortFunc(refs, func(a, b StructureSpan) int { return cmp.Compare(a.Bytes.Start, b.Bytes.Start) })
	return ServiceResult{Status: ServiceComplete, References: refs}
}

func (b *goServiceBinding) info(decl *goServiceDecl) ServiceResult {
	symbol := b.symbol(decl)
	if b.run.stop() {
		return b.run.failure
	}
	source := b.run.request.Source
	signature := source[decl.signature.start:decl.signature.end]
	var doc string
	if group := decl.doc; group != nil && len(group.List) != 0 {
		start := b.file.Offset(group.Pos())
		last := b.file.Offset(group.List[len(group.List)-1].Pos())
		// go/scanner strips CRs from comments. Find the raw final comment
		// boundary in the already-validated source rather than using Text.
		if strings.HasPrefix(source[last:], "//") {
			end := strings.IndexByte(source[last:], '\n')
			if end < 0 {
				end = len(source) - last
			} else if end > 0 && source[last+end-1] == '\r' {
				end--
			}
			doc = source[start:last+end]
		} else if end := strings.Index(source[last+2:], "*/"); end >= 0 {
			doc = source[start:last+2+end+2]
		}
	}
	if !b.run.keepText(signature, doc) {
		return b.run.failure
	}
	return ServiceResult{Status: ServiceComplete, Info: &ServiceSymbolInfo{
		Symbol: symbol, Signature: strings.Clone(signature), Documentation: strings.Clone(doc),
	}}
}
