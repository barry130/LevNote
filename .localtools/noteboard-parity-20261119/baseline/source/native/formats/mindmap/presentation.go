package mindmap

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/image/colornames"
)

// MaxDisplayLabelBytes bounds the UTF-8 prefix, excluding a trailing ellipsis.
const MaxDisplayLabelBytes = 512

const (
	IssueImagePlaceholder IssueCode = "IMAGE_PLACEHOLDER"
	IssueLabelTruncated   IssueCode = "LABEL_TRUNCATED"
	IssueUnsupportedColor IssueCode = "UNSUPPORTED_COLOR"
)

// NativeLayoutOptions returns the native canvas/export dimensions by value.
// It deliberately does not change Layout's general-purpose defaults.
func NativeLayoutOptions() LayoutOptions {
	return LayoutOptions{
		NodeHeight: 40, HorizontalGap: 72, VerticalGap: 24, Padding: 28,
		MinWidth: 92, MaxWidth: 380, CharWidth: 8, LineHeight: 20,
	}
}

// DisplayLabel assembles only a bounded prefix of icons, fallback topic and
// tags. It never joins the complete metadata merely to truncate it afterward.
// Node and fallback normally come from the same LayoutNode.
func DisplayLabel(node *Node, fallback string) string {
	if node == nil || node.Icons == nil && node.Tags == nil {
		if len(fallback) <= MaxDisplayLabelBytes {
			return fallback
		}
		end := MaxDisplayLabelBytes
		for end > 0 && !utf8.RuneStart(fallback[end]) {
			end--
		}
		return fallback[:end] + "…"
	}
	var out strings.Builder
	appendPart := func(value string) bool {
		remaining := MaxDisplayLabelBytes - out.Len()
		if len(value) <= remaining {
			out.WriteString(value)
			return true
		}
		for remaining > 0 && !utf8.RuneStart(value[remaining]) {
			remaining--
		}
		out.WriteString(value[:remaining])
		out.WriteString("…")
		return false
	}
	if node.Icons != nil {
		for _, icon := range *node.Icons {
			if !appendPart(icon) || !appendPart(" ") {
				return out.String()
			}
		}
	}
	if !appendPart(fallback) {
		return out.String()
	}
	if node.Tags != nil && len(*node.Tags) > 0 {
		if !appendPart(" [") {
			return out.String()
		}
		for i, tag := range *node.Tags {
			if i > 0 && !appendPart(" · ") || !appendPart(tag.Text) {
				return out.String()
			}
		}
		appendPart("]")
	}
	return out.String()
}

// DisplayLabelTruncated checks the topic/metadata budget without making a
// label. Each addition first checks the remaining budget to avoid overflow.
func DisplayLabelTruncated(node *Node) bool {
	if node == nil {
		return false
	}
	remaining := MaxDisplayLabelBytes
	consume := func(size int) bool {
		if size > remaining {
			return false
		}
		remaining -= size
		return true
	}
	if !consume(len(node.Topic)) {
		return true
	}
	if node.Icons != nil {
		for _, icon := range *node.Icons {
			if !consume(len(icon)) || !consume(1) {
				return true
			}
		}
	}
	if node.Tags != nil && len(*node.Tags) > 0 {
		if !consume(3) {
			return true
		}
		for i, tag := range *node.Tags {
			if !consume(len(tag.Text)) || i > 0 && !consume(len(" · ")) {
				return true
			}
		}
	}
	return false
}

// LiteralColor admits only concrete CSS colors, never paint servers,
// variables, inherited values or expressions. Input is bounded before trimming.
// Native rendering and SVG export use exactly this same subset.
func LiteralColor(value string) (string, bool) {
	if len(value) > 64 {
		return "", false
	}
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "none" || value == "transparent" {
		return "#00000000", true
	}
	if value == "rebeccapurple" {
		return "#663399", true
	}
	if len(value) == 4 || len(value) == 5 || len(value) == 7 || len(value) == 9 {
		if value[0] == '#' {
			for _, char := range value[1:] {
				if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f') {
					return "", false
				}
			}
			return value, true
		}
	}
	if color, ok := colornames.Map[value]; ok {
		return fmt.Sprintf("#%02x%02x%02x", color.R, color.G, color.B), true
	}
	return "", false
}

// PresentationIssues reports native/SVG fallbacks, including hidden children,
// in stable image, label, color order. Parse and layout issues stay separate.
// This cheap pass never creates labels, text layouts or SVG output.
func PresentationIssues(document *Document) []Issue {
	if document == nil || document.Root == nil {
		return nil
	}
	imageCount, unsupportedColors := 0, 0
	longLabel := false
	stack := []*Node{document.Root}
	for len(stack) > 0 {
		last := len(stack) - 1
		node := stack[last]
		stack = stack[:last]
		if node == nil {
			continue
		}
		longLabel = longLabel || DisplayLabelTruncated(node)
		unsupportedColors += unsupportedColorsInNode(node)
		if node.Image != nil {
			imageCount++
		}
		for i := len(node.Children) - 1; i >= 0; i-- {
			stack = append(stack, node.Children[i])
		}
	}
	var issues []Issue
	if imageCount > 0 {
		issues = append(issues, Issue{Severity: SeverityWarning, Code: IssueImagePlaceholder,
			Message: fmt.Sprintf("%d 个节点图片已保留；原生画布不访问文件或网络，显示为“图”占位", imageCount)})
	}
	if longLabel {
		issues = append(issues, Issue{Severity: SeverityWarning, Code: IssueLabelTruncated,
			Message: "画布与 SVG 的较长节点标签仅显示前 512 字节；完整文本仍保留，主题、标记与标签的完整内容请在源码中查看和编辑"})
	}
	if unsupportedColors > 0 {
		issues = append(issues, Issue{Severity: SeverityWarning, Code: IssueUnsupportedColor,
			Message: fmt.Sprintf("%d 个颜色值不受支持，画布与 SVG 使用默认颜色；原始字段已保留（支持十六进制与命名纯色）", unsupportedColors)})
	}
	return issues
}

func unsupportedColorsInNode(node *Node) int {
	count := 0
	if node.BranchColor != nil {
		if _, ok := LiteralColor(*node.BranchColor); !ok {
			count++
		}
	}
	if node.Style != nil {
		for _, key := range []string{"color", "background-color", "border"} {
			if _, exists := node.Style.Get(key); !exists {
				continue
			}
			value, isString := presentationObjectString(*node.Style, key)
			if _, ok := LiteralColor(value); !isString || !ok {
				count++
			}
		}
	}
	return count
}

func presentationObjectString(object Object, key string) (string, bool) {
	raw, ok := object.Get(key)
	if !ok {
		return "", false
	}
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return "", false
	}
	return value, true
}
