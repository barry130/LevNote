package code

import (
	"go/ast"
	"go/token"
)

// serviceGoWorkLimit is independent of source size and output counts. One
// request shares this ceiling across AST/binding visits, scope visits and
// inherited-constant probes, including rename capture and return-site checks.
const serviceGoWorkLimit = 1 << 20

const serviceGoWorkResource StructureResource = "go-binding-work"

type goServiceWork struct {
	run         *serviceRun
	limit, used int
}

func (w *goServiceWork) step() bool {
	// Check cancellation before both failure admission and the next unit of
	// actual work. This also keeps cancellation ahead of a simultaneous cap.
	if w.run.stop() {
		return false
	}
	if w.used >= w.limit {
		w.run.failure = serviceLimit(serviceGoWorkResource, w.limit)
		return false
	}
	w.used++
	return true
}

type goServiceRange struct{ start, end int }

type goServiceDecl struct {
	id          *ast.Ident
	kind        string
	scope       *goServiceScope
	visible     token.Pos
	ambiguous   bool
	unsupported bool
	signature   goServiceRange
	doc         *ast.CommentGroup
}

type goServiceScope struct {
	work   *goServiceWork
	parent *goServiceScope
	decls  map[string]*goServiceDecl
	local  bool
	opaque bool // file/receiver scope may hide lexical bindings the service cannot establish
}

func newGoServiceScope(parent *goServiceScope, local bool) *goServiceScope {
	scope := &goServiceScope{parent: parent, local: local}
	if parent != nil {
		scope.work = parent.work
	}
	return scope
}

func (s *goServiceScope) enter(d *goServiceDecl) {
	if !s.work.step() {
		return
	}
	if s.decls == nil {
		s.decls = make(map[string]*goServiceDecl)
	}
	if old := s.decls[d.id.Name]; old != nil {
		old.ambiguous = true
		d.ambiguous = true
		return
	}
	s.decls[d.id.Name] = d
}

// A local const spec with omitted expressions reuses the previous nonempty
// spec's tokens. It sees declarations made since that spec. One probe at the
// last omitted spec suffices: declarations only accumulate in this block.
type goServiceConstProbe struct {
	parent *goServiceConstProbe
	scope  *goServiceScope
	pos    token.Pos
}

// lookup sees the nearest lexically visible declaration. An optional probe
// changes visibility in the const block alone. A rename overlay adds the
// candidate spelling after the caller has ruled out same-block conflicts.
func (s *goServiceScope) lookup(name string, pos token.Pos, probe *goServiceConstProbe, moved *goServiceDecl) (d *goServiceDecl, opaque bool) {
	for ; s != nil; s = s.parent {
		if !s.work.step() {
			return nil, false
		}
		if s.opaque {
			return nil, true
		}
		at := pos
		if probe != nil && s == probe.scope {
			at = probe.pos
		}
		if moved != nil && s == moved.scope && at >= moved.visible {
			return moved, false
		}
		if d := s.decls[name]; d != nil && at >= d.visible {
			return d, false
		}
	}
	return nil, false
}

type goServiceRole uint8

const (
	goServiceUse goServiceRole = iota
	goServiceExcluded  // fields, selectors, labels, imports, blank names
	goServiceMaybeKey  // a bare composite key may name a field or a lexical operand
	goServiceEmbedded  // a type use that also determines an implicit field name
	goServiceBadShort  // syntactic := with no reliable declaration semantics
)

type goServiceOccurrence struct {
	id        *ast.Ident
	scope     *goServiceScope
	decl      *goServiceDecl
	role      goServiceRole
	inherited *goServiceConstProbe
}

func (o goServiceOccurrence) lookup(name string, probe *goServiceConstProbe, moved *goServiceDecl) (*goServiceDecl, bool) {
	return o.scope.lookup(name, o.id.Pos(), probe, moved)
}

type goServiceReturnSite struct {
	scope, results *goServiceScope
	pos            token.Pos
}

type goServiceBinding struct {
	run       *serviceRun
	work      *goServiceWork
	file      *token.File
	decls     []*goServiceDecl
	uses      []goServiceOccurrence // includes declaration names and excluded names
	top       map[*ast.Ident]*goServiceDecl
	returns   []goServiceReturnSite
	inherited *goServiceConstProbe
}

func (b *goServiceBinding) offsets(start, end token.Pos) goServiceRange {
	return goServiceRange{b.file.Offset(start), b.file.Offset(end)}
}

