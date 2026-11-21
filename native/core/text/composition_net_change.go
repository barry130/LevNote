package text

import (
	"sort"
	"unicode/utf8"
)

// BaselineID identifies a settled observation baseline. It changes when Restore
// or an active-group ResetHistory invalidates that baseline, not on edits.
// Retaining an ID retains only a non-zero-sized identity marker, never text.
type BaselineID struct{ epoch *changeBaselineEpoch }
type changeBaselineEpoch struct{ marker byte }

// ConfirmedChangeID changes once per byte-changing settled transaction or replay.
// It is opaque and comparable; its separate epoch makes counter rollover safe.
type ConfirmedChangeID struct {
	epoch      *confirmedChangeEpoch
	generation uint64
}
type confirmedChangeEpoch struct{ marker byte }

// ChangeStamp is sampled together, on the Buffer owner's goroutine. Compare
// BaselineID first: a mismatch invalidates earlier work, it is not an edit.
// Only with the same baseline does a different ConfirmedChangeID mean a real
// confirmed byte change. Neither token may be projected to a numeric revision.
// This value is not a synchronization primitive for concurrent Buffer access.
type ChangeStamp struct {
	BaselineID        BaselineID
	ConfirmedChangeID ConfirmedChangeID
}

// ChangeStamp is pure: it does not allocate, scan source/history or mutate state.
func (b *Buffer) ChangeStamp() ChangeStamp { return b.changeStamp }

func newChangeStamp() ChangeStamp {
	return ChangeStamp{
		BaselineID:        BaselineID{&changeBaselineEpoch{marker: 1}},
		ConfirmedChangeID: ConfirmedChangeID{epoch: &confirmedChangeEpoch{marker: 1}},
	}
}

func (b *Buffer) invalidateChangeBaseline() {
	b.changeStamp.BaselineID = BaselineID{&changeBaselineEpoch{marker: 1}}
}

func (b *Buffer) advanceConfirmedChange() {
	id := &b.changeStamp.ConfirmedChangeID
	if id.generation == ^uint64(0) {
		id.epoch = &confirmedChangeEpoch{marker: 1}
		id.generation = 0
	} else {
		id.generation++
	}
}

// netChangeWork records algorithm work for invariant/resource regressions. A nil
// pointer (normal commit) collects nothing. These are counts, not timing claims.
type netChangeWork struct {
	nodes, splitVisits, joinVisits           int
	indexBuildBytes, nextScanBytes           int
	oldLookups, oldScanBytes, currentLookups int
	comparedBytes, skippedCurrentBytes       int
}

type netCheckpoint struct{ units, bytes int }
type netOldText struct {
	value  string
	units  int
	points []netCheckpoint
}

// Index BEFORE the next rune would cross the 4096-byte window. Every point is
// both a UTF-8 boundary and a complete UTF-16 boundary. There is one index per
// inserted old-string occurrence, never one per subsequent split of that text.
func indexNetOld(value string, work *netChangeWork) *netOldText {
	x := &netOldText{value: value, points: []netCheckpoint{{}}}
	last := 0
	for at := 0; at < len(value); {
		r, size := utf8.DecodeRuneInString(value[at:])
		if r == utf8.RuneError && size == 1 {
			panic("text: invalid UTF-8 in retained transaction")
		}
		if size > chunkBytes-(at-last) {
			x.points = append(x.points, netCheckpoint{x.units, at})
			last = at
		}
		width := 1
		if r > 0xffff {
			width = 2
		}
		x.units = netCheckedSum(x.units, width)
		at += size
		if work != nil {
			work.indexBuildBytes += size
		}
	}
	return x
}

func (x *netOldText) byteOffset(units int, work *netChangeWork) int {
	if units < 0 || units > x.units {
		panic("text: retained transaction unit offset outside old string")
	}
	if work != nil {
		work.oldLookups++
	}
	if units == x.units {
		return len(x.value)
	}
	i := sort.Search(len(x.points), func(i int) bool { return x.points[i].units > units }) - 1
	p := x.points[i]
	at, left := p.bytes, units-p.units
	for left > 0 {
		r, size := utf8.DecodeRuneInString(x.value[at:])
		width := 1
		if r > 0xffff {
			width = 2
		}
		if left < width {
			panic("text: retained transaction splits a UTF-16 surrogate pair")
		}
		left -= width
		at += size
		if work != nil {
			work.oldScanBytes += size
		}
	}
	return at
}

// old==nil denotes a lazy absolute span of the immutable CURRENT buffer. It
// may cover arbitrarily many pieces without copying that piece list. Otherwise
// bounds address one indexed old-string occurrence. Surviving fragments of each
// source retain increasing, disjoint bounds: inverse splices never reorder or
// duplicate existing fragments, only split/delete them and insert a new source.
type netFragment struct {
	old                *netOldText
	byteStart, byteEnd int
	unitStart, units   int
}

