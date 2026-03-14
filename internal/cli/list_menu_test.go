package cli

import (
	"bufio"
	"strings"
	"testing"
)

func TestListMenuRenderList(t *testing.T) {
	menu := listMenu{
		profiles:    []string{"beta", "demo", "prod"},
		currentName: "demo",
		descriptions: map[string]string{
			"beta": "测试环境",
			"demo": "正式环境",
		},
		index: 0,
	}

	rendered := menu.render()

	for _, fragment := range []string{
		"配置列表",
		"当前配置：demo（当前） - 正式环境",
		"选择配置：",
		"> demo（当前） - 正式环境",
		"  beta - 测试环境",
		"  prod",
		"↑/↓ 选择  Enter 操作",
		"d 当前不可用",
		interactiveQuitHint,
	} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected list render to contain %q, got %q", fragment, rendered)
		}
	}
}

func TestListMenuRenderKeepsCurrentMarkerWhenSelectionMoves(t *testing.T) {
	menu := listMenu{
		profiles:    []string{"demo", "beta", "prod"},
		currentName: "demo",
		descriptions: map[string]string{
			"beta": "测试环境",
			"demo": "正式环境",
		},
		index: 1,
	}

	rendered := menu.render()

	for _, fragment := range []string{
		"当前配置：demo（当前） - 正式环境",
		"  demo（当前） - 正式环境",
		"> beta - 测试环境",
		"d 删除",
	} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected list render to contain %q, got %q", fragment, rendered)
		}
	}
}

func TestListMenuEnterActionsAndBack(t *testing.T) {
	menu := listMenu{
		profiles: []string{"beta", "demo"},
		descriptions: map[string]string{
			"beta": "测试环境",
		},
		index: 0,
	}

	menu.enterActions()
	if menu.mode != listMenuModeActions {
		t.Fatalf("expected actions mode, got %v", menu.mode)
	}

	rendered := menu.render()
	for _, fragment := range []string{
		"配置操作",
		"目标配置：beta - 测试环境",
		"可执行操作：",
		"> 切换",
		"  修改",
		"  重命名",
		"  删除",
		"  返回",
		"Enter 确认",
	} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected actions render to contain %q, got %q", fragment, rendered)
		}
	}

	menu.backToList()
	if menu.mode != listMenuModeProfiles {
		t.Fatalf("expected profile mode after back, got %v", menu.mode)
	}
}

func TestListMenuCurrentProfileActionsHideRemove(t *testing.T) {
	menu := listMenu{
		profiles:    []string{"beta", "demo"},
		currentName: "demo",
		descriptions: map[string]string{
			"demo": "正式环境",
		},
		index: 0,
	}

	menu.enterActions()
	rendered := menu.render()

	if !strings.Contains(rendered, "目标配置：demo（当前） - 正式环境") {
		t.Fatalf("expected current profile action header, got %q", rendered)
	}
	if !strings.Contains(rendered, "重命名") {
		t.Fatalf("expected current profile actions to include rename, got %q", rendered)
	}
	if strings.Contains(rendered, "\n  删除\n") {
		t.Fatalf("expected current profile actions to hide remove, got %q", rendered)
	}
}

func TestListMenuCurrentProfileActionsStayFilteredAfterSelectionChanges(t *testing.T) {
	menu := listMenu{
		profiles:    []string{"beta", "demo", "prod"},
		currentName: "demo",
		descriptions: map[string]string{
			"beta": "测试环境",
			"demo": "正式环境",
		},
		index: 1,
	}

	menu.enterActions()
	if got := menu.selectedAction(); got != listMenuActionSwitch {
		t.Fatalf("expected non-current actions to start at switch, got %q", got)
	}
	menu.moveDown()
	menu.moveDown()
	menu.moveDown()
	if got := menu.selectedAction(); got != listMenuActionRemove {
		t.Fatalf("expected non-current actions to include remove, got %q", got)
	}

	menu.backToList()
	menu.moveUp()
	menu.enterActions()
	rendered := menu.render()
	if strings.Contains(rendered, "\n  删除\n") {
		t.Fatalf("expected current profile actions to remain filtered after selection change, got %q", rendered)
	}
	menu.moveDown()
	menu.moveDown()
	menu.moveDown()
	if got := menu.selectedAction(); got != listMenuActionBack {
		t.Fatalf("expected filtered action set to wrap on back, got %q", got)
	}
}

func TestListMenuNonCurrentProfileStillShowsRemove(t *testing.T) {
	menu := listMenu{
		profiles:    []string{"beta", "demo"},
		currentName: "demo",
		descriptions: map[string]string{
			"beta": "测试环境",
			"demo": "正式环境",
		},
		index: 1,
	}

	menu.enterActions()
	rendered := menu.render()

	if !strings.Contains(rendered, "删除") {
		t.Fatalf("expected non-current profile actions to include remove, got %q", rendered)
	}
	if !strings.Contains(rendered, "重命名") {
		t.Fatalf("expected non-current profile actions to include rename, got %q", rendered)
	}
}

func TestListMenuMoveWrapsInCurrentMode(t *testing.T) {
	menu := listMenu{
		profiles: []string{"beta", "demo", "prod"},
		index:    0,
	}

	menu.moveUp()
	if got := menu.selectedProfile(); got != "prod" {
		t.Fatalf("expected wrap-around profile selection, got %q", got)
	}

	menu.enterActions()
	menu.moveUp()
	if got := menu.selectedAction(); got != listMenuActionBack {
		t.Fatalf("expected wrap-around action selection, got %q", got)
	}
}

func TestListMenuEnterDeleteConfirmAndCancel(t *testing.T) {
	menu := listMenu{
		profiles: []string{"beta", "demo"},
		descriptions: map[string]string{
			"beta": "测试环境",
		},
		index: 0,
	}

	menu.enterActions()
	menu.actionIndex = 3
	menu.enterDeleteConfirm()
	if menu.mode != listMenuModeDeleteConfirm {
		t.Fatalf("expected delete-confirm mode, got %v", menu.mode)
	}

	rendered := menu.render()
	for _, fragment := range []string{
		"删除配置",
		"目标配置：beta - 测试环境",
		"此操作不可恢复，请再次确认。",
		"> 确认删除",
		"  取消",
		"Enter 确认",
	} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected delete confirm render to contain %q, got %q", fragment, rendered)
		}
	}

	menu.moveDown()
	if got := menu.selectedConfirmAction(); got != listMenuConfirmCancel {
		t.Fatalf("expected cancel option selection, got %q", got)
	}

	menu.backToActions()
	if menu.mode != listMenuModeActions {
		t.Fatalf("expected actions mode after back, got %v", menu.mode)
	}
}

func TestReadSelectorActionSupportsDirectListShortcuts(t *testing.T) {
	cases := map[string]selectorAction{
		"e": selectorActionEdit,
		"E": selectorActionEdit,
		"r": selectorActionRename,
		"R": selectorActionRename,
		"d": selectorActionRemove,
		"D": selectorActionRemove,
	}

	for input, want := range cases {
		reader := bufio.NewReader(strings.NewReader(input))

		got, err := readSelectorAction(reader)
		if err != nil {
			t.Fatalf("expected no error for %q, got %v", input, err)
		}
		if got != want {
			t.Fatalf("expected action %v for %q, got %v", want, input, got)
		}
	}
}
