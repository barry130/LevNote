package code

import (
	"cmp"
	"go/ast"
	"go/token"
	"slices"
	"strings"
)

// renamedBinding checks whether the candidate spelling sees target at any or
// all lexical sites represented by this token, including inherited constants.
func renamedBinding(item goServiceOccurrence, target *goServiceDecl, name string) (anySite, allSites bool) {
	base, _ := item.lookup(name, nil, target)
	if item.scope.work.run.stop() {
		return false, false
	}
	anySite = base == target
	allSites = anySite
	for probe := item.inherited; probe != nil; probe = probe.parent {
		if !item.scope.work.step() {
			return false, false
		}
		alternate, _ := item.lookup(name, probe, target)
		if item.scope.work.run.stop() {
			return false, false
		}
		anySite = anySite || alternate == target
		allSites = allSites && alternate == target
	}
	return anySite, allSites
}

func (b *goServiceBinding) rename(target *goServiceDecl) ServiceResult {
	s := b.run
	name := s.request.NewName
	if len(name) > serviceFieldBytes {
		return serviceLimit(serviceFieldResource, serviceFieldBytes)
	}
	if !token.IsIdentifier(name) || name == "_" {
		return serviceStatus(ServiceInvalid, "invalid-new-name")
	}
	if target.scope == nil || !target.scope.local {
		return serviceStatus(ServiceUnsupported, "rename-local-only")
	}
	old := target.id.Name
	if len(old) > serviceFieldBytes {
		return serviceLimit(serviceFieldResource, serviceFieldBytes)
	}
	if name == old {
		return ServiceResult{Status: ServiceComplete, NoOp: true}
	}
	if target.scope.decls[name] != nil {
		return serviceStatus(ServiceAmbiguous, "rename-declaration-conflict")
	}

	// A naked return has no identifier token to replace but can still become
	// invalid if a renamed declaration hides a named result at that point.
	for _, site := range b.returns {
		if !b.work.step() {
			return s.failure
		}
		resolved, _ := site.scope.lookup(name, site.pos, nil, target)
		if s.stop() {
			return s.failure
		}
		if target.scope == site.results && target.kind == "result" && resolved != target {
			return serviceStatus(ServiceAmbiguous, "rename-captured-result")
		}
		if result := site.results.decls[name]; result != nil && result.kind == "result" && resolved == target {
			return serviceStatus(ServiceAmbiguous, "rename-captures-result")
		}
	}

	// Preflight every returned string byte and edit count before cloning names
	// or allocating the public result slice. Failed plans never retain a prefix.
	ids := make([]*ast.Ident, 0, min(len(b.uses)+1, s.limits.RenameEdits))
	plan := func(id *ast.Ident) bool {
		if !s.count(len(ids), s.limits.RenameEdits, serviceRenameEditsResource) || !s.keepText(old, name) {
			return false
		}
		span := b.identSpan(id)
		if s.request.Source[span.Bytes.Start:span.Bytes.End] != old {
			s.failure = serviceStatus(ServiceAmbiguous, "rename-original-spelling")
			return false
		}
		ids = append(ids, id)
		return true
	}
	if !plan(target.id) {
		return s.failure
	}
	for _, item := range b.uses {
		if !b.work.step() {
			return s.failure
		}
		if item.decl != nil || item.role == goServiceExcluded {
			continue
		}
		if item.id.Name == name {
			anySite, _ := renamedBinding(item, target, name)
			if s.stop() {
				return s.failure
			}
			if anySite {
				return serviceStatus(ServiceAmbiguous, "rename-captures-use")
			}
		}
		match, uncertain := b.matchingUse(item, target)
		if s.stop() {
			return s.failure
		}
		if uncertain {
			return serviceStatus(ServiceAmbiguous, "possible-reference")
		}
		if !match {
			continue
		}
		if item.role == goServiceEmbedded {
			return serviceStatus(ServiceUnsupported, "rename-embedded-field")
		}
		_, allSites := renamedBinding(item, target, name)
		if s.stop() {
			return s.failure
		}
		if !allSites {
			return serviceStatus(ServiceAmbiguous, "rename-captured-use")
		}
		if !plan(item.id) {
			return s.failure
		}
	}
	if s.stop() {
		return s.failure
	}
	oldText, newText := strings.Clone(old), strings.Clone(name)
	edits := make([]ServiceEdit, len(ids))
	for i, id := range ids {
		edits[i] = ServiceEdit{Span: b.identSpan(id), OldText: oldText, NewText: newText}
	}
	slices.SortFunc(edits, func(a, b ServiceEdit) int { return cmp.Compare(a.Span.Bytes.Start, b.Span.Bytes.Start) })
	return ServiceResult{Status: ServiceComplete, Edits: edits}
}