// Each node owns one fragment; split/join rewire unique scratch nodes in place.
// No path copies are made. Two splits and one old insertion allocate at most
// three nodes per inverse edit, so live AND transient node allocation is O(E).
type netNode struct {
	left, right          *netNode
	fragment             netFragment
	units, bytes, height int
}

type netChangeContext struct {
	buffer *Buffer
	work   *netChangeWork
}

func netCheckedSum(a, z int) int {
	const maxInt = int(^uint(0) >> 1)
	if a < 0 || z < 0 || z > maxInt-a {
		panic("text: retained transaction weight overflow")
	}
	return a + z
}

func netUnits(n *netNode) int {
	if n == nil {
		return 0
	}
	return n.units
}
func netBytes(n *netNode) int {
	if n == nil {
		return 0
	}
	return n.bytes
}
func netHeight(n *netNode) int {
	if n == nil {
		return 0
	}
	return n.height
}
func (n *netNode) recount() {
	own := n.fragment.byteEnd - n.fragment.byteStart
	if n.fragment.byteStart < 0 || n.fragment.unitStart < 0 || own <= 0 || n.fragment.units <= 0 || n.fragment.units > own {
		panic("text: invalid virtual transaction fragment")
	}
	n.units = netCheckedSum(netCheckedSum(netUnits(n.left), n.fragment.units), netUnits(n.right))
	n.bytes = netCheckedSum(netCheckedSum(netBytes(n.left), own), netBytes(n.right))
	n.height = 1 + max(netHeight(n.left), netHeight(n.right))
}
func (c *netChangeContext) node(f netFragment) *netNode {
	if f.byteEnd == f.byteStart && f.units == 0 {
		return nil
	}
	n := &netNode{fragment: f}
	n.recount()
	if c.work != nil {
		c.work.nodes++
	}
	return n
}
func netRotateLeft(n *netNode) *netNode {
	r := n.right
	n.right, r.left = r.left, n
	n.recount()
	r.recount()
	return r
}
func netRotateRight(n *netNode) *netNode {
	l := n.left
	n.left, l.right = l.right, n
	n.recount()
	l.recount()
	return l
}
func netBalance(n *netNode) *netNode {
	n.recount()
	if netHeight(n.left)-netHeight(n.right) > 1 {
		if netHeight(n.left.left) < netHeight(n.left.right) {
			n.left = netRotateLeft(n.left)
		}
		return netRotateRight(n)
	}
	if netHeight(n.right)-netHeight(n.left) > 1 {
		if netHeight(n.right.right) < netHeight(n.right.left) {
			n.right = netRotateRight(n.right)
		}
		return netRotateLeft(n)
	}
	return n
}

// Join arbitrary-height AVL sides around a detached, existing pivot node.
func (c *netChangeContext) joinPivot(left, pivot, right *netNode) *netNode {
	if c.work != nil {
		c.work.joinVisits++
	}
	if netHeight(left) > netHeight(right)+1 {
		left.right = c.joinPivot(left.right, pivot, right)
		return netBalance(left)
	}
	if netHeight(right) > netHeight(left)+1 {
		right.left = c.joinPivot(left, pivot, right.left)
		return netBalance(right)
	}
	pivot.left, pivot.right = left, right
	pivot.recount()
	return pivot
}
func (c *netChangeContext) popFirst(n *netNode) (*netNode, *netNode) {
	if n.left == nil {
		right := n.right
		n.left, n.right = nil, nil
		n.recount()
		return right, n
	}
	var first *netNode
	n.left, first = c.popFirst(n.left)
	return netBalance(n), first
}
func (c *netChangeContext) join(left, right *netNode) *netNode {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	rest, pivot := c.popFirst(right)
	return c.joinPivot(left, pivot, rest)
}

func (c *netChangeContext) splitFragment(f netFragment, units int) (netFragment, netFragment) {
	if units <= 0 || units >= f.units {
		panic("text: invalid internal virtual-fragment split")
	}
	absolute := netCheckedSum(f.unitStart, units)
	var at int
	if f.old == nil {
		if c.work != nil {
			c.work.currentLookups++
		}
		var err error
		at, err = c.buffer.byteOffset(absolute)
		if err != nil {
			panic("text: invalid current-source transaction boundary")
		}
	} else {
		at = f.old.byteOffset(absolute, c.work)
	}
	if at <= f.byteStart || at >= f.byteEnd {
		panic("text: inconsistent virtual byte/unit boundary")
	}
	left, right := f, f
	left.byteEnd, left.units = at, units
	right.byteStart, right.unitStart, right.units = at, absolute, f.units-units
	return left, right
}

