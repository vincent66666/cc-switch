package cli

import (
	"strings"
	"testing"
)

func TestStatusSelectorRender(t *testing.T) {
	selector := statusSelector{
		currentName:        "demo",
		currentDescription: "正式环境",
		baseURL:            "https://example.com",
		model:              "glm-5",
		names:              []string{"beta", "prod"},
		descriptions: map[string]string{
			"beta": "测试环境",
			"prod": "生产环境",
		},
		index: 0,
	}

	rendered := selector.render()

	for _, fragment := range []string{
		"当前配置：demo - 正式环境",
		"接口地址：https://example.com",
		"模型：glm-5",
		"可用配置：",
		"> demo（当前） - 正式环境",
		"  beta - 测试环境",
		"  prod - 生产环境",
	} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected rendered selector to contain %q, got %q", fragment, rendered)
		}
	}
}

func TestStatusSelectorRenderIncludesCurrentProfileFirstAndMarked(t *testing.T) {
	selector := statusSelector{
		currentName:        "demo",
		currentDescription: "正式环境",
		baseURL:            "https://example.com",
		model:              "glm-5",
		names:              []string{"beta", "prod"},
		descriptions: map[string]string{
			"beta": "测试环境",
			"prod": "生产环境",
		},
		index: 0,
	}

	rendered := selector.render()

	for _, fragment := range []string{
		"> demo（当前） - 正式环境",
		"  beta - 测试环境",
		"  prod - 生产环境",
	} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected rendered selector to contain %q, got %q", fragment, rendered)
		}
	}
}

func TestStatusSelectorMoveWrapsAround(t *testing.T) {
	selector := statusSelector{
		names: []string{"beta", "prod", "staging"},
		index: 0,
	}

	selector.moveUp()
	if got := selector.selectedName(); got != "staging" {
		t.Fatalf("expected wrap-around up selection, got %q", got)
	}

	selector.moveDown()
	selector.moveDown()
	if got := selector.selectedName(); got != "prod" {
		t.Fatalf("expected move down selection, got %q", got)
	}
}

func TestStatusSelectorSelectedNameStartsFromCurrentProfile(t *testing.T) {
	selector := statusSelector{
		currentName: "demo",
		names:       []string{"beta", "prod"},
		index:       0,
	}

	if got := selector.selectedName(); got != "demo" {
		t.Fatalf("expected current profile to be the first selectable item, got %q", got)
	}

	selector.moveDown()
	if got := selector.selectedName(); got != "beta" {
		t.Fatalf("expected move down to reach next profile after current, got %q", got)
	}
}

func TestStatusSelectorSelectedNameEmptyWhenNoAvailableProfiles(t *testing.T) {
	selector := statusSelector{}

	if got := selector.selectedName(); got != "" {
		t.Fatalf("expected empty selection, got %q", got)
	}
}

func TestStatusSelectorRenderUsesDashForEmptyModel(t *testing.T) {
	selector := statusSelector{
		currentName:        "demo",
		currentDescription: "正式环境",
		baseURL:            "https://example.com",
		names:              []string{"beta"},
		descriptions: map[string]string{
			"beta": "测试环境",
		},
	}

	rendered := selector.render()

	if !strings.Contains(rendered, "模型：-") {
		t.Fatalf("expected empty model to render as dash, got %q", rendered)
	}
}
