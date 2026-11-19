package app

import (
	"strings"
	"testing"

	"levnote.local/native/core/plugins"
)

func TestDeclarativeCapabilityTextDistinguishesSchemaAndMissingOutline(t *testing.T) {
	cases := []struct {
		name   string
		syntax plugins.SyntaxManifest
		want   []string
		absent []string
	}{
		{
			name:   "schema1 baseline highlight only",
			syntax: plugins.SyntaxManifest{Schema: 1},
			want:   []string{"schema1", "基础字符串/数字高亮", "未配置额外高亮规则", "不支持源码大纲"},
			absent: []string{"仅源码编辑", "源码大纲（"},
		},
		{
			name:   "schema1 with configured highlight",
			syntax: plugins.SyntaxManifest{Schema: 1, LineComment: "#"},
			want:   []string{"schema1", "基础字符串/数字高亮", "已配置行内规则", "不支持源码大纲"},
			absent: []string{"仅源码编辑", "源码大纲（"},
		},
		{
			name:   "schema2 no rules baseline highlight only",
			syntax: plugins.SyntaxManifest{Schema: 2},
			want:   []string{"schema2", "未声明源码大纲", "基础字符串/数字高亮", "未配置额外高亮规则"},
			absent: []string{"仅源码编辑", "源码大纲（"},
		},
		{
			name:   "schema2 no rules with configured highlight",
			syntax: plugins.SyntaxManifest{Schema: 2, LineComment: "#"},
			want:   []string{"schema2", "未声明源码大纲", "基础字符串/数字高亮", "已配置行内规则"},
			absent: []string{"源码大纲（"},
		},
		{
			name: "schema2 rules",
			syntax: plugins.SyntaxManifest{
				Schema: 2,
				Outline: []plugins.OutlineRule{
					{Prefix: "[", Suffix: "]", Level: 1},
					{Prefix: "# ", Level: 2},
				},
				LineComment: "#",
			},
			want: []string{
				"schema2", "源码大纲（2 条逐行字面规则）",
				"基础字符串/数字高亮", "已配置行内规则",
				"不识别多行注释/字符串上下文",
			},
			absent: []string{"仅源码编辑"},
		},
		{
			name: "schema2 rules without configured highlight",
			syntax: plugins.SyntaxManifest{
				Schema:  2,
				Outline: []plugins.OutlineRule{{Prefix: "[", Suffix: "]", Level: 1}},
			},
			want: []string{
				"schema2", "源码大纲（1 条逐行字面规则）",
				"基础字符串/数字高亮", "未配置额外高亮规则",
				"不识别多行注释/字符串上下文",
			},
			absent: []string{"已配置行内规则", "仅源码编辑"},
		},
		{
			name:   "schema3 draw.io route",
			syntax: plugins.SyntaxManifest{Schema: 3, NativeRoute: plugins.NativeRouteDrawio},
			want: []string{
				"使用 draw.io 图纸", "随包原生格式能力",
				"阅读、编辑、分屏与导出支持以当前格式为准",
				"源码高亮按单行规则处理",
			},
			absent: []string{"大纲", "复用内置编辑"},
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := declarativeCapability(test.syntax)
			for _, fragment := range test.want {
				if !strings.Contains(got, fragment) {
					t.Fatalf("capability text %q does not contain %q", got, fragment)
				}
			}
			for _, fragment := range test.absent {
				if strings.Contains(got, fragment) {
					t.Fatalf("capability text %q unexpectedly contains %q", got, fragment)
				}
			}
		})
	}
}

func TestDeclarativeCapabilityBadgeStaysCompact(t *testing.T) {
	withOutline := plugins.SyntaxManifest{
		Schema:      2,
		LineComment: "#",
		Outline:     []plugins.OutlineRule{{Prefix: "[", Suffix: "]", Level: 1}},
	}
	if got := declarativeCapabilityBadge(withOutline); got != "源码大纲 · 高亮" {
		t.Fatalf("outline badge = %q", got)
	}
	if got := declarativeCapabilityBadge(plugins.SyntaxManifest{
		Schema:  2,
		Outline: []plugins.OutlineRule{{Prefix: "[", Suffix: "]", Level: 1}},
	}); got != "源码大纲 · 高亮" {
		t.Fatalf("outline-only badge = %q", got)
	}
	for _, syntax := range []plugins.SyntaxManifest{
		{Schema: 1, LineComment: "#"},
		{Schema: 2, LineComment: "#"},
	} {
		if got := declarativeCapabilityBadge(syntax); got != "仅行内高亮" {
			t.Fatalf("highlight-only badge = %q", got)
		}
	}
	for _, syntax := range []plugins.SyntaxManifest{{Schema: 1}, {Schema: 2}} {
		if got := declarativeCapabilityBadge(syntax); got != "仅行内高亮" {
			t.Fatalf("baseline-highlight badge = %q", got)
		}
	}
	if got := declarativeCapabilityBadge(plugins.SyntaxManifest{
		Schema: 3, NativeRoute: plugins.NativeRouteDrawio,
	}); got != "使用 draw.io 图纸" {
		t.Fatalf("draw.io route badge = %q", got)
	}
}
