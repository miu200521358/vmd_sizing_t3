package ui

import (
	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/animation"
	"github.com/miu200521358/mlib_go/pkg/interface/controller"
	"github.com/miu200521358/mlib_go/pkg/interface/controller/widget"
	"github.com/miu200521358/mlib_go/pkg/mutils"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/walk/pkg/walk"
)

func newSizingTab(controlWindow *controller.ControlWindow, toolState *ToolState) {
	toolState.SizingTab = widget.NewMTabPage(mi18n.T("サイジング"))
	controlWindow.AddTabPage(toolState.SizingTab.TabPage)

	toolState.SizingTab.SetLayout(walk.NewVBoxLayout())

	// ヘッダ
	{
		headerComposite, err := walk.NewComposite(toolState.SizingTab)
		if err != nil {
			widget.RaiseError(err)
		}
		headerComposite.SetLayout(walk.NewVBoxLayout())

		// ラベル
		label, err := walk.NewTextLabel(headerComposite)
		if err != nil {
			widget.RaiseError(err)
		}
		label.SetText(mi18n.T("サイジングTabLabel"))

		// ボタンBox
		buttonComposite, err := walk.NewComposite(headerComposite)
		if err != nil {
			widget.RaiseError(err)
		}
		buttonComposite.SetLayout(walk.NewHBoxLayout())
		walk.NewHSpacer(buttonComposite)

		// サイジングセット追加ボタン
		addButton, err := walk.NewPushButton(buttonComposite)
		if err != nil {
			widget.RaiseError(err)
		}
		addButton.SetMinMaxSize(walk.Size{Width: 130, Height: 30}, walk.Size{Width: 130, Height: 30})
		addButton.SetText(mi18n.T("サイジングセット追加"))
		addButton.Clicked().Attach(func() {
			toolState.addSizingSet()
		})

		// サイジングセット全削除ボタン
		deleteButton, err := walk.NewPushButton(buttonComposite)
		if err != nil {
			widget.RaiseError(err)
		}
		deleteButton.SetMinMaxSize(walk.Size{Width: 130, Height: 30}, walk.Size{Width: 130, Height: 30})
		deleteButton.SetText(mi18n.T("サイジングセット全削除"))
		deleteButton.Clicked().Attach(func() {
			toolState.resetSizingSet()
		})

		// サイジングセット設定読み込みボタン
		loadButton, err := walk.NewPushButton(buttonComposite)
		if err != nil {
			widget.RaiseError(err)
		}
		loadButton.SetMinMaxSize(walk.Size{Width: 130, Height: 30}, walk.Size{Width: 130, Height: 30})
		loadButton.SetText(mi18n.T("サイジングセット設定読込"))

		// プレイヤー
		player := widget.NewMotionPlayer(headerComposite, controlWindow)
		controlWindow.SetPlayer(player)

		walk.NewVSeparator(toolState.SizingTab)
	}

	{
		// スクロール
		scrollView, err := walk.NewScrollView(toolState.SizingTab)
		if err != nil {
			widget.RaiseError(err)
		}
		scrollView.SetScrollbars(true, true)
		scrollView.SetLayout(walk.NewHBoxLayout())
		scrollView.SetMinMaxSize(
			walk.Size{Width: toolState.ControlWindow.Config.ControlWindowSize.Width / 2, Height: 45},
			walk.Size{Width: toolState.ControlWindow.Config.ControlWindowSize.Width * 10, Height: 45},
		)

		// ナビゲーション用ツールバー
		toolState.NavToolBar, err = walk.NewToolBarWithOrientationAndButtonStyle(
			scrollView, walk.Horizontal, walk.ToolBarButtonTextOnly)
		if err != nil {
			widget.RaiseError(err)
		}
	}

	{
		// スクロール
		scrollView, err := walk.NewScrollView(toolState.SizingTab)
		if err != nil {
			widget.RaiseError(err)
		}
		scrollView.SetScrollbars(true, true)
		scrollView.SetLayout(walk.NewVBoxLayout())
		scrollView.SetMinMaxSize(
			walk.Size{Width: toolState.ControlWindow.Config.ControlWindowSize.Width / 2,
				Height: toolState.ControlWindow.Config.ControlWindowSize.Height / 2},
			walk.Size{Width: toolState.ControlWindow.Config.ControlWindowSize.Width * 10,
				Height: toolState.ControlWindow.Config.ControlWindowSize.Height * 10},
		)

		{
			toolState.OriginalVmdPicker = widget.NewVmdVpdReadFilePicker(
				controlWindow,
				scrollView.Composite(),
				"OriginalVmd",
				mi18n.T("サイジング対象モーション(Vmd/Vpd)"),
				mi18n.T("サイジング対象モーション(Vmd/Vpd)ファイルを選択してください"),
				mi18n.T("サイジング対象モーションの使い方"))

			toolState.OriginalVmdPicker.SetOnPathChanged(func(path string) {
				if data, err := toolState.OriginalVmdPicker.Load(); err == nil {
					// 出力パス設定
					outputPath := mutils.CreateOutputPath(path, "sizing")
					toolState.OutputVmdPicker.SetPath(outputPath)

					{
						// 元モデル用モーション
						motion := data.(*vmd.VmdMotion)
						animationState := animation.NewAnimationState(1, toolState.CurrentIndex)
						animationState.SetMotion(motion)
						controlWindow.SetAnimationState(animationState)
						controlWindow.UpdateMaxFrame(motion.MaxFrame())
					}

					{
						// サイジング用モーション
						motion := toolState.OriginalVmdPicker.LoadForce().(*vmd.VmdMotion)
						animationState := animation.NewAnimationState(0, toolState.CurrentIndex)
						animationState.SetMotion(motion)
						controlWindow.SetAnimationState(animationState)
						controlWindow.UpdateMaxFrame(motion.MaxFrame())
					}
				}
			})
		}

		{
			toolState.OriginalPmxPicker = widget.NewPmxReadFilePicker(
				controlWindow,
				scrollView.Composite(),
				"OriginalPmx",
				mi18n.T("モーション作成元モデル(Pmx)"),
				mi18n.T("モーション作成元モデルPmxファイルを選択してください"),
				mi18n.T("モーション作成元モデルの使い方"))

			toolState.OriginalPmxPicker.SetOnPathChanged(func(path string) {
				if data, err := toolState.OriginalPmxPicker.Load(); err == nil {
					model := data.(*pmx.PmxModel)
					animationState := animation.NewAnimationState(1, toolState.CurrentIndex)
					animationState.SetModel(model)
					controlWindow.SetAnimationState(animationState)
				}
			})
		}

		{
			toolState.SizingPmxPicker = widget.NewPmxReadFilePicker(
				controlWindow,
				scrollView.Composite(),
				"OriginalPmx",
				mi18n.T("サイジング先モデル(Pmx)"),
				mi18n.T("サイジング先モデルPmxファイルを選択してください"),
				mi18n.T("サイジング先モデルの使い方"))

			toolState.SizingPmxPicker.SetOnPathChanged(func(path string) {
				if data, err := toolState.SizingPmxPicker.Load(); err == nil {
					model := data.(*pmx.PmxModel)
					animationState := animation.NewAnimationState(0, toolState.CurrentIndex)
					animationState.SetModel(model)
					controlWindow.SetAnimationState(animationState)
				}
			})
		}

		{
			toolState.OutputVmdPicker = widget.NewVmdVpdReadFilePicker(
				controlWindow,
				scrollView.Composite(),
				"OriginalVmd",
				mi18n.T("出力モーション(Vmd)"),
				mi18n.T("出力モーション(Vmd)ファイルパスを指定してください"),
				mi18n.T("出力モーションの使い方"))
		}
	}

	// 保存ボタン
	{
		var err error
		toolState.SizingTabSaveButton, err = walk.NewPushButton(toolState.SizingTab)
		if err != nil {
			widget.RaiseError(err)
		}
		toolState.SizingTabSaveButton.SetText(mi18n.T("保存"))
		// toolState.SizingTabSaveButton.Clicked().Attach(toolState.onClickSizingTabOk)
	}
}
