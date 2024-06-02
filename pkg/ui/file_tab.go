package ui

import (
	"embed"
	"log"
	"path/filepath"

	"github.com/miu200521358/mlib_go/pkg/mutils"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/mlib_go/pkg/mwidget"
	"github.com/miu200521358/mlib_go/pkg/pmx"
	"github.com/miu200521358/mlib_go/pkg/vmd"
	"github.com/miu200521358/walk/pkg/walk"
)

func NewFileTabPage(mWindow *mwidget.MWindow, resourceFiles embed.FS) *mwidget.MotionPlayer {
	page := mwidget.NewMTabPage(mWindow, mWindow.TabWidget, mi18n.T("ファイル"))

	mainLayout := walk.NewVBoxLayout()
	page.SetLayout(mainLayout)

	pmxReadPicker, err := (mwidget.NewPmxReadFilePicker(
		mWindow,
		page,
		"PmxPath",
		mi18n.T("Pmxファイル"),
		mi18n.T("Pmxファイルを選択してください"),
		mi18n.T("Pmxファイルの使い方"),
		func(path string) {}))
	mwidget.CheckError(err, mWindow, mi18n.T("Pmxファイルピッカー生成エラー"))

	vmdReadPicker, err := (mwidget.NewVmdReadFilePicker(
		mWindow,
		page,
		"VmdPath",
		mi18n.T("Vmdファイル"),
		mi18n.T("Vmdファイルを選択してください"),
		mi18n.T("Vmdファイルの使い方"),
		func(path string) {}))
	mwidget.CheckError(err, mWindow, mi18n.T("Vmdファイルピッカー生成エラー"))

	pmxSavePicker, err := (mwidget.NewPmxSaveFilePicker(
		mWindow,
		page,
		mi18n.T("出力Pmxファイル"),
		mi18n.T("出力Pmxファイルパスを入力もしくは選択してください"),
		mi18n.T("出力Pmxファイルの使い方"),
		func(path string) {}))
	mwidget.CheckError(err, mWindow, mi18n.T("出力Pmxファイルピッカー生成エラー"))

	_, err = walk.NewVSeparator(page)
	mwidget.CheckError(err, mWindow, mi18n.T("セパレータ生成エラー"))

	motionPlayer, err := mwidget.NewMotionPlayer(page, mWindow, resourceFiles)
	mwidget.CheckError(err, mWindow, mi18n.T("モーションプレイヤー生成エラー"))
	motionPlayer.SetEnabled(false)

	var onFilePathChanged = func() {
		if motionPlayer.Playing() {
			motionPlayer.Play(false)
		}
		motionPlayer.SetEnabled(pmxReadPicker.Exists() && vmdReadPicker.ExistsOrEmpty())
	}

	pmxReadPicker.OnPathChanged = func(path string) {
		isExist, err := mutils.ExistsFile(path)
		if !isExist || err != nil {
			pmxSavePicker.PathLineEdit.SetText("")
			return
		}

		dir, file := filepath.Split(path)
		ext := filepath.Ext(file)
		outputPath := filepath.Join(dir, file[:len(file)-len(ext)]+"_out"+ext)
		pmxSavePicker.PathLineEdit.SetText(outputPath)

		if pmxReadPicker.Exists() {
			data, err := pmxReadPicker.GetData()
			if err != nil {
				mlog.E(mi18n.T("Pmxファイル読み込みエラー"), err.Error())
				return
			}
			model := data.(*pmx.PmxModel)
			var motion *vmd.VmdMotion
			if vmdReadPicker.IsCached() {
				motion = vmdReadPicker.GetCache().(*vmd.VmdMotion)
			} else {
				motion = vmd.NewVmdMotion("")
			}

			motionPlayer.SetEnabled(true)
			mWindow.GetMainGlWindow().SetFrame(0)
			mWindow.GetMainGlWindow().Play(false)
			mWindow.GetMainGlWindow().ClearData()
			mWindow.GetMainGlWindow().AddData(model, motion)
			mWindow.GetMainGlWindow().Run()
		}

		onFilePathChanged()
	}

	vmdReadPicker.OnPathChanged = func(path string) {
		if vmdReadPicker.Exists() {
			motionData, err := vmdReadPicker.GetData()
			if err != nil {
				mlog.E(mi18n.T("Vmdファイル読み込みエラー"), err.Error())
				return
			}
			motion := motionData.(*vmd.VmdMotion)

			motionPlayer.SetRange(0, motion.GetMaxFrame()+1)
			motionPlayer.SetValue(0)

			if pmxReadPicker.Exists() {
				model := pmxReadPicker.GetCache().(*pmx.PmxModel)

				motionPlayer.SetEnabled(true)
				mWindow.GetMainGlWindow().SetFrame(0)
				mWindow.GetMainGlWindow().Play(false)
				mWindow.GetMainGlWindow().ClearData()
				mWindow.GetMainGlWindow().AddData(model, motion)
				mWindow.GetMainGlWindow().Run()
			}
		}

		onFilePathChanged()
	}

	motionPlayer.OnPlay = func(isPlaying bool) error {
		if !isPlaying {
			pmxReadPicker.SetEnabled(true)
			vmdReadPicker.SetEnabled(true)
			pmxSavePicker.SetEnabled(true)
		} else {
			pmxReadPicker.SetEnabled(false)
			vmdReadPicker.SetEnabled(false)
			pmxSavePicker.SetEnabled(false)
		}

		motionPlayer.PlayButton.SetEnabled(true)
		mWindow.GetMainGlWindow().Play(isPlaying)

		return nil
	}

	pmxReadPicker.PathLineEdit.SetFocus()

	// コンソールはタブ外に表示
	mWindow.ConsoleView, err = mwidget.NewConsoleView(page)
	mwidget.CheckError(err, mWindow, mi18n.T("コンソール生成エラー"))
	log.SetOutput(mWindow.ConsoleView)

	return motionPlayer
}
