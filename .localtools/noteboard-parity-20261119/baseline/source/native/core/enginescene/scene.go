// Package enginescene defines a bounded, format-neutral native preview tree.
// It owns no renderer, source document, file, process or application identity.
package enginescene

import (
	"errors"
	"math"
	"unicode/utf8"
)

const (
	SchemaVersion    = 1
	MaxPayloadBytes  = 32 << 20
	MaxNodes         = 100_000
	MaxEdges         = 99_999
	MaxIDBytes       = 512
	MaxLabelBytes    = 515
	MaxLayoutBytes   = 64
	MaxCoordinateAbs = 1_000_000_000
	MaxDimension     = 10_000_000
)

var (
	ErrInvalidScene  = errors.New("invalid preview scene")
	ErrLimitExceeded = errors.New("preview scene exceeds limit")
)

type Point struct{ X, Y float64 }
type Rect struct{ X, Y, Width, Height float64 }
type Node struct {
	ID, Label           string
	X, Y, Width, Height float64
	Shape               string
	Fill, Stroke, Text  string
	Root, Collapsed     bool
}
type Edge struct {
	FromID, ToID string
	From, To     Point
	Stroke       string
}
type Scene struct {
	Schema int
	Layout string
	Bounds Rect
	Nodes  []Node
	Edges  []Edge
}

func validString(value string, limit int, nonempty bool) error {
	if len(value) > limit {
		return ErrLimitExceeded
	}
	if nonempty && value == "" || !utf8.ValidString(value) {
		return ErrInvalidScene
	}
	return nil
}

func validPaint(value string) error {
	if len(value) > 9 {
		return ErrLimitExceeded
	}
	if value == "" {
		return nil
	}
	if len(value) != 9 || value[0] != '#' {
		return ErrInvalidScene
	}
	for i := 1; i < len(value); i++ {
		c := value[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return ErrInvalidScene
		}
	}
	return nil
}

func validCoordinate(value float64, dimension bool) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || dimension && value <= 0 {
		return ErrInvalidScene
	}
	limit := float64(MaxCoordinateAbs)
	if dimension {
		limit = MaxDimension
	}
	if math.Abs(value) > limit {
		return ErrLimitExceeded
	}
	return nil
}

func validRect(value Rect) error {
	for i, number := range [...]float64{value.X, value.Y, value.Width, value.Height} {
		if err := validCoordinate(number, i >= 2); err != nil {
			return err
		}
	}
	return nil
}

// validate checks all local budgets before allocating graph indexes. The
// bounded linked adjacency and queue validate connectivity without recursion.
// Callers must not mutate the scene while Encode is in progress.
func validate(s Scene) error {
	if len(s.Nodes) > MaxNodes || len(s.Edges) > MaxEdges {
		return ErrLimitExceeded
	}
	if s.Schema != SchemaVersion || len(s.Nodes) == 0 || len(s.Edges) != len(s.Nodes)-1 {
		return ErrInvalidScene
	}
	if err := validString(s.Layout, MaxLayoutBytes, true); err != nil {
		return err
	}
	switch s.Layout {
	case "logicalStructureLeft", "logicalStructureRight", "logicalStructure", "orgChart":
	default:
		return ErrInvalidScene
	}
	if err := validRect(s.Bounds); err != nil {
		return err
	}
	root := -1
	for i, node := range s.Nodes {
		if err := validString(node.ID, MaxIDBytes, true); err != nil {
			return err
		}
		if err := validString(node.Label, MaxLabelBytes, false); err != nil {
			return err
		}
		if node.Shape != "rounded" {
			return ErrInvalidScene
		}
		if err := validRect(Rect{node.X, node.Y, node.Width, node.Height}); err != nil {
			return err
		}
		for _, paint := range [...]string{node.Fill, node.Stroke, node.Text} {
			if err := validPaint(paint); err != nil {
				return err
			}
		}
		if node.Root {
			if root != -1 {
				return ErrInvalidScene
			}
			root = i
		}
	}
	if root == -1 {
		return ErrInvalidScene
	}
	for _, edge := range s.Edges {
		for _, id := range [...]string{edge.FromID, edge.ToID} {
			if err := validString(id, MaxIDBytes, true); err != nil {
				return err
			}
		}
		for _, number := range [...]float64{edge.From.X, edge.From.Y, edge.To.X, edge.To.Y} {
			if err := validCoordinate(number, false); err != nil {
				return err
			}
		}
		if err := validPaint(edge.Stroke); err != nil {
			return err
		}
	}
	ids := make(map[string]int, len(s.Nodes))
	for i, node := range s.Nodes {
		if _, exists := ids[node.ID]; exists {
			return ErrInvalidScene
		}
		ids[node.ID] = i
	}
	parents := make([]bool, len(s.Nodes))
	head := make([]int, len(s.Nodes))
	for i := range head {
		head[i] = -1
	}
	next := make([]int, len(s.Edges))
	for i, edge := range s.Edges {
		from, fromOK := ids[edge.FromID]
		to, toOK := ids[edge.ToID]
		if !fromOK || !toOK || from == to || to == root || parents[to] {
			return ErrInvalidScene
		}
		parents[to] = true
		next[i], head[from] = head[from], i
	}
	for i, hasParent := range parents {
		if i != root && !hasParent {
			return ErrInvalidScene
		}
	}
	queue := make([]int, 1, len(s.Nodes))
	queue[0] = root
	for cursor := 0; cursor < len(queue); cursor++ {
		for edge := head[queue[cursor]]; edge != -1; edge = next[edge] {
			// With one parent per nonroot and none for root, a reachable
			// cycle is impossible; any disconnected cycle is caught below.
			queue = append(queue, ids[s.Edges[edge].ToID])
		}
	}
	if len(queue) != len(s.Nodes) {
		return ErrInvalidScene
	}
	return nil
}