func (c *netChangeContext) split(n *netNode, units int) (*netNode, *netNode) {
	if units < 0 || units > netUnits(n) {
		panic("text: retained transaction split outside intermediate state")
	}
	if units == 0 {
		return nil, n
	}
	if units == netUnits(n) {
		return n, nil
	}
	if c.work != nil {
		c.work.splitVisits++
	}
	left, right := n.left, n.right
	before, through := netUnits(left), netCheckedSum(netUnits(left), n.fragment.units)
	n.left, n.right = nil, nil
	switch {
	case units < before:
		a, z := c.split(left, units)
		return a, c.joinPivot(z, n, right)
	case units > through:
		a, z := c.split(right, units-through)
		return c.joinPivot(left, n, a), z
	case units == before:
		return left, c.joinPivot(nil, n, right)
	case units == through:
		return c.joinPivot(left, n, nil), right
	default:
		a, z := c.splitFragment(n.fragment, units-before)
		n.fragment = a
		other := c.node(z)
		return c.joinPivot(left, n, nil), c.joinPivot(nil, other, right)
	}
}

// Inverse edits use their sequential UTF-16 coordinates in reverse, exactly as
// Undo does. They are NOT sorted/remapped in the final document's coordinates.
func (c *netChangeContext) inverse(t transaction) *netNode {
	root := c.node(netFragment{byteEnd: c.buffer.bytes, units: c.buffer.units})
	for i := len(t.edits) - 1; i >= 0; i-- {
		edit := t.edits[i]
		left, rest := c.split(root, edit.start)
		nextUnits := UTF16Len(edit.next)
		if c.work != nil {
			c.work.nextScanBytes += len(edit.next)
		}
		_, right := c.split(rest, nextUnits)
		var old *netNode
		if edit.old != "" {
			source := indexNetOld(edit.old, c.work)
			old = c.node(netFragment{old: source, byteEnd: len(edit.old), units: source.units})
		}
		root = c.join(c.join(left, old), right)
		if netBytes(root) > c.buffer.maxBytes {
			panic("text: retained transaction exceeds admitted intermediate ceiling")
		}
	}
	return root
}

type netFragmentIterator struct {
	node  *netNode
	stack []*netNode
}

func (it *netFragmentIterator) next() (netFragment, bool) {
	for it.node != nil {
		it.stack = append(it.stack, it.node)
		it.node = it.node.left
	}
	if len(it.stack) == 0 {
		return netFragment{}, false
	}
	n := it.stack[len(it.stack)-1]
	it.stack = it.stack[:len(it.stack)-1]
	it.node = n.right
	return n.fragment, true
}

func (c *netChangeContext) pieceAtByte(offset int) int {
	return sort.Search(len(c.buffer.pieces), func(i int) bool { return c.buffer.prefix[i+1].bytes > offset })
}

// Compare fragments directly with current immutable pieces. Each post cursor
// moves forward once. Surviving current-source bounds are also monotone; strings
// are referenced, never joined. Only identical absolute current spans may skip.
func (c *netChangeContext) equalCurrent(root *netNode) bool {
	if netBytes(root) != c.buffer.bytes {
		return false
	}
	it := netFragmentIterator{node: root}
	post := 0
	for {
		f, ok := it.next()
		if !ok {
			return post == c.buffer.bytes
		}
		if f.old == nil && f.byteStart == post {
			n := f.byteEnd - f.byteStart
			post += n
			if c.work != nil {
				c.work.skippedCurrentBytes += n
			}
			continue
		}
		source := f.byteStart
		i, j := 0, c.pieceAtByte(post)
		if f.old == nil {
			i = c.pieceAtByte(source)
		}
		for source < f.byteEnd {
			var x string
			if f.old != nil {
				x = f.old.value[source:f.byteEnd]
			} else {
				p := c.buffer.pieces[i]
				at := p.start + (source - c.buffer.prefix[i].bytes)
				n := min(p.end-at, f.byteEnd-source)
				x = p.data.value[at : at+n]
			}
			q := c.buffer.pieces[j]
			at := q.start + (post - c.buffer.prefix[j].bytes)
			n := min(len(x), q.end-at)
			if c.work != nil {
				c.work.comparedBytes += n
			}
			if x[:n] != q.data.value[at:at+n] {
				return false
			}
			source, post = source+n, post+n
			if post == c.buffer.prefix[j+1].bytes {
				j++
			}
			if f.old == nil && source == c.buffer.prefix[i+1].bytes {
				i++
			}
		}
	}
}

// transactionChangesBytes classifies only this settling transaction, never older
// history. Its temporary sequence/index sizes depend on actual retained edits,
// not a fabricated constant cap. Invalid internal coordinates fail explicitly.
// Ordinary single/irregular net-zero history semantics are otherwise unchanged.
func (b *Buffer) transactionChangesBytes(t transaction, work *netChangeWork) bool {
	if len(t.edits) == 0 {
		return false
	}
	if len(t.edits) == 1 {
		return t.edits[0].old != t.edits[0].next
	}
	c := netChangeContext{buffer: b, work: work}
	return !c.equalCurrent(c.inverse(t))
}
