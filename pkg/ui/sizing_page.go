package ui

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/miu200521358/mlib_go/pkg/mutils"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/mlib_go/pkg/mwidget"
	"github.com/miu200521358/mlib_go/pkg/pmx"
	"github.com/miu200521358/mlib_go/pkg/vmd"
	"github.com/miu200521358/walk/pkg/declarative"
	"github.com/miu200521358/walk/pkg/walk"
)

type SizingPage struct {
	*walk.Composite
	mWindow           *mwidget.MWindow
	page              *FileTabPage
	OriginalVmdPicker *mwidget.FilePicker
	OriginalPmxPicker *mwidget.FilePicker
	SizingPmxPicker   *mwidget.FilePicker
	OutputPmxPicker   *mwidget.FilePicker
	OutputVmdPicker   *mwidget.FilePicker
}

const SizingPageClass = "SizingPage Class"

func NewSizingPage(
	mWindow *mwidget.MWindow,
	ftp *FileTabPage,
	paramComposite walk.Container,
) (*SizingPage, error) {
	sp := new(SizingPage)
	sp.mWindow = mWindow
	sp.page = ftp

	if err := (declarative.Composite{
		AssignTo: &sp.Composite,
		Layout:   declarative.VBox{},
	}).Create(declarative.NewBuilder(ftp)); err != nil {
		return nil, err
	}

	if err := walk.InitWrapperWindow(sp); err != nil {
		return nil, err
	}

	var err error
	sp.OriginalVmdPicker, err = (mwidget.NewVmdVpdReadFilePicker(
		mWindow,
		sp.Composite,
		"vmd",
		mi18n.T("サイジング対象モーション(Vmd/Vpd)"),
		mi18n.T("サイジング対象モーション(Vmd/Vpd)ファイルを選択してください"),
		mi18n.T("サイジング対象モーションの使い方"),
		func(path string) {}))
	if err != nil {
		return nil, err
	}

	sp.OriginalPmxPicker, err = (mwidget.NewPmxReadFilePicker(
		mWindow,
		sp.Composite,
		"org_pmx",
		mi18n.T("モーション作成元モデル(Pmx)"),
		mi18n.T("モーション作成元モデルPmxファイルを選択してください"),
		mi18n.T("モーション作成元モデルの使い方"),
		func(path string) {}))
	if err != nil {
		return nil, err
	}

	sp.SizingPmxPicker, err = (mwidget.NewPmxReadFilePicker(
		mWindow,
		sp.Composite,
		"rep_pmx",
		mi18n.T("サイジング先モデル(Pmx)"),
		mi18n.T("サイジング先モデルPmxファイルを選択してください"),
		mi18n.T("サイジング先モデルの使い方"),
		func(path string) {}))
	if err != nil {
		return nil, err
	}

	sp.OutputVmdPicker, err = (mwidget.NewVmdSaveFilePicker(
		mWindow,
		sp.Composite,
		mi18n.T("出力モーション(Vmd)"),
		mi18n.T("出力モーション(Vmd)ファイルパスを指定してください"),
		mi18n.T("出力モーションの使い方"),
		func(path string) {}))
	if err != nil {
		return nil, err
	}

	sp.OutputPmxPicker, err = (mwidget.NewPmxSaveFilePicker(
		mWindow,
		sp.Composite,
		mi18n.T("出力モデル(Pmx)"),
		mi18n.T("出力モデル(Pmx)ファイルパスを指定してください"),
		mi18n.T("出力モデルの使い方"),
		func(path string) {}))
	if err != nil {
		return nil, err
	}

	sp.OriginalVmdPicker.PathLineEdit.SetFocus()

	var onFilePathChanged = func() {
		if ftp.MotionPlayer.Playing() {
			ftp.MotionPlayer.Play(false)
		}
		ftp.MotionPlayer.SetEnabled(sp.OutputPmxPicker.Exists() && sp.OriginalVmdPicker.ExistsOrEmpty())
	}

	// サイジング対象モーション読み込み時の処理
	sp.OriginalVmdPicker.OnPathChanged = func(path string) {
		if sp.OriginalVmdPicker.Exists() {
			_, err := sp.OriginalVmdPicker.GetData()
			if err != nil {
				mlog.E(mi18n.T("Vmdファイル読み込みエラー"), err.Error())
				return
			}

			sp.updateOutputPath()
			sp.updateMotionPlayer()
		}

		onFilePathChanged()
	}

	// サイジング先モデル読み込み時の処理
	sp.SizingPmxPicker.OnPathChanged = func(path string) {
		isExist, err := mutils.ExistsFile(path)
		if !isExist || err != nil {
			sp.OutputPmxPicker.PathLineEdit.SetText("")
			return
		}

		if sp.SizingPmxPicker.Exists() {
			_, err := sp.SizingPmxPicker.GetData()
			if err != nil {
				mlog.E(mi18n.T("Pmxファイル読み込みエラー"), err.Error())
				return
			}

			sp.updateOutputPath()
			sp.updateMotionPlayer()
		}

		onFilePathChanged()
	}

	return sp, nil
}