func (b *goServiceBinding) use(id *ast.Ident, scope *goServiceScope, role goServiceRole) {
	if id != nil {
		if !b.work.step() {
			return
		}
		if id.Name == "_" {
			role = goServiceExcluded
		}
		b.uses = append(b.uses, goServiceOccurrence{id: id, scope: scope, role: role, inherited: b.inherited})
	}
}

func (b *goServiceBinding) declare(id *ast.Ident, kind string, scope *goServiceScope, visible token.Pos, sig goServiceRange, doc *ast.CommentGroup, supported bool) *goServiceDecl {
	if !b.work.step() {
		return nil
	}
	if id.Name == "_" {
		b.use(id, scope, goServiceExcluded)
		return nil
	}
	if b.run.request.Operation == ServiceSymbols && !b.run.count(len(b.decls), b.run.limits.Symbols, serviceSymbolsResource) {
		return nil
	}
	d := &goServiceDecl{id: id, kind: kind, scope: scope, visible: visible, signature: sig, doc: doc, unsupported: !supported}
	if scope != nil {
		scope.enter(d)
	}
	b.decls = append(b.decls, d)
	b.uses = append(b.uses, goServiceOccurrence{id: id, scope: scope, decl: d})
	return d
}

func (b *goServiceBinding) at(offset int) *goServiceOccurrence {
	for i := range b.uses {
		if !b.work.step() {
			return nil
		}
		item := &b.uses[i]
		start := b.file.Offset(item.id.Pos())
		if start <= offset && offset < start+len(item.id.Name) {
			return item
		}
	}
	return nil
}

func (b *goServiceBinding) bind(file *ast.File) {
	pkg := newGoServiceScope(nil, false)
	pkg.work = b.work
	importBarrier := newGoServiceScope(pkg, false)
	localFile := newGoServiceScope(importBarrier, false)
	b.top = make(map[*ast.Ident]*goServiceDecl)
	b.use(file.Name, localFile, goServiceExcluded)
	// Package identifiers have whole-package visibility. Register only the
	// declarations actually present in this supplied file.
	for _, decl := range file.Decls {
		if !b.work.step() {
			return
		}
		switch n := decl.(type) {
		case *ast.FuncDecl:
			scope, kind := pkg, "func"
			if n.Recv != nil {
				scope, kind = nil, "method"
			} else if n.Name.Name == "init" {
				scope = nil
			}
			b.top[n.Name] = b.declare(n.Name, kind, scope, token.NoPos, b.offsets(n.Pos(), n.Type.End()), n.Doc, scope != nil)
		case *ast.GenDecl:
			for _, spec := range n.Specs {
				if !b.work.step() {
					return
				}
				switch item := spec.(type) {
				case *ast.ValueSpec:
					b.declareValues(item, n.Tok.String(), pkg, token.NoPos, n.Doc, true)
				case *ast.TypeSpec:
					b.top[item.Name] = b.declareType(item, pkg, token.NoPos, n.Doc)
				case *ast.ImportSpec:
					if item.Name == nil || item.Name.Name == "." {
						// The actual package name (and names exposed by a dot
						// import) require package loading, which this service never
						// performs. Keep lexical locals usable but stop lookup before
						// claiming a same-file package declaration is certain.
						importBarrier.opaque = true
					} else if item.Name.Name != "_" {
						b.declare(item.Name, "import", localFile, token.NoPos, b.offsets(item.Pos(), item.End()), item.Doc, false)
					}
				}
			}
		}
	}
	walker := goServiceWalk{b: b, scope: localFile}
	for _, decl := range file.Decls {
		if !b.work.step() {
			return
		}
		walker.walk(decl)
	}
}

func (b *goServiceBinding) declareValues(n *ast.ValueSpec, kind string, scope *goServiceScope, visible token.Pos, doc *ast.CommentGroup, top bool) {
	if n.Doc != nil {
		doc = n.Doc
	}
	end := n.Names[len(n.Names)-1].End()
	if n.Type != nil {
		end = n.Type.End()
	}
	sig := b.offsets(n.Pos(), end)
	for _, id := range n.Names {
		if !b.work.step() {
			return
		}
		d := b.declare(id, kind, scope, visible, sig, doc, true)
		if top {
			b.top[id] = d
		}
	}
}

func (b *goServiceBinding) declareType(n *ast.TypeSpec, scope *goServiceScope, visible token.Pos, doc *ast.CommentGroup) *goServiceDecl {
	if n.Doc != nil {
		doc = n.Doc
	}
	end := n.Name.End()
	if n.TypeParams != nil {
		end = n.TypeParams.End()
	}
	return b.declare(n.Name, "type", scope, visible, b.offsets(n.Name.Pos(), end), doc, true)
}

