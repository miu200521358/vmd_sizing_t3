package ui

import (
	"strings"

	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/interface/controller"
	"github.com/miu200521358/mlib_go/pkg/interface/controller/widget"
	"github.com/miu200521358/mlib_go/pkg/mutils"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/vmd_sizing_t3/pkg/usecase"
	"github.com/miu200521358/walk/pkg/declarative"
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
				mi18n.T("サイジング対象モーションツールチップ"),
				mi18n.T("サイジング対象モーションの使い方"))

			toolState.OriginalVmdPicker.SetOnPathChanged(func(path string) {
				if data, err := toolState.OriginalVmdPicker.Load(); err == nil {
					// 出力パス設定
					outputPath := mutils.CreateOutputPath(path, "sizing")
					toolState.OutputVmdPicker.SetPath(outputPath)

					// 元モデル用モーション
					motion := data.(*vmd.VmdMotion)
					// Fit用モーフ追加しておく
					motion = usecase.AddFitMorph(motion)
					// 強制更新用にハッシュ設定
					motion.SetRandHash()
					toolState.SizingSets[toolState.CurrentIndex].OriginalVmdPath = path
					toolState.SizingSets[toolState.CurrentIndex].OriginalVmd = motion
					toolState.SizingSets[toolState.CurrentIndex].OriginalVmdName = motion.Name()

					// サイジング先モデル用モーション
					sizingMotion := toolState.OriginalVmdPicker.LoadForce().(*vmd.VmdMotion)
					sizingMotion.SetRandHash()
					toolState.SizingSets[toolState.CurrentIndex].OutputVmdPath = outputPath
					toolState.SizingSets[toolState.CurrentIndex].OutputVmd = sizingMotion

					controlWindow.UpdateMaxFrame(motion.MaxFrame())
				} else {
					mlog.E(mi18n.T("読み込み失敗"), err)
				}
			})
		}

		{
			toolState.OriginalPmxPicker = widget.NewPmxJsonReadFilePicker(
				controlWindow,
				scrollView,
				"org_pmx",
				mi18n.T("モーション作成元モデル(Json/Pmx)"),
				mi18n.T("モーション作成元モデルツールチップ"),
				mi18n.T("モーション作成元モデルの使い方"))

			toolState.OriginalPmxPicker.SetOnPathChanged(func(path string) {
				if data, err := toolState.OriginalPmxPicker.Load(); err == nil {
					model := data.(*pmx.PmxModel)
					toolState.SetOriginalPmxParameterEnabled(false)

					// jsonから読み込んだ場合、モデル定義を適用して読み込みしなおす
					if strings.HasSuffix(strings.ToLower(toolState.OriginalPmxPicker.GetPath()), ".json") {
						originalModel, err := usecase.LoadOriginalPmx(model)
						if err != nil {
							mlog.E(mi18n.T("素体読み込み失敗"), err)
						} else {
							toolState.SizingSets[toolState.CurrentIndex].OriginalJsonPmx = model
							model = originalModel

							// 元モデル調整パラメータ有効化
							toolState.SetOriginalPmxParameterEnabled(true)
							toolState.OriginalPmxRatioEdit.SetValue(1.0)
							toolState.OriginalPmxArmStanceEdit.SetValue(0.0)
							toolState.OriginalPmxElbowStanceEdit.SetValue(0.0)
						}
					}

					model.SetRandHash()
					model.SetIndex(toolState.CurrentIndex)

					// 元モデル
					toolState.SizingSets[toolState.CurrentIndex].OriginalPmxPath = path
					toolState.SizingSets[toolState.CurrentIndex].OriginalPmx = model
					toolState.SizingSets[toolState.CurrentIndex].OriginalPmxName = model.Name()

					if toolState.SizingSets[toolState.CurrentIndex].OriginalVmd == nil {
						// モーション未設定の場合、空モーションを定義する
						toolState.SizingSets[toolState.CurrentIndex].OriginalVmd =
							usecase.AddFitMorph(vmd.NewVmdMotion(""))
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
				mi18n.T("サイジング先モデルツールチップ"),
				mi18n.T("サイジング先モデルの使い方"))

			toolState.SizingPmxPicker.SetOnPathChanged(func(path string) {
				if data, err := toolState.SizingPmxPicker.Load(); err == nil {
					model := data.(*pmx.PmxModel)
					model.SetRandHash()
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
				mi18n.T("出力モーションツールチップ"),
				mi18n.T("出力モーションの使い方"))
		}

		walk.NewVSeparator(scrollView)

		// 素体調整パラメーター
		{
			// タイトル
			titleLabel, err := walk.NewTextLabel(scrollView)
			if err != nil {
				widget.RaiseError(err)
			}
			titleLabel.SetText(mi18n.T("元モデル素体体格調整"))
			titleLabel.SetToolTipText(mi18n.T("元モデル素体体格調整説明"))
			titleLabel.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
				mlog.IL(mi18n.T("元モデル素体体格調整説明"))
			})

			composite := declarative.Composite{
				Layout:        declarative.Grid{Columns: 5},
				StretchFactor: 4,
				Children: []declarative.Widget{
					// 全体比率
					declarative.Label{Text: mi18n.T("元モデル素体体格全体比率"),
						ToolTipText: mi18n.T("元モデル素体体格全体比率説明"),
						OnMouseDown: func(x, y int, button walk.MouseButton) {
							mlog.IL(mi18n.T("元モデル素体体格全体比率説明"))
						}},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxRatioEdit,
						MinValue:           0.01,
						MaxValue:           10,
						Decimals:           2,
						Increment:          0.01,
						ColumnSpan:         4,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxRatio =
								toolState.OriginalPmxRatioEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					// 腕角度
					declarative.Label{Text: mi18n.T("元モデル素体スタンス補正"),
						ToolTipText: mi18n.T("元モデル素体スタンス補正説明"),
						OnMouseDown: func(x, y int, button walk.MouseButton) {
							mlog.IL(mi18n.T("元モデル素体スタンス補正説明"))
						}},
					declarative.Label{Text: mi18n.T("元モデル素体スタンス補正腕")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxArmStanceEdit,
						MinValue:           -90,
						MaxValue:           90,
						Decimals:           1,
						Increment:          1,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxArmStance =
								toolState.OriginalPmxArmStanceEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					// ひじ角度
					declarative.Label{Text: mi18n.T("元モデル素体スタンス補正ひじ")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxElbowStanceEdit,
						MinValue:           -90,
						MaxValue:           90,
						Decimals:           1,
						Increment:          1,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxElbowStance =
								toolState.OriginalPmxElbowStanceEdit.Value()
							remakeFitMorph(toolState)
						},
					},
				},
			}

			if err := composite.Create(declarative.NewBuilder(scrollView)); err != nil {
				widget.RaiseError(err)
			}
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

func remakeFitMorph(toolState *ToolState) {
	if toolState.SizingSets[toolState.CurrentIndex].OriginalPmx != nil &&
		toolState.SizingSets[toolState.CurrentIndex].OriginalJsonPmx != nil {
		// jsonモデル再読み込み
		toolState.SizingSets[toolState.CurrentIndex].OriginalJsonPmx =
			toolState.OriginalPmxPicker.LoadForce().(*pmx.PmxModel)
		// フィッティングモーフ再生成
		toolState.SizingSets[toolState.CurrentIndex].OriginalPmx = usecase.RemakeFitMorph(
			toolState.SizingSets[toolState.CurrentIndex].OriginalPmx,
			toolState.SizingSets[toolState.CurrentIndex].OriginalJsonPmx,
			toolState.SizingSets[toolState.CurrentIndex],
		)
		// 強制更新用にハッシュ設定
		toolState.SizingSets[toolState.CurrentIndex].OriginalPmx.SetRandHash()
	}
}
