package ui

import (
	"fmt"
	"strings"

	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/interface/app"
	"github.com/miu200521358/mlib_go/pkg/interface/controller"
	"github.com/miu200521358/mlib_go/pkg/interface/controller/widget"
	"github.com/miu200521358/vmd_sizing_t3/pkg/model"
	"github.com/miu200521358/walk/pkg/walk"
)

type ToolState struct {
	App                           *app.MApp
	ControlWindow                 *controller.ControlWindow
	SizingTab                     *widget.MTabPage   // サイジングタブページ
	CurrentIndex                  int                // 現在のインデックス
	NavToolBar                    *walk.ToolBar      // サイジングNo.ナビゲーション
	SizingSets                    []*model.SizingSet // サイジング情報セット
	OriginalVmdPicker             *widget.FilePicker // サイジング対象モーション(Vmd/Vpd)ファイル選択
	OriginalPmxPicker             *widget.FilePicker // モーション作成元モデル(Pmx)ファイル選択
	SizingPmxPicker               *widget.FilePicker // サイジング先モデル(Pmx)ファイル選択
	OutputVmdPicker               *widget.FilePicker // 出力モーション(Vmd)ファイル選択
	SizingArmStanceCheck          *walk.CheckBox     // サイジング腕スタンス補正チェックボックス
	SizingTranslateCheck          *walk.CheckBox     // サイジング移動補正チェックボックス
	OriginalPmxRatioEdit          *walk.NumberEdit   // オリジナルモデル比率編集
	OriginalPmxUpperLengthEdit    *walk.NumberEdit   // 素体上半身長さ編集
	OriginalPmxUpperAngleEdit     *walk.NumberEdit   // 素体上半身角度編集
	OriginalPmxUpper2LengthEdit   *walk.NumberEdit   // 素体上半身2長さ編集
	OriginalPmxUpper2AngleEdit    *walk.NumberEdit   // 素体上半身2角度編集
	OriginalPmxNeckLengthEdit     *walk.NumberEdit   // 素体首長さ編集
	OriginalPmxNeckAngleEdit      *walk.NumberEdit   // 素体首角度編集
	OriginalPmxHeadLengthEdit     *walk.NumberEdit   // 素体頭編集
	OriginalPmxShoulderLengthEdit *walk.NumberEdit   // 素体肩長さ
	OriginalPmxShoulderAngleEdit  *walk.NumberEdit   // 素体肩角度編集
	OriginalPmxArmLengthEdit      *walk.NumberEdit   // 素体腕長さ編集
	OriginalPmxArmAngleEdit       *walk.NumberEdit   // 素体腕角度編集
	OriginalPmxElbowLengthEdit    *walk.NumberEdit   // 素体ひじ長さ編集
	OriginalPmxElbowAngleEdit     *walk.NumberEdit   // 素体ひじ角度編集
	OriginalPmxWristLengthEdit    *walk.NumberEdit   // 素体手首長さ編集
	OriginalPmxWristAngleEdit     *walk.NumberEdit   // 素体手首角度編集
	OriginalPmxLowerLengthEdit    *walk.NumberEdit   // 素体下半身長さ編集
	OriginalPmxLowerAngleEdit     *walk.NumberEdit   // 素体下半身角度編集
	OriginalPmxLegWidthEdit       *walk.NumberEdit   // 素体足横幅編集
	OriginalPmxLegLengthEdit      *walk.NumberEdit   // 素体足長さ編集
	OriginalPmxLegAngleEdit       *walk.NumberEdit   // 素体足角度編集
	OriginalPmxKneeLengthEdit     *walk.NumberEdit   // 素体ひざ長さ編集
	OriginalPmxKneeAngleEdit      *walk.NumberEdit   // 素体ひざ角度編集
	OriginalPmxAnkleLengthEdit    *walk.NumberEdit   // 素体足首長さ編集
	SizingTabSaveButton           *walk.PushButton   // サイジングタブ保存ボタン
	currentPageChangedPublisher   walk.EventPublisher
	JsonSaveTab                   *widget.MTabPage // サイジングタブページ
}

