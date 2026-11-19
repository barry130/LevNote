package mindmap

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

type Point struct {
	X float64
	Y float64
}

type Rect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

type SceneNode struct {
	ID              string
	Text            string
	X               float64
	Y               float64
	Width           float64
	Height          float64
	Shape           string
	Parent          string
	Depth           int
	Side            string
	Collapsed       bool
	ChildCount      int
	VisibleChildren int
	BranchColor     string
	Node            *Node
}

type SceneEdge struct {
	ID        string
	From      string
	To        string
	FromPoint Point
	ToPoint   Point
	Color     string
}

type LayoutResult struct {
	Kind   string
	Nodes  []SceneNode
	Edges  []SceneEdge
	Bounds Rect
	Issues []Issue
}

// MeasureFunc returns the display width and height of a topic.
type MeasureFunc func(text string) (width, height float64)

type LayoutOptions struct {
	NodeHeight    float64
	HorizontalGap float64
	VerticalGap   float64
	Padding       float64
	MinWidth      float64
	MaxWidth      float64
	CharWidth     float64
	LineHeight    float64
	Measure       MeasureFunc
}

type layoutState struct {
	options LayoutOptions
	result  *LayoutResult
	byNode  map[*Node]int
}

// Layout returns a deterministic canvas-independent scene. Unknown layout
// names render with the default two-sided layout and keep the file unchanged.
func Layout(document *Document, options LayoutOptions) LayoutResult {
	options = normalizeLayoutOptions(options)
	kind := LayoutBoth
	if document != nil {
		kind = document.Layout
	}
	result := LayoutResult{Kind: kind, Issues: make([]Issue, 0)}
	if document == nil || document.Root == nil {
		result.Issues = append(result.Issues, Issue{
			Severity: SeverityError,
			Code:     "MISSING_ROOT",
			Message:  "cannot lay out a mindmap without a root",
		})
		return result
	}
	if err := document.validate(); err != nil {
		result.Issues = append(result.Issues, Issue{
			Severity: SeverityError,
			Code:     "INVALID_DOCUMENT",
			Message:  err.Error(),
		})
		return result
	}
	if !KnownLayout(kind) {
		result.Issues = append(result.Issues, Issue{
			Severity: SeverityWarning,
			Code:     IssueUnknownLayout,
			Message:  fmt.Sprintf("layout %q is preserved and rendered as %q", kind, LayoutBoth),
		})
		result.Kind = LayoutBoth
		kind = LayoutBoth
	}

	state := layoutState{
		options: options,
		result:  &result,
		byNode:  make(map[*Node]int),
	}
	state.makeSceneNode(document.Root, "", "center", "")
	root := state.sceneNode(document.Root)
	switch kind {
	case LayoutLeft:
		root.X = 0
		root.Y = -root.Height / 2
		state.placeHorizontal(document.Root, root.X, 0, "left", 1)
	case LayoutRight:
		root.X = 0
		root.Y = -root.Height / 2
		state.placeHorizontal(document.Root, root.X, 0, "right", 1)
	case LayoutOrg:
		state.placeOrg(document.Root, 0, 0)
	default:
		root.X = 0
		root.Y = -root.Height / 2
		if document.Root.Collapsed == nil || !*document.Root.Collapsed {
			leftChildren, rightChildren := balancedSides(document.Root.Children, &state)
			state.placeChildren(document.Root, leftChildren, "left", 1)
			state.placeChildren(document.Root, rightChildren, "right", 1)
		}
	}
	state.finishBounds()
	return result
}

func normalizeLayoutOptions(options LayoutOptions) LayoutOptions {
	if options.NodeHeight <= 0 {
		options.NodeHeight = 38
	}
	if options.HorizontalGap <= 0 {
		options.HorizontalGap = 72
	}
	if options.VerticalGap <= 0 {
		options.VerticalGap = 22
	}
	if options.Padding < 0 {
		options.Padding = 0
	} else if options.Padding == 0 {
		options.Padding = 24
	}
	if options.MinWidth <= 0 {
		options.MinWidth = 88
	}
	if options.MaxWidth <= options.MinWidth {
		options.MaxWidth = 360
	}
	if options.CharWidth <= 0 {
		options.CharWidth = 8
	}
	if options.LineHeight <= 0 {
		options.LineHeight = 20
	}
	return options
}

