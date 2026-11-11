package enginescene

import (
	"math"
	"strconv"
	"unicode/utf8"
)

// Decode admits only the fixed shallow schema, then validates its rooted tree.
// Strings are owned copies. No partial scene escapes on any failure.
func Decode(data []byte) (Scene, error) {
	reader := sceneReader{data: data}
	if err := reader.read(); err != nil {
		return Scene{}, err
	}
	return reader.scene, nil
}

type sceneReader struct {
	data  []byte
	pos   int
	scene Scene
}

var sceneKeys = [...]string{"schema", "layout", "nodeCount", "edgeCount", "bounds", "nodes", "edges"}
var nodeKeys = [...]string{"id", "label", "x", "y", "width", "height", "shape", "fill", "stroke", "text", "root", "collapsed"}
var edgeKeys = [...]string{"from", "to", "fromPoint", "toPoint", "stroke"}
var rectKeys = [...]string{"x", "y", "width", "height"}
var pointKeys = [...]string{"x", "y"}

func (p *sceneReader) read() error {
	if len(p.data) > MaxPayloadBytes {
		return ErrLimitExceeded
	}
	var nodes, edges int
	err := p.object(sceneKeys[:], func(key int) (err error) {
		switch key {
		case 0:
			var number decimalNumber
			number, err = p.number()
			if err == nil {
				var schema int
				schema, err = number.integer(SchemaVersion)
				if err != nil || schema != SchemaVersion {
					return ErrInvalidScene
				}
				p.scene.Schema = schema
			}
		case 1:
			p.scene.Layout, err = p.text(MaxLayoutBytes)
		case 2:
			nodes, err = p.count(MaxNodes)
		case 3:
			edges, err = p.count(MaxEdges)
		case 4:
			p.scene.Bounds, err = p.rect()
		case 5:
			err = p.nodes()
		case 6:
			err = p.edges()
		}
		return err
	})
	if err != nil {
		return err
	}
	p.space()
	if p.pos != len(p.data) || nodes != len(p.scene.Nodes) || edges != len(p.scene.Edges) {
		return ErrInvalidScene
	}
	return validate(p.scene)
}

func (p *sceneReader) space() {
	for p.pos < len(p.data) {
		switch p.data[p.pos] {
		case ' ', '\t', '\r', '\n':
			p.pos++
		default:
			return
		}
	}
}

func (p *sceneReader) take(c byte) bool {
	p.space()
	if p.pos == len(p.data) || p.data[p.pos] != c {
		return false
	}
	p.pos++
	return true
}

// The callers pass only fixed key arrays and fixed-schema field readers.
// Unexpected nesting is refused immediately; no arbitrary JSON subtree exists.
func (p *sceneReader) object(keys []string, field func(int) error) error {
	if !p.take('{') {
		return ErrInvalidScene
	}
	var seen uint16
	if !p.take('}') {
		for {
			key, err := p.text(9) // longest exact schema key is nine ASCII bytes
			if err != nil {
				return err
			}
			index := -1
			for i, expected := range keys {
				if key == expected {
					index = i
					break
				}
			}
			if index == -1 || seen&(1<<index) != 0 || !p.take(':') {
				return ErrInvalidScene
			}
			seen |= 1 << index
			if err := field(index); err != nil {
				return err
			}
			if p.take('}') {
				break
			}
			if !p.take(',') {
				return ErrInvalidScene
			}
		}
	}
	if seen != (1<<len(keys))-1 {
		return ErrInvalidScene
	}
	return nil
}

func (p *sceneReader) nodes() error {
	if !p.take('[') {
		return ErrInvalidScene
	}
	p.scene.Nodes = make([]Node, 0)
	if p.take(']') {
		return nil
	}
	for {
		if len(p.scene.Nodes) == MaxNodes {
			return ErrLimitExceeded
		}
		node, err := p.node()
		if err != nil {
			return err
		}
		if len(p.scene.Nodes) == cap(p.scene.Nodes) {
			capacity := min(MaxNodes, max(8, cap(p.scene.Nodes)*2))
			grown := make([]Node, len(p.scene.Nodes), capacity)
			copy(grown, p.scene.Nodes)
			p.scene.Nodes = grown
		}
		p.scene.Nodes = append(p.scene.Nodes, node)
		if p.take(']') {
			return nil
		}
		if !p.take(',') {
			return ErrInvalidScene
		}
	}
}

