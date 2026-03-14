package cli

import (
	"strings"
	"testing"
)

func TestListMenuRenderList(t *testing.T) {
	menu := listMenu{
		profiles: []string{"beta", "demo", "prod"},
		index:    1,
	}

	rendered := menu.render()

	for _, fragment := range []string{
		"配置列表：",
		"  beta",
		"> demo",
		"  prod",
	} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected list render to contain %q, got %q", fragment, rendered)
		}
	}
}

func TestListMenuEnterActionsAndBack(t *testing.T) {
	menu := listMenu{
		profiles: []string{"beta", "demo"},
		index:    0,
	}

	menu.enterActions()
	if menu.mode != listMenuModeActions {
		t.Fatalf("expected actions mode, got %v", menu.mode)
	}

	rendered := menu.render()
	for _, fragment := range []string{
		"操作：beta",
		"> 切换",
		"  修改",
		"  删除",
		"  返回",
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
		index:       1,
	}

	menu.enterActions()
	rendered := menu.render()

	if !strings.Contains(rendered, "操作：demo") {
		t.Fatalf("expected current profile action header, got %q", rendered)
	}
	if strings.Contains(rendered, "删除") {
		t.Fatalf("expected current profile actions to hide remove, got %q", rendered)
	}
}

func TestListMenuCurrentProfileActionsStayFilteredAfterSelectionChanges(t *testing.T) {
	menu := listMenu{
		profiles:    []string{"beta", "demo", "prod"},
		currentName: "demo",
		index:       0,
	}

	menu.enterActions()
	if got := menu.selectedAction(); got != listMenuActionSwitch {
		t.Fatalf("expected non-current actions to start at switch, got %q", got)
	}
	menu.moveDown()
	menu.moveDown()
	if got := menu.selectedAction(); got != listMenuActionRemove {
		t.Fatalf("expected non-current actions to include remove, got %q", got)
	}

	menu.backToList()
	menu.moveDown()
	menu.enterActions()
	rendered := menu.render()
	if strings.Contains(rendered, "删除") {
		t.Fatalf("expected current profile actions to remain filtered after selection change, got %q", rendered)
	}
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
		index:       0,
	}

	menu.enterActions()
	rendered := menu.render()

	if !strings.Contains(rendered, "删除") {
		t.Fatalf("expected non-current profile actions to include remove, got %q", rendered)
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
		index:    0,
	}

	menu.enterActions()
	menu.actionIndex = 2
	menu.enterDeleteConfirm()
	if menu.mode != listMenuModeDeleteConfirm {
		t.Fatalf("expected delete-confirm mode, got %v", menu.mode)
	}

	rendered := menu.render()
	for _, fragment := range []string{
		"确认删除：beta",
		"> 确认删除",
		"  取消",
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