func (sp *SizingPage) updateMotionPlayer() {
	if sp.SizingPmxPicker.IsCached() && sp.OriginalVmdPicker.IsCached() {
		model := sp.SizingPmxPicker.GetCache().(*pmx.PmxModel)
		motion := sp.OriginalVmdPicker.GetCache().(*vmd.VmdMotion)

		sp.page.MotionPlayer.SetRange(0, motion.GetMaxFrame()+1)
		sp.page.MotionPlayer.SetValue(0)

		sp.page.MotionPlayer.SetEnabled(true)
		sp.mWindow.GetMainGlWindow().SetFrame(0)
		sp.mWindow.GetMainGlWindow().Play(false)
		sp.mWindow.GetMainGlWindow().ClearData()
		sp.mWindow.GetMainGlWindow().AddData(model, motion)
		sp.mWindow.GetMainGlWindow().Run()
	} else {
		sp.page.MotionPlayer.SetEnabled(false)
		sp.mWindow.GetMainGlWindow().Play(false)
	}
}

func (sp *SizingPage) updateOutputPath() {
	var model *pmx.PmxModel
	if sp.SizingPmxPicker.IsCached() {
		model = sp.SizingPmxPicker.GetCache().(*pmx.PmxModel)
	}

	var motion *vmd.VmdMotion
	if sp.OriginalVmdPicker.IsCached() {
		motion = sp.OriginalVmdPicker.GetCache().(*vmd.VmdMotion)
	}

	if model == nil || motion == nil {
		return
	}

	// 出力モーションパス
	motionDir, motionFileName := filepath.Split(motion.Path)
	motionExt := filepath.Ext(motionFileName)
	motionOutputPath := filepath.Join(motionDir, fmt.Sprintf("%s_%s%s",
		motionFileName[:len(motionFileName)-len(motionExt)], time.Now().Format("20060102_150405"), motionExt))
	sp.OutputPmxPicker.PathLineEdit.SetText(motionOutputPath)

	// 出力モデルパス
	modelDir, modelFileName := filepath.Split(model.Path)
	ext := filepath.Ext(modelFileName)
	pmxOutputPath := filepath.Join(modelDir, fmt.Sprintf("%s_%s%s",
		modelFileName[:len(modelFileName)-len(ext)], motionFileName[:len(motionFileName)-len(motionExt)], ext))
	sp.OutputPmxPicker.PathLineEdit.SetText(pmxOutputPath)

}

func (sp *SizingPage) SetEnabled(visible bool) {
	sp.Composite.SetEnabled(visible)
	sp.OriginalVmdPicker.SetEnabled(visible)
	sp.OriginalPmxPicker.SetEnabled(visible)
	sp.SizingPmxPicker.SetEnabled(visible)
	sp.OutputPmxPicker.SetEnabled(visible)
	sp.OutputVmdPicker.SetEnabled(visible)
}

func (sp *SizingPage) Dispose() {
	sp.Composite.Dispose()
	sp.OriginalVmdPicker.Dispose()
	sp.OriginalPmxPicker.Dispose()
	sp.SizingPmxPicker.Dispose()
	sp.OutputPmxPicker.Dispose()
	sp.OutputVmdPicker.Dispose()
}