func (p *sceneReader) edges() error {
	if !p.take('[') {
		return ErrInvalidScene
	}
	p.scene.Edges = make([]Edge, 0)
	if p.take(']') {
		return nil
	}
	for {
		if len(p.scene.Edges) == MaxEdges {
			return ErrLimitExceeded
		}
		edge, err := p.edge()
		if err != nil {
			return err
		}
		if len(p.scene.Edges) == cap(p.scene.Edges) {
			capacity := min(MaxEdges, max(8, cap(p.scene.Edges)*2))
			grown := make([]Edge, len(p.scene.Edges), capacity)
			copy(grown, p.scene.Edges)
			p.scene.Edges = grown
		}
		p.scene.Edges = append(p.scene.Edges, edge)
		if p.take(']') {
			return nil
		}
		if !p.take(',') {
			return ErrInvalidScene
		}
	}
}

func (p *sceneReader) node() (value Node, err error) {
	err = p.object(nodeKeys[:], func(key int) (err error) {
		switch key {
		case 0:
			value.ID, err = p.text(MaxIDBytes)
			if err == nil && value.ID == "" {
				err = ErrInvalidScene
			}
		case 1:
			value.Label, err = p.text(MaxLabelBytes)
		case 2:
			value.X, err = p.coordinate(false)
		case 3:
			value.Y, err = p.coordinate(false)
		case 4:
			value.Width, err = p.coordinate(true)
		case 5:
			value.Height, err = p.coordinate(true)
		case 6:
			value.Shape, err = p.text(7)
			if err == nil && value.Shape != "rounded" {
				err = ErrInvalidScene
			}
		case 7:
			value.Fill, err = p.paint()
		case 8:
			value.Stroke, err = p.paint()
		case 9:
			value.Text, err = p.paint()
		case 10:
			value.Root, err = p.boolean()
		case 11:
			value.Collapsed, err = p.boolean()
		}
		return err
	})
	return value, err
}

func (p *sceneReader) edge() (value Edge, err error) {
	err = p.object(edgeKeys[:], func(key int) (err error) {
		switch key {
		case 0, 1:
			var id string
			id, err = p.text(MaxIDBytes)
			if err == nil && id == "" {
				err = ErrInvalidScene
			}
			if key == 0 {
				value.FromID = id
			} else {
				value.ToID = id
			}
		case 2:
			value.From, err = p.point()
		case 3:
			value.To, err = p.point()
		case 4:
			value.Stroke, err = p.paint()
		}
		return err
	})
	return value, err
}

func (p *sceneReader) rect() (value Rect, err error) {
	fields := [...]*float64{&value.X, &value.Y, &value.Width, &value.Height}
	err = p.object(rectKeys[:], func(key int) (err error) {
		*fields[key], err = p.coordinate(key >= 2)
		return err
	})
	return value, err
}

func (p *sceneReader) point() (value Point, err error) {
	fields := [...]*float64{&value.X, &value.Y}
	err = p.object(pointKeys[:], func(key int) (err error) {
		*fields[key], err = p.coordinate(false)
		return err
	})
	return value, err
}

func (p *sceneReader) paint() (string, error) {
	value, err := p.text(9)
	if err == nil {
		err = validPaint(value)
	}
	return value, err
}

func (p *sceneReader) boolean() (bool, error) {
	p.space()
	if len(p.data)-p.pos >= 4 && string(p.data[p.pos:p.pos+4]) == "true" {
		p.pos += 4
		return true, nil
	}
	if len(p.data)-p.pos >= 5 && string(p.data[p.pos:p.pos+5]) == "false" {
		p.pos += 5
		return false, nil
	}
	return false, ErrInvalidScene
}

