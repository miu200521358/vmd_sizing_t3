package ui

import (
	"fmt"
	"sync"

	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/repository"
	"github.com/miu200521358/mlib_go/pkg/interface/controller"
	"github.com/miu200521358/mlib_go/pkg/interface/controller/widget"
	"github.com/miu200521358/mlib_go/pkg/mutils"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/vmd_sizing_t3/pkg/domain"
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
		addButton.SetMinMaxSize(walk.Size{Width: 130, Height: 20}, walk.Size{Width: 130, Height: 20})
		addButton.SetText(mi18n.T("サイジングセット追加"))
		addButton.Clicked().Attach(func() {
			toolState.addSizingSet()
		})

		// サイジングセット全削除ボタン
		deleteButton, err := walk.NewPushButton(buttonComposite)
		if err != nil {
			widget.RaiseError(err)
		}
		deleteButton.SetMinMaxSize(walk.Size{Width: 130, Height: 20}, walk.Size{Width: 130, Height: 20})
		deleteButton.SetText(mi18n.T("サイジングセット全削除"))
		deleteButton.Clicked().Attach(func() {
			toolState.resetSizingSet()
		})

		// サイジングセット設定読み込みボタン
		loadButton, err := walk.NewPushButton(buttonComposite)
		if err != nil {
			widget.RaiseError(err)
		}
		loadButton.SetMinMaxSize(walk.Size{Width: 130, Height: 20}, walk.Size{Width: 130, Height: 20})
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
					if data == nil {
						toolState.OutputVmdPicker.ChangePath("")
						return
					}

					// 出力パス設定
					setOutputPath(toolState)

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
					sizingMotion.Clean()
					toolState.SizingSets[toolState.CurrentIndex].OutputVmd = sizingMotion
					toolState.ResetSizingCheck(false)

					controlWindow.UpdateMaxFrame(motion.MaxFrame())
					go execSizing(toolState)
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
					if data == nil {
						toolState.SizingSets[toolState.CurrentIndex].OriginalPmxPath = path
						toolState.SizingSets[toolState.CurrentIndex].OriginalPmx = nil
						toolState.SizingSets[toolState.CurrentIndex].OriginalPmxName = ""
						toolState.SizingSets[toolState.CurrentIndex].OriginalJsonPmx = nil
						return
					}

					model := data.(*pmx.PmxModel)
					toolState.SetOriginalPmxParameterEnabled(false)

					// jsonから読み込んだ場合、モデル定義を適用して読み込みしなおす
					if toolState.IsOriginalJson() {
						originalModel, err := usecase.LoadOriginalPmxByJson(model)
						if err != nil {
							mlog.E(mi18n.T("素体読み込み失敗"), err)
						} else {
							toolState.SizingSets[toolState.CurrentIndex].OriginalJsonPmx = model
							model = originalModel

							// 元モデル調整パラメータ有効化
							toolState.ResetOriginalPmxParameter()
							toolState.SetOriginalPmxParameterEnabled(true)
						}
					} else {
						originalModel, _, err := usecase.AdjustPmxForSizing(model, true)
						if err != nil {
							mlog.E(mi18n.T("素体読み込み失敗"), err)
							return
						} else {
							toolState.SizingSets[toolState.CurrentIndex].OriginalJsonPmx = nil
							model = originalModel
						}
					}

					model.SetRandHash()
					model.SetIndex(toolState.CurrentIndex)

					// 元モデル
					toolState.SizingSets[toolState.CurrentIndex].OriginalPmxPath = path
					toolState.SizingSets[toolState.CurrentIndex].OriginalPmx = model
					toolState.SizingSets[toolState.CurrentIndex].OriginalPmxName = model.Name()
					toolState.ResetSizingCheck(false)

					if !toolState.OriginalVmdPicker.Exists() {
						// モーション未設定の場合、空モーションを定義する
						toolState.SizingSets[toolState.CurrentIndex].OriginalVmd = vmd.NewVmdMotion("")
						toolState.SizingSets[toolState.CurrentIndex].OutputVmd = vmd.NewVmdMotion("")
					} else {
						// モーション設定済みの場合、出力VMDを読み直す
						toolState.SizingSets[toolState.CurrentIndex].OriginalVmd =
							toolState.OriginalVmdPicker.LoadForce().(*vmd.VmdMotion)
						toolState.SizingSets[toolState.CurrentIndex].OutputVmd =
							toolState.OriginalVmdPicker.LoadForce().(*vmd.VmdMotion)
					}

					// 出力パス設定
					setOutputPath(toolState)
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
					if data == nil {
						toolState.SizingSets[toolState.CurrentIndex].SizingPmxPath = path
						toolState.SizingSets[toolState.CurrentIndex].SizingPmx = nil
						toolState.SizingSets[toolState.CurrentIndex].SizingPmxName = ""

						return
					}

					model := data.(*pmx.PmxModel)
					sizingModel, addBoneNames, err := usecase.AdjustPmxForSizing(model, true)
					if err != nil {
						mlog.E(mi18n.T("素体読み込み失敗"), err)
						return
					}
					sizingModel.SetIndex(toolState.CurrentIndex)

					// サイジングモデル
					toolState.SizingSets[toolState.CurrentIndex].SizingPmxPath = path
					toolState.SizingSets[toolState.CurrentIndex].SizingPmx = sizingModel
					toolState.SizingSets[toolState.CurrentIndex].SizingPmxName = sizingModel.Name()
					toolState.SizingSets[toolState.CurrentIndex].SizingAddedBoneNames = addBoneNames
					toolState.ResetSizingCheck(false)

					isAdd := false
					if toolState.OriginalVmdPicker.Exists() {
						for _, boneName := range addBoneNames {
							nowSizingSet := toolState.SizingSets[toolState.CurrentIndex]
							if nowSizingSet.OriginalVmd.BoneFrames.ContainsActive(boneName) {
								isAdd = true
								break
							}
						}
					}

					if isAdd {
						mlog.I(mi18n.T("不足ボーンあり", map[string]interface{}{
							"No":           toolState.SizingSets[toolState.CurrentIndex].Index + 1,
							"addBoneNames": mutils.JoinSlice(addBoneNames)}))
					}

					// 出力モデル
					sizingModel.SetName(fmt.Sprintf("%s_sizing", sizingModel.Name()))
					toolState.SizingSets[toolState.CurrentIndex].OutputPmx = sizingModel
					toolState.SizingSets[toolState.CurrentIndex].OutputPmxPath =
						mutils.CreateOutputPath(path, "sizing")

					toolState.OutputPmxPicker.SetPath(toolState.SizingSets[toolState.CurrentIndex].OutputPmxPath)

					if !toolState.OriginalVmdPicker.Exists() {
						// モーション未設定の場合、空モーションを定義する
						toolState.SizingSets[toolState.CurrentIndex].OriginalVmd = vmd.NewVmdMotion("")
						toolState.SizingSets[toolState.CurrentIndex].OutputVmd = vmd.NewVmdMotion("")
					} else {
						// モーション設定済みの場合、出力VMDを読み直す
						toolState.SizingSets[toolState.CurrentIndex].OriginalVmd =
							toolState.OriginalVmdPicker.LoadForce().(*vmd.VmdMotion)
						toolState.SizingSets[toolState.CurrentIndex].OutputVmd =
							toolState.OriginalVmdPicker.LoadForce().(*vmd.VmdMotion)
					}

					// 出力パス設定
					setOutputPath(toolState)
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

		{
			toolState.OutputPmxPicker = widget.NewPmxSaveFilePicker(
				controlWindow,
				scrollView,
				mi18n.T("出力モデル(Pmx)"),
				mi18n.T("出力モデルツールチップ"),
				mi18n.T("出力モデルの使い方"))
		}

		walk.NewVSeparator(scrollView)

		// 一括オプション
		{
			headerComposite, err := walk.NewComposite(scrollView)
			if err != nil {
				widget.RaiseError(err)
			}
			headerComposite.SetLayout(walk.NewHBoxLayout())

			// タイトル
			titleLabel, err := walk.NewTextLabel(headerComposite)
			if err != nil {
				widget.RaiseError(err)
			}
			titleLabel.SetText(mi18n.T("サイジング一括オプション"))
			titleLabel.SetToolTipText(mi18n.T("サイジング一括オプション説明"))
			titleLabel.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
				mlog.IL(mi18n.T("サイジング一括オプション説明"))
			})

			// 即時反映
			toolState.AdoptSizingCheck, err = walk.NewCheckBox(headerComposite)
			if err != nil {
				widget.RaiseError(err)
			}
			toolState.AdoptSizingCheck.SetMinMaxSize(walk.Size{Width: 100, Height: 20}, walk.Size{Width: 100, Height: 20})
			toolState.AdoptSizingCheck.SetText(mi18n.T("即時反映"))
			toolState.AdoptSizingCheck.SetToolTipText(mi18n.T("即時反映説明"))
			toolState.AdoptSizingCheck.UpdateChecked(true)
			toolState.AdoptSizingCheck.CheckedChanged().Attach(func() {
				go execSizing(toolState)
			})

			composite := declarative.Composite{
				Layout: declarative.Grid{Columns: 3},
				Children: []declarative.Widget{
					// 全補正&最適化
					declarative.CheckBox{
						AssignTo: &toolState.SizingCleanAllCheck,
						OnCheckedChanged: func() {
							for _, sizingSet := range toolState.SizingSets {
								sizingSet.IsSizingCleanAll = toolState.SizingCleanAllCheck.Checked()

								sizingSet.IsSizingLeg = toolState.SizingCleanAllCheck.Checked()
								sizingSet.IsSizingUpper = toolState.SizingCleanAllCheck.Checked()
								sizingSet.IsSizingShoulder = toolState.SizingCleanAllCheck.Checked()
								sizingSet.IsSizingArmStance = toolState.SizingCleanAllCheck.Checked()
								sizingSet.IsSizingFingerStance = toolState.SizingCleanAllCheck.Checked()
								sizingSet.IsSizingArmTwist = toolState.SizingCleanAllCheck.Checked()

								sizingSet.IsCleanRoot = toolState.SizingCleanAllCheck.Checked()
								sizingSet.IsCleanCenter = toolState.SizingCleanAllCheck.Checked()
								sizingSet.IsCleanLegIkParent = toolState.SizingCleanAllCheck.Checked()
								sizingSet.IsCleanShoulderP = toolState.SizingCleanAllCheck.Checked()
								sizingSet.IsCleanArmIk = toolState.SizingCleanAllCheck.Checked()
								sizingSet.IsCleanGrip = toolState.SizingCleanAllCheck.Checked()
							}

							toolState.SizingLegCheck.UpdateChecked(toolState.SizingCleanAllCheck.Checked())
							toolState.SizingUpperCheck.UpdateChecked(toolState.SizingCleanAllCheck.Checked())
							toolState.SizingShoulderCheck.UpdateChecked(toolState.SizingCleanAllCheck.Checked())
							toolState.SizingArmStanceCheck.UpdateChecked(toolState.SizingCleanAllCheck.Checked())
							toolState.SizingFingerStanceCheck.UpdateChecked(toolState.SizingCleanAllCheck.Checked())
							toolState.SizingArmTwistCheck.UpdateChecked(toolState.SizingCleanAllCheck.Checked())

							toolState.CleanRootCheck.UpdateChecked(toolState.SizingCleanAllCheck.Checked())
							toolState.CleanCenterCheck.UpdateChecked(toolState.SizingCleanAllCheck.Checked())
							toolState.CleanLegIkParentCheck.UpdateChecked(toolState.SizingCleanAllCheck.Checked())
							toolState.CleanShoulderPCheck.UpdateChecked(toolState.SizingCleanAllCheck.Checked())
							toolState.CleanArmIkCheck.UpdateChecked(toolState.SizingCleanAllCheck.Checked())
							toolState.CleanGripCheck.UpdateChecked(toolState.SizingCleanAllCheck.Checked())

							go execSizing(toolState)

							// 出力パス設定
							setOutputPath(toolState)
						},
						MinSize:     declarative.Size{Width: 150, Height: 20},
						MaxSize:     declarative.Size{Width: 150, Height: 20},
						Text:        mi18n.T("全補正&最適化"),
						ToolTipText: mi18n.T("全補正&最適化説明"),
					},
					// 全補正
					declarative.CheckBox{
						AssignTo: &toolState.SizingAllCheck,
						OnCheckedChanged: func() {
							for _, sizingSet := range toolState.SizingSets {
								sizingSet.IsSizingAll = toolState.SizingAllCheck.Checked()

								sizingSet.IsSizingLeg = toolState.SizingAllCheck.Checked()
								sizingSet.IsSizingUpper = toolState.SizingAllCheck.Checked()
								sizingSet.IsSizingShoulder = toolState.SizingAllCheck.Checked()
								sizingSet.IsSizingArmStance = toolState.SizingAllCheck.Checked()
								sizingSet.IsSizingFingerStance = toolState.SizingAllCheck.Checked()
								sizingSet.IsSizingArmTwist = toolState.SizingAllCheck.Checked()
							}

							toolState.SizingLegCheck.UpdateChecked(toolState.SizingAllCheck.Checked())
							toolState.SizingUpperCheck.UpdateChecked(toolState.SizingAllCheck.Checked())
							toolState.SizingShoulderCheck.UpdateChecked(toolState.SizingAllCheck.Checked())
							toolState.SizingArmStanceCheck.UpdateChecked(toolState.SizingAllCheck.Checked())
							toolState.SizingFingerStanceCheck.UpdateChecked(toolState.SizingAllCheck.Checked())
							toolState.SizingArmTwistCheck.UpdateChecked(toolState.SizingAllCheck.Checked())

							go execSizing(toolState)

							// 出力パス設定
							setOutputPath(toolState)
						},
						MinSize:     declarative.Size{Width: 150, Height: 20},
						MaxSize:     declarative.Size{Width: 150, Height: 20},
						Text:        mi18n.T("全補正"),
						ToolTipText: mi18n.T("全補正説明"),
					},
					// 全最適化
					declarative.CheckBox{
						AssignTo: &toolState.CleanAllCheck,
						OnCheckedChanged: func() {
							for _, sizingSet := range toolState.SizingSets {
								sizingSet.IsCleanAll = toolState.CleanAllCheck.Checked()

								sizingSet.IsCleanRoot = toolState.CleanAllCheck.Checked()
								sizingSet.IsCleanCenter = toolState.CleanAllCheck.Checked()
								sizingSet.IsCleanLegIkParent = toolState.CleanAllCheck.Checked()
								sizingSet.IsCleanShoulderP = toolState.CleanAllCheck.Checked()
								sizingSet.IsCleanArmIk = toolState.CleanAllCheck.Checked()
								sizingSet.IsCleanGrip = toolState.CleanAllCheck.Checked()
							}

							toolState.CleanRootCheck.UpdateChecked(toolState.CleanAllCheck.Checked())
							toolState.CleanCenterCheck.UpdateChecked(toolState.CleanAllCheck.Checked())
							toolState.CleanLegIkParentCheck.UpdateChecked(toolState.CleanAllCheck.Checked())
							toolState.CleanShoulderPCheck.UpdateChecked(toolState.CleanAllCheck.Checked())
							toolState.CleanArmIkCheck.UpdateChecked(toolState.CleanAllCheck.Checked())
							toolState.CleanGripCheck.UpdateChecked(toolState.CleanAllCheck.Checked())

							go execSizing(toolState)

							// 出力パス設定
							setOutputPath(toolState)
						},
						MinSize:     declarative.Size{Width: 150, Height: 20},
						MaxSize:     declarative.Size{Width: 150, Height: 20},
						Text:        mi18n.T("全最適化"),
						ToolTipText: mi18n.T("全最適化説明"),
					},
				},
			}

			if err := composite.Create(declarative.NewBuilder(scrollView)); err != nil {
				widget.RaiseError(err)
			}
		}

		walk.NewVSeparator(scrollView)

		// サイジングオプション
		{
			headerComposite, err := walk.NewComposite(scrollView)
			if err != nil {
				widget.RaiseError(err)
			}
			headerComposite.SetLayout(walk.NewHBoxLayout())

			// タイトル
			titleLabel, err := walk.NewTextLabel(headerComposite)
			if err != nil {
				widget.RaiseError(err)
			}
			titleLabel.SetText(mi18n.T("サイジングオプション"))
			titleLabel.SetToolTipText(mi18n.T("サイジングオプション説明"))
			titleLabel.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
				mlog.IL(mi18n.T("サイジングオプション説明"))
			})

			composite := declarative.Composite{
				Layout: declarative.Grid{Columns: 3},
				Children: []declarative.Widget{
					// 足補正
					declarative.CheckBox{
						AssignTo: &toolState.SizingLegCheck,
						OnCheckedChanged: func() {
							// 足補正は全セットに適用する
							for _, sizingSet := range toolState.SizingSets {
								sizingSet.IsSizingLeg = toolState.SizingLegCheck.Checked()

								sizingSet.IsCleanRoot = toolState.SizingLegCheck.Checked()
								sizingSet.IsCleanCenter = toolState.SizingLegCheck.Checked()
								sizingSet.IsCleanLegIkParent = toolState.SizingLegCheck.Checked()
							}

							toolState.CleanRootCheck.UpdateChecked(
								toolState.SizingSets[toolState.CurrentIndex].IsCleanRoot)
							toolState.CleanCenterCheck.UpdateChecked(
								toolState.SizingSets[toolState.CurrentIndex].IsCleanCenter)
							toolState.CleanLegIkParentCheck.UpdateChecked(
								toolState.SizingSets[toolState.CurrentIndex].IsCleanLegIkParent)

							go execSizing(toolState)
							// 出力パス設定
							setOutputPath(toolState)
						},
						MinSize:     declarative.Size{Width: 150, Height: 20},
						MaxSize:     declarative.Size{Width: 150, Height: 20},
						Text:        mi18n.T("足補正"),
						ToolTipText: mi18n.T("足補正説明"),
					},
					// 上半身補正
					declarative.CheckBox{
						AssignTo: &toolState.SizingUpperCheck,
						OnCheckedChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].IsSizingUpper =
								toolState.SizingUpperCheck.Checked()

							go execSizing(toolState)
							// 出力パス設定
							setOutputPath(toolState)
						},
						MinSize:     declarative.Size{Width: 150, Height: 20},
						MaxSize:     declarative.Size{Width: 150, Height: 20},
						Text:        mi18n.T("上半身補正"),
						ToolTipText: mi18n.T("上半身補正説明"),
					},
					// 肩補正
					declarative.CheckBox{
						AssignTo: &toolState.SizingShoulderCheck,
						OnCheckedChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].IsSizingShoulder =
								toolState.SizingShoulderCheck.Checked()

							toolState.SizingSets[toolState.CurrentIndex].IsCleanShoulderP =
								toolState.SizingShoulderCheck.Checked()
							toolState.CleanShoulderPCheck.UpdateChecked(
								toolState.SizingSets[toolState.CurrentIndex].IsCleanShoulderP)

							go execSizing(toolState)
							// 出力パス設定
							setOutputPath(toolState)
						},
						MinSize:     declarative.Size{Width: 150, Height: 20},
						MaxSize:     declarative.Size{Width: 150, Height: 20},
						Text:        mi18n.T("肩補正"),
						ToolTipText: mi18n.T("肩補正説明"),
					},
					// 腕スタンス補正
					declarative.CheckBox{
						AssignTo: &toolState.SizingArmStanceCheck,
						OnCheckedChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].IsSizingArmStance =
								toolState.SizingArmStanceCheck.Checked()

							toolState.SizingSets[toolState.CurrentIndex].IsCleanArmIk =
								toolState.SizingArmStanceCheck.Checked()
							toolState.CleanArmIkCheck.UpdateChecked(
								toolState.SizingSets[toolState.CurrentIndex].IsCleanArmIk)

							go execSizing(toolState)
							// 出力パス設定
							setOutputPath(toolState)
						},
						MinSize:     declarative.Size{Width: 150, Height: 20},
						MaxSize:     declarative.Size{Width: 150, Height: 20},
						Text:        mi18n.T("腕スタンス補正"),
						ToolTipText: mi18n.T("腕スタンス補正説明"),
					},
					// 指スタンス補正
					declarative.CheckBox{
						AssignTo: &toolState.SizingFingerStanceCheck,
						OnCheckedChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].IsSizingFingerStance =
								toolState.SizingFingerStanceCheck.Checked()

							toolState.SizingSets[toolState.CurrentIndex].IsCleanGrip =
								toolState.SizingFingerStanceCheck.Checked()
							toolState.CleanGripCheck.UpdateChecked(
								toolState.SizingSets[toolState.CurrentIndex].IsCleanGrip)

							go execSizing(toolState)
							// 出力パス設定
							setOutputPath(toolState)
						},
						MinSize:     declarative.Size{Width: 150, Height: 20},
						MaxSize:     declarative.Size{Width: 150, Height: 20},
						Text:        mi18n.T("指スタンス補正"),
						ToolTipText: mi18n.T("指スタンス補正説明"),
					},
					// 捩り補正
					declarative.CheckBox{
						AssignTo: &toolState.SizingArmTwistCheck,
						OnCheckedChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].IsSizingArmTwist =
								toolState.SizingArmTwistCheck.Checked()

							toolState.SizingSets[toolState.CurrentIndex].IsCleanArmIk =
								toolState.SizingArmTwistCheck.Checked()
							toolState.CleanArmIkCheck.UpdateChecked(
								toolState.SizingSets[toolState.CurrentIndex].IsCleanArmIk)

							go execSizing(toolState)
							// 出力パス設定
							setOutputPath(toolState)
						},
						MinSize:     declarative.Size{Width: 150, Height: 20},
						MaxSize:     declarative.Size{Width: 150, Height: 20},
						Text:        mi18n.T("捩り補正"),
						ToolTipText: mi18n.T("捩り補正説明"),
					},
				},
			}

			if err := composite.Create(declarative.NewBuilder(scrollView)); err != nil {
				widget.RaiseError(err)
			}
		}

		walk.NewVSeparator(scrollView)

		// 最適化オプション
		{
			// ボタンBox
			buttonComposite, err := walk.NewComposite(scrollView)
			if err != nil {
				widget.RaiseError(err)
			}
			buttonComposite.SetLayout(walk.NewHBoxLayout())

			// タイトル
			titleLabel, err := walk.NewTextLabel(buttonComposite)
			if err != nil {
				widget.RaiseError(err)
			}
			titleLabel.SetText(mi18n.T("最適化オプション"))
			titleLabel.SetToolTipText(mi18n.T("最適化オプション説明"))
			titleLabel.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
				mlog.IL(mi18n.T("最適化オプション説明"))
			})

			composite := declarative.Composite{
				Layout: declarative.Grid{Columns: 3},
				Children: []declarative.Widget{
					// 全親最適化
					declarative.CheckBox{
						AssignTo: &toolState.CleanRootCheck,
						OnCheckedChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].IsCleanRoot =
								toolState.CleanRootCheck.Checked()
							go execSizing(toolState)
							// 出力パス設定
							setOutputPath(toolState)
						},
						MinSize:     declarative.Size{Width: 150, Height: 20},
						MaxSize:     declarative.Size{Width: 150, Height: 20},
						Text:        mi18n.T("全ての親最適化"),
						ToolTipText: mi18n.T("全ての親最適化説明"),
					},
					// センター最適化
					declarative.CheckBox{
						AssignTo: &toolState.CleanCenterCheck,
						OnCheckedChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].IsCleanRoot =
								toolState.SizingSets[toolState.CurrentIndex].IsCleanRoot ||
									toolState.CleanCenterCheck.Checked()
							toolState.SizingSets[toolState.CurrentIndex].IsCleanCenter =
								toolState.CleanCenterCheck.Checked()
							toolState.CleanRootCheck.UpdateChecked(
								toolState.SizingSets[toolState.CurrentIndex].IsCleanRoot)

							go execSizing(toolState)
							// 出力パス設定
							setOutputPath(toolState)
						},
						MinSize:     declarative.Size{Width: 150, Height: 20},
						MaxSize:     declarative.Size{Width: 150, Height: 20},
						Text:        mi18n.T("センター最適化"),
						ToolTipText: mi18n.T("センター最適化説明"),
					},
					// 足IK親最適化
					declarative.CheckBox{
						AssignTo: &toolState.CleanLegIkParentCheck,
						OnCheckedChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].IsCleanRoot =
								toolState.SizingSets[toolState.CurrentIndex].IsCleanRoot ||
									toolState.CleanLegIkParentCheck.Checked()
							toolState.SizingSets[toolState.CurrentIndex].IsCleanCenter =
								toolState.SizingSets[toolState.CurrentIndex].IsCleanCenter ||
									toolState.CleanLegIkParentCheck.Checked()
							toolState.SizingSets[toolState.CurrentIndex].IsCleanLegIkParent =
								toolState.CleanLegIkParentCheck.Checked()

							toolState.CleanRootCheck.UpdateChecked(
								toolState.SizingSets[toolState.CurrentIndex].IsCleanRoot)
							toolState.CleanCenterCheck.UpdateChecked(
								toolState.SizingSets[toolState.CurrentIndex].IsCleanCenter)

							go execSizing(toolState)
							// 出力パス設定
							setOutputPath(toolState)
						},
						MinSize:     declarative.Size{Width: 150, Height: 20},
						MaxSize:     declarative.Size{Width: 150, Height: 20},
						Text:        mi18n.T("足IK親最適化"),
						ToolTipText: mi18n.T("足IK親最適化説明"),
					},
					// 肩P最適化
					declarative.CheckBox{
						AssignTo: &toolState.CleanShoulderPCheck,
						OnCheckedChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].IsCleanShoulderP =
								toolState.CleanShoulderPCheck.Checked()

							go execSizing(toolState)
							// 出力パス設定
							setOutputPath(toolState)
						},
						MinSize:     declarative.Size{Width: 150, Height: 20},
						MaxSize:     declarative.Size{Width: 150, Height: 20},
						Text:        mi18n.T("肩P最適化"),
						ToolTipText: mi18n.T("肩P最適化説明"),
					},
					// 腕IK最適化
					declarative.CheckBox{
						AssignTo: &toolState.CleanArmIkCheck,
						OnCheckedChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].IsCleanArmIk =
								toolState.CleanArmIkCheck.Checked()

							go execSizing(toolState)
							// 出力パス設定
							setOutputPath(toolState)
						},
						MinSize:     declarative.Size{Width: 150, Height: 20},
						MaxSize:     declarative.Size{Width: 150, Height: 20},
						Text:        mi18n.T("腕IK最適化"),
						ToolTipText: mi18n.T("腕IK最適化説明"),
					},
					// 握り最適化
					declarative.CheckBox{
						AssignTo: &toolState.CleanGripCheck,
						OnCheckedChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].IsCleanGrip =
								toolState.CleanGripCheck.Checked()

							go execSizing(toolState)
							// 出力パス設定
							setOutputPath(toolState)
						},
						MinSize:     declarative.Size{Width: 150, Height: 20},
						MaxSize:     declarative.Size{Width: 150, Height: 20},
						Text:        mi18n.T("握り最適化"),
						ToolTipText: mi18n.T("握り最適化説明"),
					},
				},
			}

			if err := composite.Create(declarative.NewBuilder(scrollView)); err != nil {
				widget.RaiseError(err)
			}
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
				Layout:        declarative.Grid{Columns: 7},
				StretchFactor: 6,
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
						ColumnSpan:         6,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxRatio =
								toolState.OriginalPmxRatioEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					// 上半身
					declarative.Label{Text: mi18n.T("元モデル素体上半身補正"),
						ToolTipText: mi18n.T("元モデル素体上半身補正説明"),
						OnMouseDown: func(x, y int, button walk.MouseButton) {
							mlog.IL(mi18n.T("元モデル素体上半身補正説明"))
						}},
					declarative.Label{Text: mi18n.T("長さ")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxUpperLengthEdit,
						MinValue:           0.01,
						MaxValue:           10,
						Decimals:           2,
						Increment:          0.01,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxUpperLength =
								toolState.OriginalPmxUpperLengthEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					declarative.HSpacer{ColumnSpan: 2},
					declarative.Label{Text: mi18n.T("角度")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxUpperAngleEdit,
						MinValue:           -90,
						MaxValue:           90,
						Decimals:           1,
						Increment:          1,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxUpperAngle =
								toolState.OriginalPmxUpperAngleEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					// 上半身2
					declarative.Label{Text: mi18n.T("元モデル素体上半身2補正"),
						ToolTipText: mi18n.T("元モデル素体上半身2補正説明"),
						OnMouseDown: func(x, y int, button walk.MouseButton) {
							mlog.IL(mi18n.T("元モデル素体上半身2補正説明"))
						}},
					declarative.Label{Text: mi18n.T("長さ")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxUpper2LengthEdit,
						MinValue:           0.01,
						MaxValue:           10,
						Decimals:           2,
						Increment:          0.01,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxUpper2Length =
								toolState.OriginalPmxUpper2LengthEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					declarative.HSpacer{ColumnSpan: 2},
					declarative.Label{Text: mi18n.T("角度")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxUpper2AngleEdit,
						MinValue:           -90,
						MaxValue:           90,
						Decimals:           1,
						Increment:          1,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxUpper2Angle =
								toolState.OriginalPmxUpper2AngleEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					// 首
					declarative.Label{Text: mi18n.T("元モデル素体首補正"),
						ToolTipText: mi18n.T("元モデル素体首補正説明"),
						OnMouseDown: func(x, y int, button walk.MouseButton) {
							mlog.IL(mi18n.T("元モデル素体首補正説明"))
						}},
					declarative.Label{Text: mi18n.T("長さ")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxNeckLengthEdit,
						MinValue:           0.01,
						MaxValue:           10,
						Decimals:           2,
						Increment:          0.01,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxNeckLength =
								toolState.OriginalPmxNeckLengthEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					declarative.HSpacer{ColumnSpan: 2},
					declarative.Label{Text: mi18n.T("角度")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxNeckAngleEdit,
						MinValue:           -90,
						MaxValue:           90,
						Decimals:           1,
						Increment:          1,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxNeckAngle =
								toolState.OriginalPmxNeckAngleEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					// 頭
					declarative.Label{Text: mi18n.T("元モデル素体頭補正"),
						ToolTipText: mi18n.T("元モデル素体頭補正説明"),
						OnMouseDown: func(x, y int, button walk.MouseButton) {
							mlog.IL(mi18n.T("元モデル素体頭補正説明"))
						}},
					declarative.Label{Text: mi18n.T("長さ")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxHeadLengthEdit,
						MinValue:           0.01,
						MaxValue:           10,
						Decimals:           2,
						Increment:          0.01,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxHeadLength =
								toolState.OriginalPmxHeadLengthEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					declarative.HSpacer{ColumnSpan: 4},
					// 肩
					declarative.Label{Text: mi18n.T("元モデル素体肩補正"),
						ToolTipText: mi18n.T("元モデル素体肩補正説明"),
						OnMouseDown: func(x, y int, button walk.MouseButton) {
							mlog.IL(mi18n.T("元モデル素体肩補正説明"))
						}},
					declarative.Label{Text: mi18n.T("長さ")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxShoulderLengthEdit,
						MinValue:           0.01,
						MaxValue:           10,
						Decimals:           2,
						Increment:          0.01,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxShoulderLength =
								toolState.OriginalPmxShoulderLengthEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					declarative.HSpacer{ColumnSpan: 2},
					declarative.Label{Text: mi18n.T("角度")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxShoulderAngleEdit,
						MinValue:           -90,
						MaxValue:           90,
						Decimals:           1,
						Increment:          1,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxShoulderAngle =
								toolState.OriginalPmxShoulderAngleEdit.Value()
							remakeFitMorph(toolState)
						},
					},

					// 腕
					declarative.Label{Text: mi18n.T("元モデル素体腕補正"),
						ToolTipText: mi18n.T("元モデル素体腕補正説明"),
						OnMouseDown: func(x, y int, button walk.MouseButton) {
							mlog.IL(mi18n.T("元モデル素体腕補正説明"))
						}},
					declarative.Label{Text: mi18n.T("長さ")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxArmLengthEdit,
						MinValue:           0.01,
						MaxValue:           10,
						Decimals:           2,
						Increment:          0.01,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxArmLength =
								toolState.OriginalPmxArmLengthEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					declarative.HSpacer{ColumnSpan: 2},
					declarative.Label{Text: mi18n.T("角度")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxArmAngleEdit,
						MinValue:           -90,
						MaxValue:           90,
						Decimals:           1,
						Increment:          1,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxArmAngle =
								toolState.OriginalPmxArmAngleEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					// ひじ
					declarative.Label{Text: mi18n.T("元モデル素体ひじ補正"),
						ToolTipText: mi18n.T("元モデル素体ひじ補正説明"),
						OnMouseDown: func(x, y int, button walk.MouseButton) {
							mlog.IL(mi18n.T("元モデル素体ひじ補正説明"))
						}},
					declarative.Label{Text: mi18n.T("長さ")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxElbowLengthEdit,
						MinValue:           0.01,
						MaxValue:           10,
						Decimals:           2,
						Increment:          0.01,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxElbowLength =
								toolState.OriginalPmxElbowLengthEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					declarative.HSpacer{ColumnSpan: 2},
					declarative.Label{Text: mi18n.T("角度")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxElbowAngleEdit,
						MinValue:           -90,
						MaxValue:           90,
						Decimals:           1,
						Increment:          1,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxElbowAngle =
								toolState.OriginalPmxElbowAngleEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					// 手首
					declarative.Label{Text: mi18n.T("元モデル素体手首補正"),
						ToolTipText: mi18n.T("元モデル素体手首補正説明"),
						OnMouseDown: func(x, y int, button walk.MouseButton) {
							mlog.IL(mi18n.T("元モデル素体手首補正説明"))
						}},
					declarative.Label{Text: mi18n.T("長さ")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxWristLengthEdit,
						MinValue:           0.01,
						MaxValue:           10,
						Decimals:           2,
						Increment:          0.01,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxWristLength =
								toolState.OriginalPmxWristLengthEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					declarative.HSpacer{ColumnSpan: 2},
					declarative.Label{Text: mi18n.T("角度")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxWristAngleEdit,
						MinValue:           -90,
						MaxValue:           90,
						Decimals:           1,
						Increment:          1,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxWristAngle =
								toolState.OriginalPmxWristAngleEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					// 下半身
					declarative.Label{Text: mi18n.T("元モデル素体下半身補正"),
						ToolTipText: mi18n.T("元モデル素体下半身補正説明"),
						OnMouseDown: func(x, y int, button walk.MouseButton) {
							mlog.IL(mi18n.T("元モデル素体下半身補正説明"))
						}},
					declarative.Label{Text: mi18n.T("長さ")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxLowerLengthEdit,
						MinValue:           0.01,
						MaxValue:           10,
						Decimals:           2,
						Increment:          0.01,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxLowerLength =
								toolState.OriginalPmxLowerLengthEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					declarative.HSpacer{ColumnSpan: 2},
					declarative.Label{Text: mi18n.T("角度")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxLowerAngleEdit,
						MinValue:           -90,
						MaxValue:           90,
						Decimals:           1,
						Increment:          1,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxLowerAngle =
								toolState.OriginalPmxLowerAngleEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					// 足
					declarative.Label{Text: mi18n.T("元モデル素体足補正"),
						ToolTipText: mi18n.T("元モデル素体足補正説明"),
						OnMouseDown: func(x, y int, button walk.MouseButton) {
							mlog.IL(mi18n.T("元モデル素体足補正説明"))
						}},
					declarative.Label{Text: mi18n.T("長さ")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxLegLengthEdit,
						MinValue:           0.01,
						MaxValue:           10,
						Decimals:           2,
						Increment:          0.01,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxLegLength =
								toolState.OriginalPmxLegLengthEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					declarative.Label{Text: mi18n.T("横幅")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxLegWidthEdit,
						MinValue:           0.01,
						MaxValue:           10,
						Decimals:           2,
						Increment:          0.01,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxLegWidth =
								toolState.OriginalPmxLegWidthEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					declarative.Label{Text: mi18n.T("角度")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxLegAngleEdit,
						MinValue:           -90,
						MaxValue:           90,
						Decimals:           1,
						Increment:          1,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxLegAngle =
								toolState.OriginalPmxLegAngleEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					// ひざ
					declarative.Label{Text: mi18n.T("元モデル素体ひざ補正"),
						ToolTipText: mi18n.T("元モデル素体ひざ補正説明"),
						OnMouseDown: func(x, y int, button walk.MouseButton) {
							mlog.IL(mi18n.T("元モデル素体ひざ補正説明"))
						}},
					declarative.Label{Text: mi18n.T("長さ")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxKneeLengthEdit,
						MinValue:           0.01,
						MaxValue:           10,
						Decimals:           2,
						Increment:          0.01,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxKneeLength =
								toolState.OriginalPmxKneeLengthEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					declarative.HSpacer{ColumnSpan: 2},
					declarative.Label{Text: mi18n.T("角度")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxKneeAngleEdit,
						MinValue:           -90,
						MaxValue:           90,
						Decimals:           1,
						Increment:          1,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxKneeAngle =
								toolState.OriginalPmxKneeAngleEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					// 足首
					declarative.Label{Text: mi18n.T("元モデル素体足首補正"),
						ToolTipText: mi18n.T("元モデル素体足首補正説明"),
						OnMouseDown: func(x, y int, button walk.MouseButton) {
							mlog.IL(mi18n.T("元モデル素体足首補正説明"))
						}},
					declarative.Label{Text: mi18n.T("長さ")},
					declarative.NumberEdit{
						AssignTo:           &toolState.OriginalPmxAnkleLengthEdit,
						MinValue:           0.01,
						MaxValue:           10,
						Decimals:           2,
						Increment:          0.01,
						SpinButtonsVisible: true,
						OnValueChanged: func() {
							toolState.SizingSets[toolState.CurrentIndex].OriginalPmxAnkleLength =
								toolState.OriginalPmxAnkleLengthEdit.Value()
							remakeFitMorph(toolState)
						},
					},
					declarative.HSpacer{ColumnSpan: 4},
				},
			}

			if err := composite.Create(declarative.NewBuilder(scrollView)); err != nil {
				widget.RaiseError(err)
			}
		}
	}

	// フッター
	{
		walk.NewVSeparator(toolState.SizingTab)

		playerComposite, err := walk.NewComposite(toolState.SizingTab)
		if err != nil {
			widget.RaiseError(err)
		}
		playerComposite.SetLayout(walk.NewVBoxLayout())

		// プレイヤー
		player := widget.NewMotionPlayer(playerComposite, controlWindow)
		player.SetOnTriggerPlay(func(playing bool) { toolState.onPlay(playing) })
		controlWindow.SetPlayer(player)

		walk.NewVSeparator(toolState.SizingTab)

		saveComposite, err := walk.NewComposite(toolState.SizingTab)
		if err != nil {
			widget.RaiseError(err)
		}
		saveComposite.SetLayout(walk.NewHBoxLayout())

		toolState.SizingTabMotionSaveButton, err = walk.NewPushButton(saveComposite)
		if err != nil {
			widget.RaiseError(err)
		}
		toolState.SizingTabMotionSaveButton.SetText(mi18n.T("モーション保存"))
		toolState.SizingTabMotionSaveButton.Clicked().Attach(toolState.onClickSizingTabMotionSave)

		toolState.SizingTabModelSaveButton, err = walk.NewPushButton(saveComposite)
		if err != nil {
			widget.RaiseError(err)
		}
		toolState.SizingTabModelSaveButton.SetText(mi18n.T("モデル保存"))
		toolState.SizingTabModelSaveButton.Clicked().Attach(toolState.onClickSizingTabModelSave)
	}
}

func execSizing(toolState *ToolState) {
	if !toolState.AdoptSizingCheck.Checked() ||
		toolState.SizingSets[toolState.CurrentIndex].OriginalPmx == nil ||
		toolState.SizingSets[toolState.CurrentIndex].SizingPmx == nil ||
		toolState.SizingSets[toolState.CurrentIndex].OriginalVmd == nil {
		return
	}

	mlog.IL(mi18n.T("サイジング開始"))

	toolState.ControlWindow.Synchronize(func() {
		toolState.SetEnabled(false)
	})

	allScales := usecase.GenerateSizingScales(toolState.SizingSets)
	isExec := false

	errorChan := make(chan error, len(toolState.SizingSets))

	var wg sync.WaitGroup
	for _, sizingSet := range toolState.SizingSets {
		if sizingSet.OriginalPmx != nil && sizingSet.SizingPmx != nil &&
			sizingSet.OriginalVmd != nil {
			wg.Add(1)
			go func(sizingSet *domain.SizingSet) {
				defer wg.Done()
				if (!sizingSet.IsSizingLeg && sizingSet.CompletedSizingLeg) ||
					(!sizingSet.IsSizingUpper && sizingSet.CompletedSizingUpper) ||
					(!sizingSet.IsSizingShoulder && sizingSet.CompletedSizingShoulder) ||
					(!sizingSet.IsSizingArmStance && sizingSet.CompletedSizingArmStance) ||
					(!sizingSet.IsSizingFingerStance && sizingSet.CompletedSizingFingerStance) ||
					(!sizingSet.IsSizingArmTwist && sizingSet.CompletedSizingArmTwist) ||
					(!sizingSet.IsCleanRoot && sizingSet.CompletedCleanRoot) ||
					(!sizingSet.IsCleanCenter && sizingSet.CompletedCleanCenter) ||
					(!sizingSet.IsCleanLegIkParent && sizingSet.CompletedCleanLegIkParent) ||
					(!sizingSet.IsCleanShoulderP && sizingSet.CompletedCleanShoulderP) ||
					(!sizingSet.IsCleanArmIk && sizingSet.CompletedCleanArmIk) ||
					(!sizingSet.IsCleanGrip && sizingSet.CompletedCleanGrip) {
					// チェックを外したら読み直し
					sizingMotion, err := repository.NewVmdVpdRepository().Load(sizingSet.OriginalVmdPath)
					if err != nil {
						mlog.E(mi18n.T("読み込み失敗"), err)
						return
					}
					sizingSet.OutputVmd = sizingMotion.(*vmd.VmdMotion)

					sizingSet.CompletedSizingLeg = false
					sizingSet.CompletedSizingUpper = false
					sizingSet.CompletedSizingShoulder = false
					sizingSet.CompletedSizingArmStance = false
					sizingSet.CompletedSizingFingerStance = false
					sizingSet.CompletedSizingArmTwist = false

					sizingSet.CompletedCleanRoot = false
					sizingSet.CompletedCleanCenter = false
					sizingSet.CompletedCleanLegIkParent = false
					sizingSet.CompletedCleanShoulderP = false
					sizingSet.CompletedCleanArmIk = false
					sizingSet.CompletedCleanGrip = false
				}

				if res, err := usecase.CleanRoot(sizingSet, len(toolState.SizingSets)); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					sizingSet.OutputVmd.SetRandHash()
				}

				if res, err := usecase.CleanCenter(sizingSet, len(toolState.SizingSets)); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					sizingSet.OutputVmd.SetRandHash()
				}

				if res, err := usecase.CleanLegIkParent(sizingSet, len(toolState.SizingSets)); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					sizingSet.OutputVmd.SetRandHash()
				}

				if res, err := usecase.CleanShoulderP(sizingSet, len(toolState.SizingSets)); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					sizingSet.OutputVmd.SetRandHash()
				}

				if res, err := usecase.CleanArmIk(sizingSet, len(toolState.SizingSets)); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					sizingSet.OutputVmd.SetRandHash()
				}

				if res, err := usecase.CleanGrip(sizingSet, len(toolState.SizingSets)); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					sizingSet.OutputVmd.SetRandHash()
				}

				if res, err := usecase.SizingLeg(sizingSet, allScales[sizingSet.Index], len(toolState.SizingSets)); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					sizingSet.OutputVmd.SetRandHash()
				}

				if res, err := usecase.SizingUpper(sizingSet, len(toolState.SizingSets)); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					sizingSet.OutputVmd.SetRandHash()
				}

				if res, err := usecase.SizingShoulder(sizingSet, len(toolState.SizingSets)); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					sizingSet.OutputVmd.SetRandHash()
				}

				if res, err := usecase.SizingArmFingerStance(sizingSet, len(toolState.SizingSets)); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					sizingSet.OutputVmd.SetRandHash()
				}

				if res, err := usecase.SizingArmTwist(sizingSet, len(toolState.SizingSets)); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					sizingSet.OutputVmd.SetRandHash()
				}
			}(sizingSet)
		}
	}
	wg.Wait()
	close(errorChan)

	// チャネルからエラーを受け取る
	for err := range errorChan {
		if err != nil {
			widget.RaiseError(err)
		}
	}

	toolState.ControlWindow.Synchronize(func() {
		toolState.SetEnabled(true)
		toolState.SetOriginalPmxParameterEnabled(toolState.IsOriginalJson())
	})

	if isExec {
		mlog.ILT(mi18n.T("サイジング終了"), mi18n.T("サイジング終了メッセージ"))
	} else {
		mlog.I(mi18n.T("サイジング終了"))
	}

	widget.Beep()
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

func setOutputPath(toolState *ToolState) {
	for i, sizingSet := range toolState.SizingSets {
		// 出力パス設定
		if sizingSet.OriginalVmdPath != "" {
			// サイジング先モデルが指定されている場合、ファイル名を含める
			_, fileName, _ := mutils.SplitPath(sizingSet.SizingPmxPath)

			suffix := ""
			if toolState.SizingSets[i].IsSizingLeg {
				suffix += "L"
			}
			if toolState.SizingSets[i].IsSizingUpper {
				suffix += "U"
			}
			if toolState.SizingSets[i].IsSizingShoulder {
				suffix += "S"
			}
			if toolState.SizingSets[i].IsSizingArmStance {
				suffix += "A"
			}
			if toolState.SizingSets[i].IsSizingFingerStance {
				suffix += "F"
			}
			if toolState.SizingSets[i].IsSizingArmTwist {
				suffix += "W"
			}
			if len(suffix) > 0 {
				suffix = fmt.Sprintf("_%s", suffix)
			}

			sizingSet.OutputVmdPath = mutils.CreateOutputPath(
				sizingSet.OriginalVmdPath, fmt.Sprintf("%s%s", fileName, suffix))
			if i == toolState.CurrentIndex {
				toolState.OutputVmdPicker.SetPath(sizingSet.OutputVmdPath)
			}
		}
	}
}