func (s *layoutState) measure(text string) (float64, float64) {
	if s.options.Measure != nil {
		width, height := s.options.Measure(text)
		if math.IsNaN(width) || math.IsInf(width, 0) || width <= 0 {
			width = s.options.MinWidth
		}
		if math.IsNaN(height) || math.IsInf(height, 0) || height <= 0 {
			height = s.options.NodeHeight
		}
		return clamp(width, s.options.MinWidth, s.options.MaxWidth), math.Max(height, s.options.NodeHeight)
	}
	lines := strings.Split(text, "\n")
	longest := 0
	for _, line := range lines {
		count := utf8.RuneCountInString(line)
		if count > longest {
			longest = count
		}
	}
	width := clamp(float64(longest)*s.options.CharWidth+28, s.options.MinWidth, s.options.MaxWidth)
	height := math.Max(s.options.NodeHeight, float64(len(lines))*s.options.LineHeight+16)
	return width, height
}

func (s *layoutState) makeSceneNode(node *Node, parent, side, inheritedColor string) int {
	width, height := s.measure(node.Topic)
	color := inheritedColor
	if node.BranchColor != nil {
		color = *node.BranchColor
	}
	collapsed := node.Collapsed != nil && *node.Collapsed
	scene := &SceneNode{
		ID:          node.ID,
		Text:        node.Topic,
		Width:       width,
		Height:      height,
		Shape:       "rounded",
		Parent:      parent,
		Side:        side,
		Collapsed:   collapsed,
		ChildCount:  len(node.Children),
		BranchColor: color,
		Node:        node,
	}
	if !collapsed {
		scene.VisibleChildren = len(node.Children)
	}
	s.result.Nodes = append(s.result.Nodes, *scene)
	index := len(s.result.Nodes) - 1
	s.byNode[node] = index
	return index
}

func (s *layoutState) sceneNode(node *Node) *SceneNode {
	if index, ok := s.byNode[node]; ok {
		return &s.result.Nodes[index]
	}
	return nil
}

func (s *layoutState) placeHorizontal(node *Node, left, centerY float64, side string, depth int) {
	scene := s.sceneNode(node)
	if scene == nil {
		return
	}
	scene.X = left
	scene.Y = centerY - scene.Height/2
	scene.Depth = depth - 1
	if node.Collapsed != nil && *node.Collapsed {
		return
	}
	s.placeChildren(node, node.Children, side, depth)
}

func (s *layoutState) placeChildren(parent *Node, children []*Node, side string, depth int) {
	if len(children) == 0 {
		return
	}
	parentScene := s.sceneNode(parent)
	if parentScene == nil {
		return
	}
	parentCopy := *parentScene
	centerY := parentCopy.Y + parentCopy.Height/2
	total := 0.0
	heights := make([]float64, len(children))
	for i, child := range children {
		heights[i] = s.horizontalSubtreeHeight(child)
		total += heights[i]
	}
	if len(children) > 1 {
		total += float64(len(children)-1) * s.options.VerticalGap
	}
	cursor := centerY - total/2
	inherited := parentCopy.BranchColor
	for i, child := range children {
		childCenter := cursor + heights[i]/2
		var childLeft float64
		width, _ := s.measure(child.Topic)
		if side == "left" {
			childLeft = parentCopy.X - s.options.HorizontalGap - width
		} else {
			childLeft = parentCopy.X + parentCopy.Width + s.options.HorizontalGap
		}
		s.makeSceneNode(child, parent.ID, side, inherited)
		childScene := s.sceneNode(child)
		childScene.X = childLeft
		childScene.Y = childCenter - childScene.Height/2
		childScene.Depth = depth
		if child.Collapsed == nil || !*child.Collapsed {
			s.placeChildren(child, child.Children, side, depth+1)
		}
		parentFinal := *s.sceneNode(parent)
		childFinal := *s.sceneNode(child)
		s.addHorizontalEdge(&parentFinal, &childFinal, side)
		cursor += heights[i] + s.options.VerticalGap
	}
}

func (s *layoutState) horizontalSubtreeHeight(node *Node) float64 {
	width, height := s.measure(node.Topic)
	_ = width
	if node.Collapsed != nil && *node.Collapsed || len(node.Children) == 0 {
		return height
	}
	childTotal := 0.0
	for _, child := range node.Children {
		childTotal += s.horizontalSubtreeHeight(child)
	}
	childTotal += float64(len(node.Children)-1) * s.options.VerticalGap
	return math.Max(height, childTotal)
}

