package ui

import (
	"strings"

	"github.com/miu200521358/mlib_go/pkg/infrastructure/repository"
	"github.com/miu200521358/mlib_go/pkg/interface/controller"
	"github.com/miu200521358/mlib_go/pkg/interface/controller/widget"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/walk/pkg/walk"
)

func newJsonSaveTab(controlWindow *controller.ControlWindow, toolState *ToolState) {
	toolState.JsonSaveTab = widget.NewMTabPage(mi18n.T("モデルjson出力"))
	controlWindow.AddTabPage(toolState.JsonSaveTab.TabPage)

	toolState.JsonSaveTab.SetLayout(walk.NewVBoxLayout())

	composite, err := walk.NewComposite(toolState.JsonSaveTab)
	if err != nil {
		widget.RaiseError(err)
	}
	composite.SetLayout(walk.NewVBoxLayout())

	// ラベル
	label, err := walk.NewTextLabel(composite)
	if err != nil {
		widget.RaiseError(err)
	}
	label.SetText(mi18n.T("モデルjson出力説明"))

	jsonSavePmxPicker := widget.NewPmxReadFilePicker(
		controlWindow,
		composite,
		"org_pmx",
		mi18n.T("json出力対象モデル"),
		mi18n.T("json出力対象モデルツールチップ"),
		mi18n.T("json出力対象モデルの使い方"))

	jsonSavePicker := widget.NewPmxJsonSaveFilePicker(
		controlWindow,
		composite,
		mi18n.T("モデル定義json出力先"),
		mi18n.T("モデル定義json出力先ツールチップ"),
		mi18n.T("モデル定義json出力先の使い方"))

	saveButton, err := walk.NewPushButton(composite)
	if err != nil {
		widget.RaiseError(err)
	}
	saveButton.SetText(mi18n.T("モデル定義json出力"))

	walk.NewVSpacer(toolState.JsonSaveTab)

	// --------------

	jsonSavePmxPicker.SetOnPathChanged(func(path string) {
		saveButton.SetEnabled(false)

		if _, err := jsonSavePmxPicker.Load(); err == nil {
			// 出力パス設定
			outputPath := strings.ReplaceAll(path, ".pmx", "_config.json")
			jsonSavePicker.SetPath(outputPath)
			saveButton.SetEnabled(true)

		} else {
			mlog.E(mi18n.T("読み込み失敗"), err)
		}
	})

	saveButton.Clicked().Attach(func() {
		if data, err := jsonSavePmxPicker.Load(); err == nil {
			rep := repository.NewPmxJsonRepository()

			if err := rep.Save(jsonSavePicker.GetPath(), data, false); err == nil {
				mlog.IT(mi18n.T("出力成功"), mi18n.T("json出力成功メッセージ",
					map[string]interface{}{"Path": jsonSavePicker.GetPath()}))
			} else {
				mlog.ET(mi18n.T("出力失敗"), mi18n.T("json出力失敗メッセージ",
					map[string]interface{}{"Error": err.Error()}))
			}
		} else {
			mlog.E(mi18n.T("読み込み失敗"), err)
		}
	})

}
