package enginescene

import "strconv"

// Encode validates a scene and emits deterministic JSON with one exact-sized
// output buffer. Every failure returns nil, never partial bytes. The payload
// cap does not bound the caller's scene or total process memory.
func Encode(s Scene) ([]byte, error) {
	if err := validate(s); err != nil {
		return nil, err
	}
	count := sceneWriter{limit: MaxPayloadBytes}
	emitScene(&count, s)
	if count.err != nil {
		return nil, count.err
	}
	fill := sceneWriter{limit: count.written, data: make([]byte, count.written)}
	emitScene(&fill, s)
	if fill.err != nil {
		return nil, fill.err
	}
	if fill.written != len(fill.data) {
		return nil, ErrInvalidScene
	}
	return fill.data, nil
}

type sceneWriter struct {
	data           []byte
	limit, written int
	err            error
}

func (w *sceneWriter) text(value string) {
	if w.err != nil {
		return
	}
	if len(value) > w.limit-w.written {
		w.err = ErrLimitExceeded
		return
	}
	if w.data != nil {
		copy(w.data[w.written:], value)
	}
	w.written += len(value)
}

func (w *sceneWriter) quoted(value string) {
	w.text(`"`)
	start := 0
	for i := 0; i < len(value) && w.err == nil; i++ {
		c := value[i]
		if c >= 0x20 && c != '"' && c != '\\' {
			continue
		}
		w.text(value[start:i])
		switch c {
		case '"':
			w.text(`\"`)
		case '\\':
			w.text(`\\`)
		case '\b':
			w.text(`\b`)
		case '\f':
			w.text(`\f`)
		case '\n':
			w.text(`\n`)
		case '\r':
			w.text(`\r`)
		case '\t':
			w.text(`\t`)
		default:
			const hex = "0123456789abcdef"
			var escaped = [6]byte{'\\', 'u', '0', '0', hex[c>>4], hex[c&15]}
			w.text(string(escaped[:]))
		}
		start = i + 1
	}
	w.text(value[start:])
	w.text(`"`)
}

func (w *sceneWriter) number(value float64) {
	var buf [32]byte
	w.text(string(strconv.AppendFloat(buf[:0], value, 'g', -1, 64)))
}

func (w *sceneWriter) integer(value int) {
	var buf [24]byte
	w.text(string(strconv.AppendInt(buf[:0], int64(value), 10)))
}

func (w *sceneWriter) boolean(value bool) {
	if value {
		w.text("true")
	} else {
		w.text("false")
	}
}

func (w *sceneWriter) point(value Point) {
	w.text(`{"x":`)
	w.number(value.X)
	w.text(`,"y":`)
	w.number(value.Y)
	w.text(`}`)
}

func (w *sceneWriter) rect(value Rect) {
	w.text(`{"x":`)
	w.number(value.X)
	w.text(`,"y":`)
	w.number(value.Y)
	w.text(`,"width":`)
	w.number(value.Width)
	w.text(`,"height":`)
	w.number(value.Height)
	w.text(`}`)
}

func emitScene(w *sceneWriter, s Scene) {
	w.text(`{"schema":`)
	w.integer(s.Schema)
	w.text(`,"layout":`)
	w.quoted(s.Layout)
	w.text(`,"nodeCount":`)
	w.integer(len(s.Nodes))
	w.text(`,"edgeCount":`)
	w.integer(len(s.Edges))
	w.text(`,"bounds":`)
	w.rect(s.Bounds)
	w.text(`,"nodes":[`)
	for i, node := range s.Nodes {
		if w.err != nil {
			return
		}
		if i > 0 {
			w.text(",")
		}
		w.text(`{"id":`)
		w.quoted(node.ID)
		w.text(`,"label":`)
		w.quoted(node.Label)
		w.text(`,"x":`)
		w.number(node.X)
		w.text(`,"y":`)
		w.number(node.Y)
		w.text(`,"width":`)
		w.number(node.Width)
		w.text(`,"height":`)
		w.number(node.Height)
		w.text(`,"shape":`)
		w.quoted(node.Shape)
		w.text(`,"fill":`)
		w.quoted(node.Fill)
		w.text(`,"stroke":`)
		w.quoted(node.Stroke)
		w.text(`,"text":`)
		w.quoted(node.Text)
		w.text(`,"root":`)
		w.boolean(node.Root)
		w.text(`,"collapsed":`)
		w.boolean(node.Collapsed)
		w.text(`}`)
	}
	w.text(`],"edges":[`)
	for i, edge := range s.Edges {
		if w.err != nil {
			return
		}
		if i > 0 {
			w.text(",")
		}
		w.text(`{"from":`)
		w.quoted(edge.FromID)
		w.text(`,"to":`)
		w.quoted(edge.ToID)
		w.text(`,"fromPoint":`)
		w.point(edge.From)
		w.text(`,"toPoint":`)
		w.point(edge.To)
		w.text(`,"stroke":`)
		w.quoted(edge.Stroke)
		w.text(`}`)
	}
	w.text(`]}`)
}
