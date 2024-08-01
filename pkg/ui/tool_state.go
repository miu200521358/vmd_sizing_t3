package ui

import (
	"fmt"

	"github.com/miu200521358/mlib_go/pkg/infrastructure/state"
	"github.com/miu200521358/mlib_go/pkg/interface/controller"
	"github.com/miu200521358/mlib_go/pkg/interface/controller/widget"
	"github.com/miu200521358/vmd_sizing_t3/pkg/model"
	"github.com/miu200521358/walk/pkg/walk"
)

type ToolState struct {
	AppState                    state.IAppState
	ControlWindow               *controller.ControlWindow
	SizingTab                   *widget.MTabPage   // ファイルタブページ
	CurrentIndex                int                // 現在のインデックス
	NavToolBar                  *walk.ToolBar      // サイジングNo.ナビゲーション
	SizingSets                  []*model.SizingSet // サイジング情報セット
	OriginalVmdPicker           *widget.FilePicker // サイジング対象モーション(Vmd/Vpd)ファイル選択
	OriginalPmxPicker           *widget.FilePicker // モーション作成元モデル(Pmx)ファイル選択
	SizingPmxPicker             *widget.FilePicker // サイジング先モデル(Pmx)ファイル選択
	OutputPmxPicker             *widget.FilePicker // 出力モデル(Pmx)ファイル選択
	OutputVmdPicker             *widget.FilePicker // 出力モーション(Vmd)ファイル選択
	SizingTabSaveButton         *walk.PushButton   // サイジングタブ保存ボタン
	currentPageChangedPublisher walk.EventPublisher
}

func NewToolState(appState state.IAppState, controlWindow *controller.ControlWindow) *ToolState {
	toolState := &ToolState{
		AppState:      appState,
		ControlWindow: controlWindow,
		SizingSets:    make([]*model.SizingSet, 0),
	}

	newSizingTab(controlWindow, toolState)
	toolState.addSizingSet()

	return toolState
}

func (toolState *ToolState) resetSizingSet() error {
	// 一旦全部削除
	for range toolState.NavToolBar.Actions().Len() {
		toolState.NavToolBar.Actions().RemoveAt(toolState.NavToolBar.Actions().Len() - 1)
	}
	toolState.SizingSets = make([]*model.SizingSet, 0)
	toolState.CurrentIndex = -1

	// 1セット追加
	err := toolState.addSizingSet()
	if err != nil {
		return err
	}

	return nil
}

func (toolState *ToolState) CurrentPageChanged() *walk.Event {
	return toolState.currentPageChangedPublisher.Event()
}

func (toolState *ToolState) addSizingSet() error {
	action, err := toolState.newPageAction()
	if err != nil {
		return err
	}
	toolState.NavToolBar.Actions().Add(action)
	toolState.SizingSets = append(toolState.SizingSets, model.NewSizingSet(len(toolState.SizingSets)))

	if len(toolState.SizingSets) > 0 {
		if err := toolState.setCurrentAction(len(toolState.SizingSets) - 1); err != nil {
			return err
		}
	}

	return nil
}

func (toolState *ToolState) newPageAction() (*walk.Action, error) {
	action := walk.NewAction()
	action.SetCheckable(true)
	action.SetExclusive(true)
	action.SetText(fmt.Sprintf("No. %d", len(toolState.SizingSets)+1))
	index := len(toolState.SizingSets)

	action.Triggered().Attach(func() {
		toolState.setCurrentAction(index)
	})

	return action, nil
}

func (toolState *ToolState) setCurrentAction(index int) error {
	// 一旦すべてのチェックを外す
	for i := range len(toolState.SizingSets) {
		toolState.NavToolBar.Actions().At(i).SetChecked(false)
	}
	// 該当INDEXのみチェックON
	toolState.CurrentIndex = index
	toolState.NavToolBar.Actions().At(index).SetChecked(true)
	toolState.currentPageChangedPublisher.Publish()

	return nil
}