// A fixed scratch buffer admits decoded bytes before copying. In particular,
// long raw/escaped tokens and unknown keys never create token-sized strings.
func (p *sceneReader) text(limit int) (string, error) {
	if !p.take('"') {
		return "", ErrInvalidScene
	}
	var buffer [MaxLabelBytes]byte
	written := 0
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		p.pos++
		if c == '"' {
			return string(buffer[:written]), nil
		}
		var value rune
		switch {
		case c == '\\':
			if p.pos == len(p.data) {
				return "", ErrInvalidScene
			}
			escaped := p.data[p.pos]
			p.pos++
			switch escaped {
			case '"', '\\', '/':
				value = rune(escaped)
			case 'b':
				value = '\b'
			case 'f':
				value = '\f'
			case 'n':
				value = '\n'
			case 'r':
				value = '\r'
			case 't':
				value = '\t'
			case 'u':
				var err error
				value, err = p.quad()
				if err != nil || value >= 0xdc00 && value <= 0xdfff {
					return "", ErrInvalidScene
				}
				if value >= 0xd800 && value <= 0xdbff {
					if len(p.data)-p.pos < 2 || p.data[p.pos] != '\\' || p.data[p.pos+1] != 'u' {
						return "", ErrInvalidScene
					}
					p.pos += 2
					low, err := p.quad()
					if err != nil || low < 0xdc00 || low > 0xdfff {
						return "", ErrInvalidScene
					}
					value = 0x10000 + (value-0xd800)*0x400 + low - 0xdc00
				}
			default:
				return "", ErrInvalidScene
			}
		case c < 0x20:
			return "", ErrInvalidScene
		case c < utf8.RuneSelf:
			value = rune(c)
		default:
			var size int
			value, size = utf8.DecodeRune(p.data[p.pos-1:])
			if value == utf8.RuneError && size == 1 {
				return "", ErrInvalidScene
			}
			p.pos += size - 1
		}
		size := utf8.RuneLen(value)
		if size > limit-written {
			return "", ErrLimitExceeded
		}
		written += utf8.EncodeRune(buffer[written:], value)
	}
	return "", ErrInvalidScene
}

func (p *sceneReader) quad() (rune, error) {
	if len(p.data)-p.pos < 4 {
		return 0, ErrInvalidScene
	}
	var value rune
	for end := p.pos + 4; p.pos < end; p.pos++ {
		c := p.data[p.pos]
		value <<= 4
		switch {
		case c >= '0' && c <= '9':
			value |= rune(c - '0')
		case c >= 'a' && c <= 'f':
			value |= rune(c-'a') + 10
		case c >= 'A' && c <= 'F':
			value |= rune(c-'A') + 10
		default:
			return 0, ErrInvalidScene
		}
	}
	return value, nil
}

// 800 significant digits exceed the precision needed to distinguish every
// binary64 rounding midpoint (including subnormals). A sticky nonzero digit
// preserves the side of a midpoint; strconv's bounded decimal fallback uses
// the same 800-digit precision. Never pass the untrusted whole token to it.
type decimalNumber struct {
	negative              bool
	digits                [800]byte
	kept, total, fraction int
	first, last, exponent int
	sticky                bool
}

