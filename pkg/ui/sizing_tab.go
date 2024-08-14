package ui

import (
	"fmt"
	"math/rand"

	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/interface/controller"
	"github.com/miu200521358/mlib_go/pkg/interface/controller/widget"
	"github.com/miu200521358/mlib_go/pkg/mutils"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
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
				scrollView,
				"vmd",
				mi18n.T("サイジング対象モーション(Vmd/Vpd)"),
				mi18n.T("サイジング対象モーション(Vmd/Vpd)ファイルを選択してください"),
				mi18n.T("サイジング対象モーションの使い方"))

			toolState.OriginalVmdPicker.SetOnPathChanged(func(path string) {
				if data, err := toolState.OriginalVmdPicker.Load(); err == nil {
					// 出力パス設定
					outputPath := mutils.CreateOutputPath(path, "sizing")
					toolState.OutputVmdPicker.SetPath(outputPath)

					// 元モデル用モーション
					motion := data.(*vmd.VmdMotion)
					motion.SetHash(fmt.Sprintf("%d", rand.Intn(10000)))
					toolState.SizingSets[toolState.CurrentIndex].OriginalVmdPath = path
					toolState.SizingSets[toolState.CurrentIndex].OriginalVmd = motion
					toolState.SizingSets[toolState.CurrentIndex].OriginalVmdName = motion.Name()

					// サイジング先モデル用モーション
					sizingMotion := toolState.OriginalVmdPicker.LoadForce().(*vmd.VmdMotion)
					sizingMotion.SetHash(fmt.Sprintf("%d", rand.Intn(10000)))
					toolState.SizingSets[toolState.CurrentIndex].OutputVmdPath = outputPath
					toolState.SizingSets[toolState.CurrentIndex].OutputVmd = sizingMotion

					controlWindow.UpdateMaxFrame(motion.MaxFrame())
				} else {
					mlog.E(mi18n.T("読み込み失敗"), err)
				}
			})
		}

		{
			toolState.OriginalPmxPicker = widget.NewPmxReadFilePicker(
				controlWindow,
				scrollView,
				"org_pmx",
				mi18n.T("モーション作成元モデル(Pmx)"),
				mi18n.T("モーション作成元モデルPmxファイルを選択してください"),
				mi18n.T("モーション作成元モデルの使い方"))

			toolState.OriginalPmxPicker.SetOnPathChanged(func(path string) {
				if data, err := toolState.OriginalPmxPicker.Load(); err == nil {
					model := data.(*pmx.PmxModel)
					model.SetHash(fmt.Sprintf("%d", rand.Intn(10000)))
					model.SetIndex(toolState.CurrentIndex)

					// 元モデル
					toolState.SizingSets[toolState.CurrentIndex].OriginalPmxPath = path
					toolState.SizingSets[toolState.CurrentIndex].OriginalPmx = model
					toolState.SizingSets[toolState.CurrentIndex].OriginalPmxName = model.Name()

					if toolState.SizingSets[toolState.CurrentIndex].OriginalVmd == nil {
						toolState.SizingSets[toolState.CurrentIndex].OriginalVmd = vmd.NewVmdMotion("")
					}
				} else {
					mlog.E(mi18n.T("読み込み失敗"), err)
				}
			})
		}

		{
			toolState.SizingPmxPicker = widget.NewPmxReadFilePicker(
				controlWindow,
				scrollView,
				"rep_pmx",
				mi18n.T("サイジング先モデル(Pmx)"),
				mi18n.T("サイジング先モデルPmxファイルを選択してください"),
				mi18n.T("サイジング先モデルの使い方"))

			toolState.SizingPmxPicker.SetOnPathChanged(func(path string) {
				if data, err := toolState.SizingPmxPicker.Load(); err == nil {
					model := data.(*pmx.PmxModel)
					model.SetHash(fmt.Sprintf("%d", rand.Intn(10000)))
					model.SetIndex(toolState.CurrentIndex)

					// サイジングモデル
					toolState.SizingSets[toolState.CurrentIndex].SizingPmxPath = path
					toolState.SizingSets[toolState.CurrentIndex].SizingPmx = model
					toolState.SizingSets[toolState.CurrentIndex].SizingPmxName = model.Name()

					if toolState.SizingSets[toolState.CurrentIndex].OutputVmd == nil {
						toolState.SizingSets[toolState.CurrentIndex].OutputVmd = vmd.NewVmdMotion("")
					}
				} else {
					mlog.E(mi18n.T("読み込み失敗"), err)
				}
			})
		}

		{
			toolState.OutputVmdPicker = widget.NewVmdSaveFilePicker(
				controlWindow,
				scrollView,
				mi18n.T("出力モーション(Vmd)"),
				mi18n.T("出力モーション(Vmd)ファイルパスを指定してください"),
				mi18n.T("出力モーションの使い方"))
		}
	}

	// ヘッダ
	{
		walk.NewVSeparator(toolState.SizingTab)

		playerComposite, err := walk.NewComposite(toolState.SizingTab)
		if err != nil {
			widget.RaiseError(err)
		}
		playerComposite.SetLayout(walk.NewVBoxLayout())

		// プレイヤー
		player := widget.NewMotionPlayer(playerComposite, controlWindow)
		controlWindow.SetPlayer(player)

		toolState.SizingTabSaveButton, err = walk.NewPushButton(playerComposite)
		if err != nil {
			widget.RaiseError(err)
		}
		toolState.SizingTabSaveButton.SetText(mi18n.T("保存"))
		// toolState.SizingTabSaveButton.Clicked().Attach(toolState.onClickSizingTabOk)
	}
}