func (s *layoutState) addHorizontalEdge(parent, child *SceneNode, side string) {
	edge := SceneEdge{
		ID:    fmt.Sprintf("edge:%s:%s", parent.ID, child.ID),
		From:  parent.ID,
		To:    child.ID,
		Color: child.BranchColor,
	}
	if side == "left" {
		edge.FromPoint = Point{X: parent.X, Y: parent.Y + parent.Height/2}
		edge.ToPoint = Point{X: child.X + child.Width, Y: child.Y + child.Height/2}
	} else {
		edge.FromPoint = Point{X: parent.X + parent.Width, Y: parent.Y + parent.Height/2}
		edge.ToPoint = Point{X: child.X, Y: child.Y + child.Height/2}
	}
	s.result.Edges = append(s.result.Edges, edge)
}

func (s *layoutState) placeOrg(node *Node, centerX, depthY float64) {
	scene := s.sceneNode(node)
	if scene == nil {
		return
	}
	scene.X = centerX - scene.Width/2
	scene.Y = depthY
	s.placeOrgChildren(node, centerX)
}

func (s *layoutState) placeOrgChildren(node *Node, centerX float64) {
	if node.Collapsed != nil && *node.Collapsed || len(node.Children) == 0 {
		return
	}
	parentScene := *s.sceneNode(node)
	total := 0.0
	widths := make([]float64, len(node.Children))
	for i, child := range node.Children {
		widths[i] = s.orgSubtreeWidth(child)
		total += widths[i]
	}
	total += float64(len(node.Children)-1) * s.options.HorizontalGap
	cursor := centerX - total/2
	childY := parentScene.Y + parentScene.Height + s.options.VerticalGap
	for i, child := range node.Children {
		childCenter := cursor + widths[i]/2
		s.makeSceneNode(child, node.ID, "down", parentScene.BranchColor)
		childScene := s.sceneNode(child)
		childScene.X = childCenter - childScene.Width/2
		childScene.Y = childY
		childScene.Depth = parentScene.Depth + 1
		s.placeOrgChildren(child, childCenter)
		parentFinal := *s.sceneNode(node)
		childFinal := *s.sceneNode(child)
		s.result.Edges = append(s.result.Edges, SceneEdge{
			ID:        fmt.Sprintf("edge:%s:%s", node.ID, child.ID),
			From:      node.ID,
			To:        child.ID,
			FromPoint: Point{X: centerX, Y: parentFinal.Y + parentFinal.Height},
			ToPoint:   Point{X: childFinal.X + childFinal.Width/2, Y: childFinal.Y},
			Color:     childFinal.BranchColor,
		})
		cursor += widths[i] + s.options.HorizontalGap
	}
}

func (s *layoutState) orgSubtreeWidth(node *Node) float64 {
	width, _ := s.measure(node.Topic)
	if node.Collapsed != nil && *node.Collapsed || len(node.Children) == 0 {
		return width
	}
	childrenWidth := float64(len(node.Children)-1) * s.options.HorizontalGap
	for _, child := range node.Children {
		childrenWidth += s.orgSubtreeWidth(child)
	}
	return math.Max(width, childrenWidth)
}

func (s *layoutState) finishBounds() {
	if len(s.result.Nodes) == 0 {
		s.result.Bounds = Rect{}
		return
	}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, node := range s.result.Nodes {
		minX = math.Min(minX, node.X)
		minY = math.Min(minY, node.Y)
		maxX = math.Max(maxX, node.X+node.Width)
		maxY = math.Max(maxY, node.Y+node.Height)
	}
	offsetX := s.options.Padding - minX
	offsetY := s.options.Padding - minY
	for i := range s.result.Nodes {
		s.result.Nodes[i].X += offsetX
		s.result.Nodes[i].Y += offsetY
	}
	for i := range s.result.Edges {
		s.result.Edges[i].FromPoint.X += offsetX
		s.result.Edges[i].FromPoint.Y += offsetY
		s.result.Edges[i].ToPoint.X += offsetX
		s.result.Edges[i].ToPoint.Y += offsetY
	}
	s.result.Bounds = Rect{
		X:      0,
		Y:      0,
		Width:  maxX - minX + 2*s.options.Padding,
		Height: maxY - minY + 2*s.options.Padding,
	}
}

func balancedSides(children []*Node, state *layoutState) (left, right []*Node) {
	leftHeight, rightHeight := 0.0, 0.0
	for _, child := range children {
		height := state.horizontalSubtreeHeight(child)
		if leftHeight <= rightHeight {
			left = append(left, child)
			leftHeight += height
		} else {
			right = append(right, child)
			rightHeight += height
		}
	}
	return left, right
}

func clamp(value, low, high float64) float64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