func (n *decimalNumber) digit(c byte) {
	if c != '0' {
		if n.first == -1 {
			n.first = n.total
		}
		n.last = n.total
	}
	if n.first >= 0 {
		if n.kept < len(n.digits) {
			n.digits[n.kept] = c
			n.kept++
		} else if c != '0' {
			n.sticky = true
		}
	}
	n.total++
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func (p *sceneReader) number() (decimalNumber, error) {
	n := decimalNumber{first: -1, last: -1}
	p.space()
	if p.pos < len(p.data) && p.data[p.pos] == '-' {
		n.negative = true
		p.pos++
	}
	if p.pos == len(p.data) || !isDigit(p.data[p.pos]) {
		return n, ErrInvalidScene
	}
	if p.data[p.pos] == '0' {
		n.digit('0')
		p.pos++
	} else {
		for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
			n.digit(p.data[p.pos])
			p.pos++
		}
	}
	if p.pos < len(p.data) && p.data[p.pos] == '.' {
		p.pos++
		start := n.total
		for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
			n.digit(p.data[p.pos])
			p.pos++
		}
		n.fraction = n.total - start
		if n.fraction == 0 {
			return n, ErrInvalidScene
		}
	}
	if p.pos < len(p.data) && (p.data[p.pos] == 'e' || p.data[p.pos] == 'E') {
		p.pos++
		negative := false
		if p.pos < len(p.data) && (p.data[p.pos] == '+' || p.data[p.pos] == '-') {
			negative = p.data[p.pos] == '-'
			p.pos++
		}
		start := p.pos
		// Larger magnitudes cannot be canceled by any coefficient fitting
		// the payload. Saturate arithmetic, but still scan all JSON digits.
		const exponentLimit = MaxPayloadBytes + 1024
		for p.pos < len(p.data) && isDigit(p.data[p.pos]) {
			if n.exponent < exponentLimit {
				n.exponent = min(exponentLimit, n.exponent*10+int(p.data[p.pos]-'0'))
			}
			p.pos++
		}
		if start == p.pos {
			return n, ErrInvalidScene
		}
		if negative {
			n.exponent = -n.exponent
		}
	}
	if p.pos < len(p.data) {
		switch p.data[p.pos] {
		case ' ', '\t', '\r', '\n', ',', ']', '}':
		default:
			return n, ErrInvalidScene
		}
	}
	return n, nil
}

func (n *decimalNumber) order() int { return n.total - n.fraction + n.exponent - n.first }

func (n *decimalNumber) integer(limit int) (int, error) {
	if n.first == -1 {
		return 0, nil
	}
	// Compare exact decimal positions, never a rounded float64.
	point := n.total - n.fraction + n.exponent
	if n.negative || n.last >= point {
		return 0, ErrInvalidScene
	}
	order := n.order()
	if order > 6 { // every count limit is at most 100000
		return 0, ErrLimitExceeded
	}
	value := 0
	for i := 0; i < order; i++ {
		digit := 0
		if i < n.kept {
			digit = int(n.digits[i] - '0')
		}
		if value > (limit-digit)/10 || digit > limit {
			return 0, ErrLimitExceeded
		}
		value = value*10 + digit
	}
	return value, nil
}

func (p *sceneReader) count(limit int) (int, error) {
	n, err := p.number()
	if err != nil {
		return 0, err
	}
	return n.integer(limit)
}

func (p *sceneReader) coordinate(dimension bool) (float64, error) {
	n, err := p.number()
	if err != nil {
		return 0, err
	}
	if n.first == -1 {
		if dimension {
			return 0, ErrInvalidScene
		}
		if n.negative {
			return math.Copysign(0, -1), nil
		}
		return 0, nil
	}
	if dimension && n.negative {
		return 0, ErrInvalidScene
	}
	maxOrder := 10 // 1e9 coordinate, or 1e7 dimension
	if dimension {
		maxOrder = 8
	}
	order := n.order()
	if order > maxOrder || order == maxOrder && (n.digits[0] != '1' || n.last != n.first) {
		return 0, ErrLimitExceeded
	}
	var buffer [840]byte
	text := buffer[:0]
	if n.negative {
		text = append(text, '-')
	}
	text = append(text, n.digits[0], '.')
	text = append(text, n.digits[1:n.kept]...)
	if n.sticky {
		text = append(text, '1')
	}
	text = append(text, 'e')
	text = strconv.AppendInt(text, int64(order-1), 10)
	value, err := strconv.ParseFloat(string(text), 64)
	if err != nil {
		return 0, ErrInvalidScene
	}
	if err := validCoordinate(value, dimension); err != nil {
		return 0, err
	}
	return value, nil
}
