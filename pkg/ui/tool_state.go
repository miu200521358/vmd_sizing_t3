package ui

import (
	"fmt"

	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/interface/app"
	"github.com/miu200521358/mlib_go/pkg/interface/controller"
	"github.com/miu200521358/mlib_go/pkg/interface/controller/widget"
	"github.com/miu200521358/vmd_sizing_t3/pkg/model"
	"github.com/miu200521358/walk/pkg/walk"
)

type ToolState struct {
	App                         *app.MApp
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

func NewToolState(app *app.MApp, controlWindow *controller.ControlWindow) *ToolState {
	toolState := &ToolState{
		App:           app,
		ControlWindow: controlWindow,
		SizingSets:    make([]*model.SizingSet, 0),
	}

	newSizingTab(controlWindow, toolState)
	toolState.addSizingSet()

	toolState.App.SetFuncGetModels(
		func() [][]*pmx.PmxModel {
			models := make([][]*pmx.PmxModel, 2)
			models[0] = make([]*pmx.PmxModel, len(toolState.SizingSets))
			models[1] = make([]*pmx.PmxModel, len(toolState.SizingSets))

			for i, sizingSet := range toolState.SizingSets {
				models[0][i] = sizingSet.SizingPmx
				models[1][i] = sizingSet.OriginalPmx
			}

			return models
		},
	)

	toolState.App.SetFuncGetMotions(
		func() [][]*vmd.VmdMotion {
			motions := make([][]*vmd.VmdMotion, 2)
			motions[0] = make([]*vmd.VmdMotion, len(toolState.SizingSets))
			motions[1] = make([]*vmd.VmdMotion, len(toolState.SizingSets))

			for i, sizingSet := range toolState.SizingSets {
				motions[0][i] = sizingSet.OutputVmd
				motions[1][i] = sizingSet.OriginalVmd
			}

			return motions
		},
	)

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
