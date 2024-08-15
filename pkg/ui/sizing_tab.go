package ui

import (
	"embed"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/miu200521358/mlib_go/pkg/domain/pmx"
	"github.com/miu200521358/mlib_go/pkg/domain/vmd"
	"github.com/miu200521358/mlib_go/pkg/infrastructure/repository"
	"github.com/miu200521358/mlib_go/pkg/interface/controller"
	"github.com/miu200521358/mlib_go/pkg/interface/controller/widget"
	"github.com/miu200521358/mlib_go/pkg/mutils"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/walk/pkg/walk"
)

//go:embed model/*
var modelFs embed.FS

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

					if strings.HasSuffix(strings.ToLower(toolState.OriginalPmxPicker.GetPath()), ".json") {
						// JSONファイルが指定されている場合、embedからPMXモデルの素体を読み込む
						if f, err := modelFs.Open("model/mannequin.pmx"); err == nil {
							if pmxData, err := repository.NewPmxRepository().LoadByFile(f); err == nil {
								pmxModel := pmxData.(*pmx.PmxModel)
								toolState.loadOriginalPmxTextures(pmxModel)
								model = pmxModel
							}
						}
					}

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
				mi18n.T("サイジング先モデルツールチップ"),
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
				mi18n.T("出力モーションツールチップ"),
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

func (toolState *ToolState) loadOriginalPmxTextures(model *pmx.PmxModel) {
	model.SetPath(filepath.Join(os.TempDir(), "model"))
	for _, tex := range model.Textures.Data {
		texPath := filepath.Join("model", tex.Name())
		err := toolState.loadTex(texPath)
		if err == nil {
			tex.SetName(texPath)
		}
	}
}

func (toolState *ToolState) loadTex(texPath string) error {
	fsTexPath := strings.ReplaceAll(texPath, "\\", "/")
	texFile, err := modelFs.ReadFile(fsTexPath)
	if err != nil {
		mlog.E(fmt.Sprintf("Failed to read original pmx tex file: %s", texPath), err)
		return err
	}

	tmpTexPath := filepath.Join(os.TempDir(), texPath)

	// 仮パスのフォルダ構成を作成する
	err = os.MkdirAll(filepath.Dir(tmpTexPath), 0755)
	if err != nil {
		mlog.E(fmt.Sprintf("Failed to create directory: %s", tmpTexPath), err)
		return err
	}

	// 作業フォルダにファイルを書き込む
	err = os.WriteFile(tmpTexPath, texFile, 0644)
	if err != nil {
		mlog.E(fmt.Sprintf("Failed to write file: %s", tmpTexPath), err)
		return err
	}

	return nil
}