func NewToolState(app *app.MApp, controlWindow *controller.ControlWindow) *ToolState {
	toolState := &ToolState{
		App:           app,
		ControlWindow: controlWindow,
		SizingSets:    make([]*model.SizingSet, 0),
	}

	newSizingTab(controlWindow, toolState)
	toolState.addSizingSet()
	toolState.SetOriginalPmxParameterEnabled(false)

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

	// json保存タブ
	newJsonSaveTab(controlWindow, toolState)

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

	// セット追加したら一旦クリア
	toolState.OriginalVmdPicker.SetPath("")
	toolState.OriginalVmdPicker.SetName("")

	toolState.OriginalPmxPicker.SetPath("")
	toolState.OriginalPmxPicker.SetName("")

	toolState.SizingPmxPicker.SetPath("")
	toolState.SizingPmxPicker.SetName("")

	toolState.OutputVmdPicker.SetPath("")

	toolState.ResetSizingParameter()
	toolState.ResetOriginalPmxParameter()

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

	// サイジングセットの情報を切り替え
	sizingSet := toolState.SizingSets[index]
	toolState.OriginalVmdPicker.SetPath(sizingSet.OriginalVmdPath)
	toolState.OriginalVmdPicker.SetName(sizingSet.OriginalPmxName)

	toolState.OriginalPmxPicker.SetPath(sizingSet.OriginalPmxPath)
	toolState.OriginalPmxPicker.SetName(sizingSet.OriginalPmxName)

	toolState.SizingPmxPicker.SetPath(sizingSet.SizingPmxPath)
	toolState.SizingPmxPicker.SetName(sizingSet.SizingPmxName)

	toolState.OutputVmdPicker.SetPath(sizingSet.OutputVmdPath)

	toolState.OriginalPmxRatioEdit.SetValue(sizingSet.OriginalPmxRatio)
	toolState.OriginalPmxUpperLengthEdit.SetValue(sizingSet.OriginalPmxUpperLength)
	toolState.OriginalPmxUpperAngleEdit.SetValue(sizingSet.OriginalPmxUpperAngle)
	toolState.OriginalPmxUpper2LengthEdit.SetValue(sizingSet.OriginalPmxUpper2Length)
	toolState.OriginalPmxUpper2AngleEdit.SetValue(sizingSet.OriginalPmxUpper2Angle)
	toolState.OriginalPmxNeckLengthEdit.SetValue(sizingSet.OriginalPmxNeckLength)
	toolState.OriginalPmxNeckAngleEdit.SetValue(sizingSet.OriginalPmxNeckAngle)
	toolState.OriginalPmxHeadLengthEdit.SetValue(sizingSet.OriginalPmxHeadLength)
	toolState.OriginalPmxShoulderLengthEdit.SetValue(sizingSet.OriginalPmxShoulderLength)
	toolState.OriginalPmxShoulderAngleEdit.SetValue(sizingSet.OriginalPmxShoulderAngle)
	toolState.OriginalPmxArmLengthEdit.SetValue(sizingSet.OriginalPmxArmLength)
	toolState.OriginalPmxArmAngleEdit.SetValue(sizingSet.OriginalPmxArmAngle)
	toolState.OriginalPmxElbowLengthEdit.SetValue(sizingSet.OriginalPmxElbowLength)
	toolState.OriginalPmxElbowAngleEdit.SetValue(sizingSet.OriginalPmxElbowAngle)
	toolState.OriginalPmxWristLengthEdit.SetValue(sizingSet.OriginalPmxWristLength)
	toolState.OriginalPmxWristAngleEdit.SetValue(sizingSet.OriginalPmxWristAngle)
	toolState.OriginalPmxLowerLengthEdit.SetValue(sizingSet.OriginalPmxLowerLength)
	toolState.OriginalPmxLowerAngleEdit.SetValue(sizingSet.OriginalPmxLowerAngle)
	toolState.OriginalPmxLegWidthEdit.SetValue(sizingSet.OriginalPmxLegWidth)
	toolState.OriginalPmxLegLengthEdit.SetValue(sizingSet.OriginalPmxLegLength)
	toolState.OriginalPmxLegAngleEdit.SetValue(sizingSet.OriginalPmxLegAngle)
	toolState.OriginalPmxKneeLengthEdit.SetValue(sizingSet.OriginalPmxKneeLength)
	toolState.OriginalPmxKneeAngleEdit.SetValue(sizingSet.OriginalPmxKneeAngle)
	toolState.OriginalPmxAnkleLengthEdit.SetValue(sizingSet.OriginalPmxAnkleLength)

	return nil
}

func (toolState *ToolState) ResetSizingParameter() {
	toolState.SizingArmStanceCheck.SetChecked(false)
	toolState.SizingTranslateCheck.SetChecked(false)
}

