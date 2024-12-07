package ui

import (
	"fmt"
	"runtime"
	"sync"
	"time"

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
		composite := &declarative.Composite{
			Layout: declarative.VBox{},
			Children: []declarative.Widget{
				declarative.TextLabel{
					Text: mi18n.T("サイジングTabLabel"),
				},
				declarative.Composite{
					Layout: declarative.HBox{},
					Children: []declarative.Widget{
						declarative.HSpacer{},
						// サイジングセット追加ボタン
						declarative.PushButton{
							Text: mi18n.T("サイジングセット追加"),
							OnClicked: func() {
								toolState.addSizingSet()
							},
							MinSize: declarative.Size{Width: 130, Height: 20},
							MaxSize: declarative.Size{Width: 130, Height: 20},
						},
						// サイジングセット全削除ボタン
						declarative.PushButton{
							Text: mi18n.T("サイジングセット全削除"),
							OnClicked: func() {
								toolState.resetSizingSet()
							},
							MinSize: declarative.Size{Width: 130, Height: 20},
							MaxSize: declarative.Size{Width: 130, Height: 20},
						},
						// サイジングセット設定読み込みボタン
						declarative.PushButton{
							Text: mi18n.T("サイジングセット設定読込"),
							OnClicked: func() {
								// toolState.loadSizingSet()
							},
							MinSize: declarative.Size{Width: 130, Height: 20},
							MaxSize: declarative.Size{Width: 130, Height: 20},
						},
					},
				},
				// スクロール
				declarative.ScrollView{
					Layout:  declarative.HBox{},
					MinSize: declarative.Size{Width: toolState.ControlWindow.Config.ControlWindowSize.Width / 2, Height: 45},
					MaxSize: declarative.Size{Width: toolState.ControlWindow.Config.ControlWindowSize.Width * 10, Height: 45},
					Children: []declarative.Widget{
						// ナビゲーション用ツールバー
						declarative.ToolBar{
							AssignTo:    &toolState.NavToolBar,
							Orientation: walk.Horizontal,
							ButtonStyle: declarative.ToolBarButtonTextOnly,
						},
					},
				},
			},
		}

		if err := composite.Create(declarative.NewBuilder(toolState.SizingTab)); err != nil {
			widget.RaiseError(err)
		}
	}

	// スクロール
	scrollView, err := walk.NewScrollView(toolState.SizingTab)
	if err != nil {
		widget.RaiseError(err)
	}
	scrollView.SetScrollbars(true, true)
	scrollView.SetLayout(walk.NewVBoxLayout())
	scrollView.SetMinMaxSize(
		walk.Size{Width: toolState.ControlWindow.Config.ControlWindowSize.Width / 2,
			Height: int(float64(toolState.ControlWindow.Config.ControlWindowSize.Height) / 2.5)},
		walk.Size{Width: toolState.ControlWindow.Config.ControlWindowSize.Width * 8,
			Height: toolState.ControlWindow.Config.ControlWindowSize.Height * 18},
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
			toolState.SetEnabled(false)

			if canLoad, err := toolState.OriginalVmdPicker.CanLoad(); !canLoad {
				if err != nil {
					mlog.ET(mi18n.T("読み込み失敗"), err.Error())
				}
				return
			}

			loadVmd(toolState, path, true)
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
			toolState.SetEnabled(false)

			if canLoad, err := toolState.OriginalPmxPicker.CanLoad(); !canLoad {
				if err != nil {
					mlog.ET(mi18n.T("読み込み失敗"), err.Error())
				}
				return
			}

			resultChan := make(chan loadPmxResult, 1)
			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				defer wg.Done()

				var loadResult loadPmxResult
				rep := repository.NewPmxRepository()
				if data, err := rep.Load(path); err != nil {
					loadResult.model = nil
					loadResult.err = err
					resultChan <- loadResult
					return
				} else {
					model := data.(*pmx.PmxModel)

					if toolState.IsOriginalJson() {
						// jsonから読み込んだ場合、モデル定義を適用して読み込みしなおす
						originalModel, err := usecase.LoadOriginalPmxByJson(model)
						if err != nil {
							loadResult.model = nil
							loadResult.err = err
							resultChan <- loadResult
						} else {
							toolState.SizingSets[toolState.CurrentIndex].OriginalJsonPmx = model
							loadResult.model = originalModel
						}
					} else {
						// pmxを読み込んだ場合、サイジング用に最適化する
						originalModel, _, err := usecase.AdjustPmxForSizing(model, true)
						if err != nil {
							loadResult.model = nil
							loadResult.err = err
							resultChan <- loadResult
						} else {
							toolState.SizingSets[toolState.CurrentIndex].OriginalJsonPmx = nil
							loadResult.model = originalModel
						}
					}

					loadResult.err = nil
					resultChan <- loadResult
				}
			}()

			// 非同期で結果を受け取る
			go func() {
				wg.Wait()
				close(resultChan)

				result := <-resultChan

				if result.err != nil {
					mlog.ET(mi18n.T("読み込み失敗"), err.Error())
				} else if result.model == nil {
					toolState.SizingSets[toolState.CurrentIndex].OriginalPmxPath = path
					toolState.SizingSets[toolState.CurrentIndex].OriginalPmx = nil
					toolState.SizingSets[toolState.CurrentIndex].OriginalPmxName = ""
					toolState.SizingSets[toolState.CurrentIndex].OriginalJsonPmx = nil
				} else {
					// 強制更新用にハッシュ設定
					result.model.SetRandHash()

					toolState.SizingSets[toolState.CurrentIndex].OriginalPmxPath = path
					toolState.SizingSets[toolState.CurrentIndex].OriginalPmx = result.model
					toolState.SizingSets[toolState.CurrentIndex].OriginalPmx.SetIndex(toolState.CurrentIndex)
					toolState.SizingSets[toolState.CurrentIndex].OriginalPmxName = result.model.Name()

					toolState.ControlWindow.Synchronize(func() {
						toolState.ResetSizingCheck(false)
					})

					if !toolState.OriginalVmdPicker.Exists() {
						// モーション未設定の場合、空モーションを定義する
						toolState.SizingSets[toolState.CurrentIndex].OriginalVmd = vmd.NewVmdMotion("")
						toolState.SizingSets[toolState.CurrentIndex].OutputVmd = vmd.NewVmdMotion("")
						toolState.SizingSets[toolState.CurrentIndex].StoreOutputVmd(
							toolState.SizingSets[toolState.CurrentIndex].OutputVmd)
					} else {
						// モーション設定済みの場合、出力VMDを読み直す
						loadVmd(toolState, toolState.SizingSets[toolState.CurrentIndex].OriginalVmdPath, false)
					}
				}

				go func() {
					runtime.GC() // 読み込み時のメモリ解放
				}()

				defer toolState.ControlWindow.Synchronize(func() {
					// 出力パス設定
					setOutputPath(toolState)
					// 画面活性化
					toolState.SetEnabled(true)
					toolState.SetOriginalPmxParameterEnabled(toolState.IsOriginalJson())
				})
			}()
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
			toolState.SetEnabled(false)

			if canLoad, err := toolState.SizingPmxPicker.CanLoad(); !canLoad {
				if err != nil {
					mlog.ET(mi18n.T("読み込み失敗"), err.Error())
				}
				return
			}

			resultChan := make(chan loadPmxResult, 1)
			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				defer wg.Done()

				var loadResult loadPmxResult
				rep := repository.NewPmxRepository()
				if data, err := rep.Load(path); err != nil {
					loadResult.model = nil
					loadResult.err = err
					resultChan <- loadResult
					return
				} else {
					model := data.(*pmx.PmxModel)

					// pmxを読み込んだ場合、サイジング用に最適化する
					originalModel, addBoneNames, err := usecase.AdjustPmxForSizing(model, true)
					if err != nil {
						loadResult.model = nil
						loadResult.err = err
						resultChan <- loadResult
						return
					} else {
						loadResult.model = originalModel
					}

					loadResult.addBoneNames = addBoneNames
					loadResult.err = nil
					resultChan <- loadResult
				}
			}()

			// 非同期で結果を受け取る
			go func() {
				wg.Wait()
				close(resultChan)

				result := <-resultChan
				if result.err != nil {
					mlog.ET(mi18n.T("読み込み失敗"), err.Error())
				} else if result.model == nil {
					toolState.SizingSets[toolState.CurrentIndex].SizingPmxPath = path
					toolState.SizingSets[toolState.CurrentIndex].SizingPmx = nil
					toolState.SizingSets[toolState.CurrentIndex].SizingPmxName = ""
				} else {
					// 強制更新用にハッシュ設定
					result.model.SetRandHash()

					toolState.SizingSets[toolState.CurrentIndex].SizingPmxPath = path
					toolState.SizingSets[toolState.CurrentIndex].SizingPmx = result.model
					toolState.SizingSets[toolState.CurrentIndex].SizingPmx.SetIndex(toolState.CurrentIndex)
					toolState.SizingSets[toolState.CurrentIndex].SizingPmxName = result.model.Name()

					toolState.ControlWindow.Synchronize(func() {
						toolState.ResetSizingCheck(false)
					})

					isAdd := false
					if toolState.OriginalVmdPicker.Exists() {
						for _, boneName := range result.addBoneNames {
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
							"addBoneNames": mutils.JoinSlice(result.addBoneNames)}))
					}

					// 出力モデル
					result.model.SetName(fmt.Sprintf("%s_sizing", result.model.Name()))
					toolState.SizingSets[toolState.CurrentIndex].OutputPmx = result.model
					toolState.SizingSets[toolState.CurrentIndex].OutputPmx.SetIndex(toolState.CurrentIndex)
					toolState.SizingSets[toolState.CurrentIndex].OutputPmxPath = mutils.CreateOutputPath(path, "sizing")

					if !toolState.OriginalVmdPicker.Exists() {
						// モーション未設定の場合、空モーションを定義する
						toolState.SizingSets[toolState.CurrentIndex].OriginalVmd = vmd.NewVmdMotion("")
						toolState.SizingSets[toolState.CurrentIndex].OutputVmd = vmd.NewVmdMotion("")
						toolState.SizingSets[toolState.CurrentIndex].StoreOutputVmd(
							toolState.SizingSets[toolState.CurrentIndex].OutputVmd)
					} else {
						// モーション設定済みの場合、出力VMDを読み直す
						loadVmd(toolState, toolState.SizingSets[toolState.CurrentIndex].OriginalVmdPath, false)
					}
				}

				go func() {
					runtime.GC() // 読み込み時のメモリ解放
				}()

				defer toolState.ControlWindow.Synchronize(func() {
					toolState.OutputPmxPicker.SetPath(toolState.SizingSets[toolState.CurrentIndex].OutputPmxPath)
					// 出力パス設定
					setOutputPath(toolState)
					// 画面活性化
					toolState.SetEnabled(true)
					toolState.SetOriginalPmxParameterEnabled(toolState.IsOriginalJson())
				})
			}()
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
		composite := &declarative.Composite{
			Layout: declarative.VBox{},
			Children: []declarative.Widget{
				&declarative.Composite{
					Layout: declarative.HBox{},
					Children: []declarative.Widget{
						// タイトル
						declarative.TextLabel{
							Text:        mi18n.T("サイジング一括オプション"),
							ToolTipText: mi18n.T("サイジング一括オプション説明"),
							OnMouseDown: func(x, y int, button walk.MouseButton) {
								mlog.IL(mi18n.T("サイジング一括オプション説明"))
							},
						},
						declarative.HSpacer{},
						// 即時反映
						declarative.CheckBox{
							AssignTo: &toolState.AdoptSizingCheck,
							OnCheckedChanged: func() {
								go execSizing(toolState)
							},
							MinSize:     declarative.Size{Width: 100, Height: 20},
							MaxSize:     declarative.Size{Width: 100, Height: 20},
							Text:        mi18n.T("即時反映"),
							ToolTipText: mi18n.T("即時反映説明"),
							Checked:     true,
						},
					},
				},
				&declarative.Composite{
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
								toolState.SizingReductionCheck.UpdateChecked(toolState.SizingCleanAllCheck.Checked())

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
								toolState.SizingReductionCheck.UpdateChecked(toolState.SizingAllCheck.Checked())

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
		composite := &declarative.Composite{
			Layout: declarative.VBox{},
			Children: []declarative.Widget{
				&declarative.Composite{
					Layout: declarative.HBox{},
					Children: []declarative.Widget{
						// タイトル
						declarative.TextLabel{
							Text:        mi18n.T("サイジングオプション"),
							ToolTipText: mi18n.T("サイジングオプション説明"),
							OnMouseDown: func(x, y int, button walk.MouseButton) {
								mlog.IL(mi18n.T("サイジングオプション説明"))
							},
						},
					},
				},
				&declarative.Composite{
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
						// 不要キー間引き
						declarative.CheckBox{
							AssignTo: &toolState.SizingReductionCheck,
							OnCheckedChanged: func() {
								toolState.SizingSets[toolState.CurrentIndex].IsSizingReduction =
									toolState.SizingReductionCheck.Checked()

								go execSizing(toolState)
								// 出力パス設定
								setOutputPath(toolState)
							},
							MinSize:     declarative.Size{Width: 150, Height: 20},
							MaxSize:     declarative.Size{Width: 150, Height: 20},
							Text:        mi18n.T("不要キー間引き"),
							ToolTipText: mi18n.T("不要キー間引き説明"),
						},
					},
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
		composite := &declarative.Composite{
			Layout: declarative.VBox{},
			Children: []declarative.Widget{
				&declarative.Composite{
					Layout: declarative.HBox{},
					Children: []declarative.Widget{
						// タイトル
						declarative.TextLabel{
							Text:        mi18n.T("最適化オプション"),
							ToolTipText: mi18n.T("最適化オプション説明"),
							OnMouseDown: func(x, y int, button walk.MouseButton) {
								mlog.IL(mi18n.T("最適化オプション説明"))
							},
						},
					},
				},
				&declarative.Composite{
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
		composite := &declarative.Composite{
			Layout: declarative.VBox{},
			Children: []declarative.Widget{
				&declarative.Composite{
					Layout: declarative.HBox{},
					Children: []declarative.Widget{
						// タイトル
						declarative.TextLabel{
							Text:        mi18n.T("元モデル素体体格調整"),
							ToolTipText: mi18n.T("元モデル素体体格調整説明"),
							OnMouseDown: func(x, y int, button walk.MouseButton) {
								mlog.IL(mi18n.T("元モデル素体体格調整説明"))
							},
						},
					},
				},
				&declarative.Composite{
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
				},
			},
		}

		if err := composite.Create(declarative.NewBuilder(scrollView)); err != nil {
			widget.RaiseError(err)
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

	completedProcessCount := 1
	totalProcessCount := 0
	if toolState.CleanRootCheck.Checked() {
		totalProcessCount++
	}
	if toolState.CleanCenterCheck.Checked() {
		totalProcessCount++
	}
	if toolState.CleanLegIkParentCheck.Checked() {
		totalProcessCount++
	}
	if toolState.CleanShoulderPCheck.Checked() {
		totalProcessCount++
	}
	if toolState.CleanArmIkCheck.Checked() {
		totalProcessCount++
	}
	if toolState.CleanGripCheck.Checked() {
		totalProcessCount++
	}
	if toolState.SizingLegCheck.Checked() {
		totalProcessCount++
	}
	if toolState.SizingUpperCheck.Checked() {
		totalProcessCount++
	}
	if toolState.SizingShoulderCheck.Checked() {
		totalProcessCount++
	}
	if toolState.SizingArmStanceCheck.Checked() || toolState.SizingFingerStanceCheck.Checked() {
		totalProcessCount++
	}
	if toolState.SizingArmTwistCheck.Checked() {
		totalProcessCount++
	}
	if toolState.SizingReductionCheck.Checked() {
		totalProcessCount++
	}

	start := time.Now()

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
					(!sizingSet.IsSizingReduction && sizingSet.CompletedSizingReduction) ||
					(!sizingSet.IsCleanRoot && sizingSet.CompletedCleanRoot) ||
					(!sizingSet.IsCleanCenter && sizingSet.CompletedCleanCenter) ||
					(!sizingSet.IsCleanLegIkParent && sizingSet.CompletedCleanLegIkParent) ||
					(!sizingSet.IsCleanShoulderP && sizingSet.CompletedCleanShoulderP) ||
					(!sizingSet.IsCleanArmIk && sizingSet.CompletedCleanArmIk) ||
					(!sizingSet.IsCleanGrip && sizingSet.CompletedCleanGrip) {
					// チェックを外したら読み直し
					sizingMotion, err := repository.NewVmdVpdRepository().Load(sizingSet.OriginalVmdPath)
					if err != nil {
						mlog.ET(mi18n.T("読み込み失敗"), err.Error())
						return
					}
					sizingSet.OutputVmd = sizingMotion.(*vmd.VmdMotion)

					sizingSet.CompletedSizingLeg = false
					sizingSet.CompletedSizingUpper = false
					sizingSet.CompletedSizingShoulder = false
					sizingSet.CompletedSizingArmStance = false
					sizingSet.CompletedSizingFingerStance = false
					sizingSet.CompletedSizingArmTwist = false
					sizingSet.CompletedSizingReduction = false

					sizingSet.CompletedCleanRoot = false
					sizingSet.CompletedCleanCenter = false
					sizingSet.CompletedCleanLegIkParent = false
					sizingSet.CompletedCleanShoulderP = false
					sizingSet.CompletedCleanArmIk = false
					sizingSet.CompletedCleanGrip = false
				}

				if res, err := usecase.CleanRoot(sizingSet, len(toolState.SizingSets), completedProcessCount, totalProcessCount); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					if res {
						sizingSet.OutputVmd.SetRandHash()
						sizingSet.StoreOutputVmd(sizingSet.OutputVmd)
						completedProcessCount++
					}
				}

				if res, err := usecase.CleanCenter(sizingSet, len(toolState.SizingSets), completedProcessCount, totalProcessCount); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					if res {
						sizingSet.OutputVmd.SetRandHash()
						sizingSet.StoreOutputVmd(sizingSet.OutputVmd)
						completedProcessCount++
					}
				}

				if res, err := usecase.CleanLegIkParent(sizingSet, len(toolState.SizingSets), completedProcessCount, totalProcessCount); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					if res {
						sizingSet.OutputVmd.SetRandHash()
						sizingSet.StoreOutputVmd(sizingSet.OutputVmd)
						completedProcessCount++
					}
				}

				if res, err := usecase.CleanArmIk(sizingSet, len(toolState.SizingSets), completedProcessCount, totalProcessCount); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					if res {
						sizingSet.OutputVmd.SetRandHash()
						sizingSet.StoreOutputVmd(sizingSet.OutputVmd)
						completedProcessCount++
					}
				}

				if res, err := usecase.CleanShoulderP(sizingSet, len(toolState.SizingSets), completedProcessCount, totalProcessCount); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					if res {
						sizingSet.OutputVmd.SetRandHash()
						sizingSet.StoreOutputVmd(sizingSet.OutputVmd)
						completedProcessCount++
					}
				}

				if res, err := usecase.CleanGrip(sizingSet, len(toolState.SizingSets), completedProcessCount, totalProcessCount); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					if res {
						sizingSet.OutputVmd.SetRandHash()
						sizingSet.StoreOutputVmd(sizingSet.OutputVmd)
						completedProcessCount++
					}
				}

				if res, err := usecase.SizingLeg(sizingSet, allScales[sizingSet.Index], len(toolState.SizingSets), completedProcessCount, totalProcessCount); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					if res {
						sizingSet.OutputVmd.SetRandHash()
						sizingSet.StoreOutputVmd(sizingSet.OutputVmd)
						completedProcessCount++
					}
				}

				if res, err := usecase.SizingUpper(sizingSet, len(toolState.SizingSets), completedProcessCount, totalProcessCount); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					if res {
						sizingSet.OutputVmd.SetRandHash()
						sizingSet.StoreOutputVmd(sizingSet.OutputVmd)
						completedProcessCount++
					}
				}

				if res, err := usecase.SizingShoulder(sizingSet, len(toolState.SizingSets), completedProcessCount, totalProcessCount); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					if res {
						sizingSet.OutputVmd.SetRandHash()
						sizingSet.StoreOutputVmd(sizingSet.OutputVmd)
						completedProcessCount++
					}
				}

				if res, err := usecase.SizingArmFingerStance(sizingSet, len(toolState.SizingSets), completedProcessCount, totalProcessCount); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					if res {
						sizingSet.OutputVmd.SetRandHash()
						sizingSet.StoreOutputVmd(sizingSet.OutputVmd)
						completedProcessCount++
					}
				}

				if res, err := usecase.SizingArmTwist(sizingSet, len(toolState.SizingSets), completedProcessCount, totalProcessCount); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					if res {
						sizingSet.OutputVmd.SetRandHash()
						sizingSet.StoreOutputVmd(sizingSet.OutputVmd)
						completedProcessCount++
					}
				}

				if res, err := usecase.SizingReduction(sizingSet, len(toolState.SizingSets), completedProcessCount, totalProcessCount); err != nil {
					errorChan <- err
					return
				} else {
					isExec = res || isExec
					if res {
						sizingSet.OutputVmd.SetRandHash()
						sizingSet.StoreOutputVmd(sizingSet.OutputVmd)
						completedProcessCount++
					}
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

	// 処理時間の計測終了
	elapsed := time.Since(start)

	if isExec {
		mlog.ILT(mi18n.T("サイジング終了"), mi18n.T("サイジング終了メッセージ",
			map[string]interface{}{"ProcessTime": widget.FormatDuration(elapsed)}))
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
			toolState.OriginalPmxPicker.LoadForce(
				toolState.OriginalPmxPicker.GetPath()).(*pmx.PmxModel)
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
			if toolState.SizingSets[i].IsSizingReduction {
				suffix += "R"
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

func loadVmd(toolState *ToolState, path string, enableFormOnCompletion bool) {

	resultChan := make(chan loadVmdResult, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	for range 2 {
		go func() {
			defer wg.Done()

			var loadResult loadVmdResult
			rep := repository.NewVmdRepository()
			if data, err := rep.Load(path); err != nil {
				loadResult.motion = nil
				loadResult.err = err
			} else {
				motion := data.(*vmd.VmdMotion)
				loadResult.motion = motion
				loadResult.err = nil
			}

			resultChan <- loadResult
		}()
	}

	// 非同期で結果を受け取る
	go func() {
		wg.Wait()
		close(resultChan)

		originalResult := <-resultChan
		if originalResult.err != nil {
			mlog.ET(mi18n.T("読み込み失敗"), originalResult.err.Error())
		} else if originalResult.motion != nil {
			// 強制更新用にハッシュ設定
			originalResult.motion.SetRandHash()

			toolState.SizingSets[toolState.CurrentIndex].OriginalVmdPath = path
			toolState.SizingSets[toolState.CurrentIndex].OriginalVmd = originalResult.motion
			toolState.SizingSets[toolState.CurrentIndex].OriginalVmdName = originalResult.motion.Name()
		}

		sizingResult := <-resultChan
		if sizingResult.err != nil {
			if originalResult.err == nil {
				mlog.ET(mi18n.T("読み込み失敗"), sizingResult.err.Error())
			}
		} else if sizingResult.motion == nil {
			toolState.ControlWindow.Synchronize(func() {
				// 出力パス設定
				toolState.OutputVmdPicker.ChangePath("")
				setOutputPath(toolState)
			})
		} else {
			// 強制更新用にハッシュ設定
			toolState.SizingSets[toolState.CurrentIndex].OutputVmd = sizingResult.motion
			toolState.SizingSets[toolState.CurrentIndex].OutputVmd = sizingResult.motion
			toolState.SizingSets[toolState.CurrentIndex].OutputVmd.SetRandHash()
			toolState.SizingSets[toolState.CurrentIndex].StoreOutputVmd(
				toolState.SizingSets[toolState.CurrentIndex].OutputVmd)
		}

		if toolState.SizingSets[toolState.CurrentIndex].OriginalVmd != nil &&
			toolState.SizingSets[toolState.CurrentIndex].OutputVmd != nil {

			toolState.ControlWindow.Synchronize(func() {
				toolState.ResetSizingCheck(false)
				toolState.ControlWindow.UpdateMaxFrame(
					toolState.SizingSets[toolState.CurrentIndex].OriginalVmd.MaxFrame())
			})

			go execSizing(toolState)
		}

		go func() {
			runtime.GC() // 読み込み時のメモリ解放
		}()

		defer toolState.ControlWindow.Synchronize(func() {
			if enableFormOnCompletion {
				// 出力パス設定
				setOutputPath(toolState)
				// 画面活性化
				toolState.SetEnabled(true)
				toolState.SetOriginalPmxParameterEnabled(toolState.IsOriginalJson())
			}
		})
	}()
}