type goServiceWalk struct {
	b       *goServiceBinding
	scope   *goServiceScope
	results *goServiceScope
}

func (w goServiceWalk) child() goServiceWalk {
	w.scope = newGoServiceScope(w.scope, true)
	return w
}

func (w goServiceWalk) walk(n ast.Node) {
	if n != nil {
		ast.Walk(w, n)
	}
}

func (w goServiceWalk) expression(n ast.Expr) {
	if n != nil {
		w.walk(n)
	}
}

func (w goServiceWalk) statements(list []ast.Stmt) {
	for _, stmt := range list {
		if !w.b.work.step() {
			return
		}
		w.walk(stmt)
	}
}

func (w goServiceWalk) expressions(list []ast.Expr) {
	for _, expr := range list {
		if !w.b.work.step() {
			return
		}
		w.expression(expr)
	}
}

func (w goServiceWalk) body(block *ast.BlockStmt) {
	if block != nil {
		w.statements(block.List)
	}
}

func (w goServiceWalk) Visit(n ast.Node) ast.Visitor {
	if n == nil || !w.b.work.step() {
		return nil
	}
	switch n := n.(type) {
	case *ast.Ident:
		w.b.use(n, w.scope, goServiceUse)
	case *ast.SelectorExpr:
		w.expression(n.X)
		w.b.use(n.Sel, w.scope, goServiceExcluded)
	case *ast.CompositeLit:
		w.expression(n.Type)
		keyRole := goServiceMaybeKey
		switch n.Type.(type) {
		case *ast.StructType:
			keyRole = goServiceExcluded
		case *ast.MapType, *ast.ArrayType:
			keyRole = goServiceUse
		}
		for _, element := range n.Elts {
			if !w.b.work.step() {
				break
			}
			if pair, ok := element.(*ast.KeyValueExpr); ok {
				w.compositeEntry(pair, keyRole)
			} else {
				w.expression(element)
			}
		}
	case *ast.KeyValueExpr:
		w.compositeEntry(n, goServiceMaybeKey)
	case *ast.StructType:
		for _, field := range n.Fields.List {
			if !w.b.work.step() {
				break
			}
			if len(field.Names) != 0 {
				w.walk(field)
			} else {
				w.embeddedType(field.Type)
			}
		}
	case *ast.Field:
		for _, id := range n.Names {
			if !w.b.work.step() {
				break
			}
			w.b.use(id, w.scope, goServiceExcluded)
		}
		w.expression(n.Type)
	case *ast.BlockStmt:
		w.child().body(n)
	case *ast.IfStmt:
		inner := w.child()
		inner.walk(n.Init)
		inner.expression(n.Cond)
		inner.walk(n.Body)
		inner.walk(n.Else)
	case *ast.ForStmt:
		inner := w.child()
		inner.walk(n.Init)
		inner.expression(n.Cond)
		if post, ok := n.Post.(*ast.AssignStmt); ok && post.Tok == token.DEFINE {
			inner.invalidShort(post.Lhs)
			inner.expressions(post.Rhs)
		} else {
			inner.walk(n.Post)
		}
		inner.walk(n.Body)
	case *ast.RangeStmt:
		inner := w.child()
		inner.expression(n.X)
		if n.Tok == token.DEFINE {
			inner.short([]ast.Expr{n.Key, n.Value}, n.Body.Pos(), w.b.offsets(n.Pos(), n.TokPos), false)
		} else {
			inner.expression(n.Key)
			inner.expression(n.Value)
		}
		inner.walk(n.Body)
	case *ast.SwitchStmt:
		inner := w.child()
		inner.walk(n.Init)
		inner.expression(n.Tag)
		inner.body(n.Body) // each case establishes its own block
	case *ast.TypeSwitchStmt:
		w.typeSwitch(n)
	case *ast.CaseClause:
		w.expressions(n.List)
		w.child().statements(n.Body)
	case *ast.SelectStmt:
		w.body(n.Body)
	case *ast.CommClause:
		inner := w.child()
		inner.walk(n.Comm)
		inner.statements(n.Body)
	case *ast.LabeledStmt:
		w.b.use(n.Label, w.scope, goServiceExcluded)
		w.walk(n.Stmt)
	case *ast.BranchStmt:
		w.b.use(n.Label, w.scope, goServiceExcluded)
	case *ast.ReturnStmt:
		if w.results != nil && len(n.Results) == 0 && w.b.run.request.Operation == ServiceRename {
			w.b.returns = append(w.b.returns, goServiceReturnSite{w.scope, w.results, n.Return})
		}
		w.expressions(n.Results)
	case *ast.AssignStmt:
		if n.Tok != token.DEFINE {
			return w
		}
		w.expressions(n.Rhs)
		w.short(n.Lhs, n.End(), w.b.offsets(n.Pos(), n.TokPos), true)
	case *ast.FuncDecl:
		w.function(n.Recv, n.Type, n.Body, n.Name.End())
	case *ast.FuncLit:
		w.function(nil, n.Type, n.Body, n.Type.Pos())
	case *ast.GenDecl:
		w.genDecl(n)
	default:
		return w
	}
	return nil
}