func (toolState *ToolState) ResetOriginalPmxParameter() {
	toolState.OriginalPmxRatioEdit.SetValue(1.0)
	toolState.OriginalPmxUpperLengthEdit.SetValue(1.0)
	toolState.OriginalPmxUpperAngleEdit.SetValue(0.0)
	toolState.OriginalPmxUpper2LengthEdit.SetValue(1.0)
	toolState.OriginalPmxUpper2AngleEdit.SetValue(0.0)
	toolState.OriginalPmxNeckLengthEdit.SetValue(1.0)
	toolState.OriginalPmxNeckAngleEdit.SetValue(0.0)
	toolState.OriginalPmxHeadLengthEdit.SetValue(1.0)
	toolState.OriginalPmxShoulderLengthEdit.SetValue(1.0)
	toolState.OriginalPmxShoulderAngleEdit.SetValue(0.0)
	toolState.OriginalPmxArmLengthEdit.SetValue(1.0)
	toolState.OriginalPmxArmAngleEdit.SetValue(0.0)
	toolState.OriginalPmxElbowLengthEdit.SetValue(1.0)
	toolState.OriginalPmxElbowAngleEdit.SetValue(0.0)
	toolState.OriginalPmxWristLengthEdit.SetValue(1.0)
	toolState.OriginalPmxWristAngleEdit.SetValue(0.0)
	toolState.OriginalPmxLowerLengthEdit.SetValue(1.0)
	toolState.OriginalPmxLowerAngleEdit.SetValue(0.0)
	toolState.OriginalPmxLegWidthEdit.SetValue(1.0)
	toolState.OriginalPmxLegLengthEdit.SetValue(1.0)
	toolState.OriginalPmxLegAngleEdit.SetValue(0.0)
	toolState.OriginalPmxKneeLengthEdit.SetValue(1.0)
	toolState.OriginalPmxKneeAngleEdit.SetValue(0.0)
	toolState.OriginalPmxAnkleLengthEdit.SetValue(1.0)
}

// 素体モデルの編集パラメーターの有効/無効を設定
func (toolState *ToolState) SetOriginalPmxParameterEnabled(enabled bool) {
	toolState.OriginalPmxRatioEdit.SetEnabled(enabled)
	toolState.OriginalPmxUpperLengthEdit.SetEnabled(enabled)
	toolState.OriginalPmxUpperAngleEdit.SetEnabled(enabled)
	toolState.OriginalPmxUpper2LengthEdit.SetEnabled(enabled)
	toolState.OriginalPmxUpper2AngleEdit.SetEnabled(enabled)
	toolState.OriginalPmxNeckLengthEdit.SetEnabled(enabled)
	toolState.OriginalPmxNeckAngleEdit.SetEnabled(enabled)
	toolState.OriginalPmxHeadLengthEdit.SetEnabled(enabled)
	toolState.OriginalPmxShoulderLengthEdit.SetEnabled(enabled)
	toolState.OriginalPmxShoulderAngleEdit.SetEnabled(enabled)
	toolState.OriginalPmxArmLengthEdit.SetEnabled(enabled)
	toolState.OriginalPmxArmAngleEdit.SetEnabled(enabled)
	toolState.OriginalPmxElbowLengthEdit.SetEnabled(enabled)
	toolState.OriginalPmxElbowAngleEdit.SetEnabled(enabled)
	toolState.OriginalPmxWristLengthEdit.SetEnabled(enabled)
	toolState.OriginalPmxWristAngleEdit.SetEnabled(enabled)
	toolState.OriginalPmxLowerLengthEdit.SetEnabled(enabled)
	toolState.OriginalPmxLowerAngleEdit.SetEnabled(enabled)
	toolState.OriginalPmxLegWidthEdit.SetEnabled(enabled)
	toolState.OriginalPmxLegLengthEdit.SetEnabled(enabled)
	toolState.OriginalPmxLegAngleEdit.SetEnabled(enabled)
	toolState.OriginalPmxKneeLengthEdit.SetEnabled(enabled)
	toolState.OriginalPmxKneeAngleEdit.SetEnabled(enabled)
	toolState.OriginalPmxAnkleLengthEdit.SetEnabled(enabled)
}

func (toolState *ToolState) OriginalPmxParameterEnabled() bool {
	return toolState.OriginalPmxRatioEdit.Enabled()
}

func (toolState *ToolState) onPlay(playing bool) {
	toolState.OriginalVmdPicker.SetEnabled(!playing)
	toolState.OriginalPmxPicker.SetEnabled(!playing)
	toolState.SizingPmxPicker.SetEnabled(!playing)
	toolState.OutputVmdPicker.SetEnabled(!playing)
	toolState.SetOriginalPmxParameterEnabled(!playing && toolState.IsOriginalJson())
}

func (toolState *ToolState) IsOriginalJson() bool {
	return strings.HasSuffix(strings.ToLower(toolState.OriginalPmxPicker.GetPath()), ".json")
}