func (w goServiceWalk) compositeEntry(n *ast.KeyValueExpr, keyRole goServiceRole) {
	if id, ok := n.Key.(*ast.Ident); ok {
		w.b.use(id, w.scope, keyRole)
	} else {
		w.expression(n.Key)
	}
	w.expression(n.Value)
}

func (w goServiceWalk) genDecl(n *ast.GenDecl) {
	for i, spec := range n.Specs {
		if !w.b.work.step() {
			return
		}
		switch nSpec := spec.(type) {
		case *ast.ValueSpec:
			_, predeclared := w.b.top[nSpec.Names[0]]
			if !predeclared {
				w.b.declareValues(nSpec, n.Tok.String(), w.scope, nSpec.End(), n.Doc, false)
			}
			previous := w.b.inherited
			if !predeclared && n.Tok == token.CONST && (nSpec.Type != nil || len(nSpec.Values) != 0) {
				if last := w.lastInheritedSpec(n.Specs[i+1:]); last.IsValid() {
					w.b.inherited = &goServiceConstProbe{previous, w.scope, last}
				}
			}
			w.expression(nSpec.Type)
			w.expressions(nSpec.Values)
			w.b.inherited = previous
		case *ast.TypeSpec:
			if _, predeclared := w.b.top[nSpec.Name]; !predeclared {
				w.b.declareType(nSpec, w.scope, nSpec.Name.Pos(), n.Doc)
			}
			if nSpec.TypeParams != nil {
				inner := w.child()
				inner.fields(nSpec.TypeParams, "type-parameter", nSpec.Name.End())
				inner.expression(nSpec.Type)
			} else {
				w.expression(nSpec.Type)
			}
		}
	}
}

func (w goServiceWalk) lastInheritedSpec(specs []ast.Spec) token.Pos {
	var last token.Pos
	for _, spec := range specs {
		if !w.b.work.step() {
			break
		}
		value, ok := spec.(*ast.ValueSpec)
		if !ok || value.Type != nil || len(value.Values) != 0 {
			break
		}
		last = spec.Pos()
	}
	return last
}

func (w goServiceWalk) fields(list *ast.FieldList, kind string, visible token.Pos) {
	if list == nil {
		return
	}
	for _, field := range list.List {
		if !w.b.work.step() {
			return
		}
		sig := w.b.offsets(field.Pos(), field.End())
		for _, id := range field.Names {
			if !w.b.work.step() {
				return
			}
			w.b.declare(id, kind, w.scope, visible, sig, field.Doc, true)
		}
		w.expression(field.Type)
	}
}

func (w goServiceWalk) function(recv *ast.FieldList, typ *ast.FuncType, body *ast.BlockStmt, typeVisible token.Pos) {
	inner := w.child()
	bodyStart := typ.End()
	if body != nil {
		bodyStart = body.Pos()
	}
	if recv != nil {
		inner.receiver(recv, typeVisible, bodyStart)
	}
	inner.fields(typ.TypeParams, "type-parameter", typeVisible)
	inner.fields(typ.Params, "parameter", bodyStart)
	inner.fields(typ.Results, "result", bodyStart)
	inner.results = nil // a closure does not inherit the outer function's results
	if typ.Results != nil {
		inner.results = inner.scope
	}
	inner.body(body) // parameters and the function's outer block share declarations
}

func (w goServiceWalk) receiver(list *ast.FieldList, typeVisible, bodyStart token.Pos) {
	if len(list.List) != 1 {
		w.scope.opaque = true
		w.fields(list, "receiver", bodyStart)
		return
	}
	field := list.List[0]
	if len(field.Names) > 1 {
		w.scope.opaque = true
	}
	for _, id := range field.Names {
		if !w.b.work.step() {
			return
		}
		w.b.declare(id, "receiver", w.scope, bodyStart, w.b.offsets(field.Pos(), field.End()), field.Doc, true)
	}
	base := ast.Unparen(field.Type)
	if star, ok := base.(*ast.StarExpr); ok {
		base = ast.Unparen(star.X)
	}
	var params []ast.Expr
	switch n := base.(type) {
	case *ast.IndexExpr:
		base, params = n.X, []ast.Expr{n.Index}
	case *ast.IndexListExpr:
		base, params = n.X, n.Indices
	}
	if _, ok := base.(*ast.Ident); !ok {
		w.scope.opaque = true
	}
	w.expression(base)
	for _, param := range params {
		if !w.b.work.step() {
			return
		}
		id, ok := param.(*ast.Ident)
		if !ok {
			w.scope.opaque = true
			w.expression(param)
			continue
		}
		w.b.declare(id, "type-parameter", w.scope, typeVisible, w.b.offsets(id.Pos(), id.End()), nil, true)
	}
}

func (w goServiceWalk) embeddedType(expr ast.Expr) {
	switch n := expr.(type) {
	case *ast.Ident:
		w.b.use(n, w.scope, goServiceEmbedded)
	case *ast.StarExpr:
		w.embeddedType(n.X)
	case *ast.ParenExpr:
		w.embeddedType(n.X)
	case *ast.IndexExpr:
		w.embeddedType(n.X)
		w.expression(n.Index)
	case *ast.IndexListExpr:
		w.embeddedType(n.X)
		w.expressions(n.Indices)
	default:
		w.expression(expr)
	}
}

func (w goServiceWalk) typeSwitch(n *ast.TypeSwitchStmt) {
	inner := w.child()
	inner.walk(n.Init)
	var alias *ast.Ident
	if assign, ok := n.Assign.(*ast.AssignStmt); ok && assign.Tok == token.DEFINE {
		if len(assign.Lhs) == 1 {
			alias, _ = assign.Lhs[0].(*ast.Ident)
		}
		if alias != nil {
			// The guard's spelling declares one implicit variable per case. It
			// can never be a use of an outer declaration of the same name.
			w.b.use(alias, inner.scope, goServiceExcluded)
		} else {
			inner.invalidShort(assign.Lhs)
		}
		inner.expressions(assign.Rhs)
	} else {
		inner.walk(n.Assign)
	}
	for _, entry := range n.Body.List {
		if !w.b.work.step() {
			return
		}
		clause, ok := entry.(*ast.CaseClause)
		if !ok {
			inner.walk(entry)
			continue
		}
		inner.expressions(clause.List)
		branch := inner.child()
		if alias != nil && alias.Name != "_" {
			branch.scope.enter(&goServiceDecl{id: alias, scope: branch.scope, visible: clause.Colon, unsupported: true})
		}
		branch.statements(clause.Body)
	}
}

func (w goServiceWalk) invalidShort(lhs []ast.Expr) {
	for _, expr := range lhs {
		if !w.b.work.step() {
			return
		}
		if id, ok := expr.(*ast.Ident); ok {
			w.b.use(id, w.scope, goServiceBadShort)
		} else {
			w.expression(expr)
		}
	}
}

func (w goServiceWalk) short(lhs []ast.Expr, visible token.Pos, sig goServiceRange, reassign bool) {
	seen := make(map[string]bool, len(lhs))
	hasNew := false
	for _, expr := range lhs {
		if !w.b.work.step() {
			return
		}
		if expr == nil { // range with only one iteration variable
			continue
		}
		id, ok := expr.(*ast.Ident)
		if !ok || (id.Name != "_" && seen[id.Name]) {
			w.invalidShort(lhs)
			return
		}
		if id.Name == "_" {
			continue
		}
		seen[id.Name] = true
		old := w.scope.decls[id.Name]
		if old == nil || old.visible > id.Pos() {
			hasNew = true
		} else if !reassign || !goServiceVariable(old.kind) {
			w.invalidShort(lhs)
			return
		}
	}
	if !hasNew {
		w.invalidShort(lhs)
		return
	}
	for _, expr := range lhs {
		if !w.b.work.step() {
			return
		}
		if expr == nil {
			continue
		}
		id := expr.(*ast.Ident)
		if old := w.scope.decls[id.Name]; old != nil && old.visible <= id.Pos() {
			w.b.use(id, w.scope, goServiceUse)
		} else {
			w.b.declare(id, "var", w.scope, visible, sig, nil, true)
		}
	}
}

func goServiceVariable(kind string) bool {
	switch kind {
	case "var", "parameter", "result", "receiver":
		return true
	}
	return false
}
