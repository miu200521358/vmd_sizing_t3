package ui

import (
	"path/filepath"

	"github.com/miu200521358/mlib_go/pkg/mutils"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/mlib_go/pkg/mwidget"
	"github.com/miu200521358/mlib_go/pkg/pmx"
	"github.com/miu200521358/mlib_go/pkg/vmd"
	"github.com/miu200521358/vmd_sizing_t3/pkg/model"
	"github.com/miu200521358/vmd_sizing_t3/pkg/usecase"
	"github.com/miu200521358/walk/pkg/declarative"
	"github.com/miu200521358/walk/pkg/walk"
)

type sizingItem struct {
	*walk.Composite
	mWindow           *mwidget.MWindow
	page              *FileTabPage // ファイルタブページ
	currentIndex      int
	sizingSets        []*model.SizingSet  // サイジング情報セット
	originalVmdPicker *mwidget.FilePicker // サイジング対象モーション(Vmd/Vpd)ファイル選択
	originalPmxPicker *mwidget.FilePicker // モーション作成元モデル(Pmx)ファイル選択
	sizingPmxPicker   *mwidget.FilePicker // サイジング先モデル(Pmx)ファイル選択
	outputPmxPicker   *mwidget.FilePicker // 出力モデル(Pmx)ファイル選択
	outputVmdPicker   *mwidget.FilePicker // 出力モーション(Vmd)ファイル選択
}

const SizingPageClass = "SizingPage Class"

func NewSizingItem(
	mWindow *mwidget.MWindow,
	ftp *FileTabPage,
	paramComposite walk.Container,
) (*sizingItem, error) {
	si := new(sizingItem)
	si.mWindow = mWindow
	si.page = ftp
	si.currentIndex = -1

	if err := (declarative.Composite{
		AssignTo: &si.Composite,
		Layout:   declarative.VBox{},
	}).Create(declarative.NewBuilder(ftp)); err != nil {
		return nil, err
	}

	if err := walk.InitWrapperWindow(si); err != nil {
		return nil, err
	}

	var err error
	si.originalVmdPicker, err = (mwidget.NewVmdVpdReadFilePicker(
		mWindow,
		si.Composite,
		"vmd",
		mi18n.T("サイジング対象モーション(Vmd/Vpd)"),
		mi18n.T("サイジング対象モーション(Vmd/Vpd)ファイルを選択してください"),
		mi18n.T("サイジング対象モーションの使い方"),
		func(path string) {}))
	if err != nil {
		return nil, err
	}

	si.originalPmxPicker, err = (mwidget.NewPmxReadFilePicker(
		mWindow,
		si.Composite,
		"org_pmx",
		mi18n.T("モーション作成元モデル(Pmx)"),
		mi18n.T("モーション作成元モデルPmxファイルを選択してください"),
		mi18n.T("モーション作成元モデルの使い方"),
		func(path string) {}))
	if err != nil {
		return nil, err
	}

	si.sizingPmxPicker, err = (mwidget.NewPmxReadFilePicker(
		mWindow,
		si.Composite,
		"rep_pmx",
		mi18n.T("サイジング先モデル(Pmx)"),
		mi18n.T("サイジング先モデルPmxファイルを選択してください"),
		mi18n.T("サイジング先モデルの使い方"),
		func(path string) {}))
	if err != nil {
		return nil, err
	}

	si.outputVmdPicker, err = (mwidget.NewVmdSaveFilePicker(
		mWindow,
		si.Composite,
		mi18n.T("出力モーション(Vmd)"),
		mi18n.T("出力モーション(Vmd)ファイルパスを指定してください"),
		mi18n.T("出力モーションの使い方"),
		func(path string) {}))
	if err != nil {
		return nil, err
	}

	si.outputPmxPicker, err = (mwidget.NewPmxSaveFilePicker(
		mWindow,
		si.Composite,
		mi18n.T("出力モデル(Pmx)"),
		mi18n.T("出力モデル(Pmx)ファイルパスを指定してください"),
		mi18n.T("出力モデルの使い方"),
		func(path string) {}))
	if err != nil {
		return nil, err
	}

	si.originalVmdPicker.PathLineEdit.SetFocus()

	// モーション作成元モデル読み込み時の処理
	si.originalPmxPicker.OnPathChanged = si.onOriginalPmxPathChanged()

	// サイジング対象モーション読み込み時の処理
	si.originalVmdPicker.OnPathChanged = si.onOriginalVmdPathChanged()

	// サイジング先モデル読み込み時の処理
	si.sizingPmxPicker.OnPathChanged = si.onSizingPmxPathChanged()

	return si, nil
}

// オリジナルモデル読み込み時の処理
func (si *sizingItem) onOriginalPmxPathChanged() func(string) {
	return func(path string) {
		if si.originalPmxPicker.Exists() {
			si.sizingPmxPicker.ClearCache()
			data, err := si.originalPmxPicker.GetData()
			if err != nil {
				mlog.E(mi18n.T("Pmxファイル読み込みエラー"), err.Error())
				return
			}

			model := data.(*pmx.PmxModel)
			model = usecase.SetupOriginalPmx(model)

			si.sizingSets[si.currentIndex].OriginalPmx = model
			si.sizingSets[si.currentIndex].OriginalPmxPath = path

			go func() {
				si.mWindow.GetMainGlWindow().FrameChannel <- 0
				si.mWindow.GetMainGlWindow().IsPlayingChannel <- false
				si.mWindow.GetMainGlWindow().ReplaceModelSetChannel <- map[int]*mwidget.ModelSet{
					(si.currentIndex * 2): {NextModel: si.sizingSets[si.currentIndex].OriginalPmx}}
			}()
		}
	}
}

// オリジナルモーション読み込み時の処理
func (si *sizingItem) onOriginalVmdPathChanged() func(string) {
	return func(path string) {
		if si.originalVmdPicker.Exists() {
			si.originalVmdPicker.ClearCache()
			data, err := si.originalVmdPicker.GetData()
			if err != nil {
				mlog.E(mi18n.T("Vmdファイル読み込みエラー"), err.Error())
				return
			}

			motion := data.(*vmd.VmdMotion)
			motion = usecase.SetupOriginalVmd(motion)

			si.sizingSets[si.currentIndex].OriginalVmd = motion
			si.sizingSets[si.currentIndex].OriginalVmdPath = path

			// もう一回出力用に読み直す
			si.sizingSets[si.currentIndex].OutputVmd = si.originalVmdPicker.GetDataForce().(*vmd.VmdMotion)

			go func() {
				si.mWindow.GetMainGlWindow().FrameChannel <- 0
				si.mWindow.GetMainGlWindow().IsPlayingChannel <- false
				si.mWindow.GetMainGlWindow().ReplaceModelSetChannel <- map[int]*mwidget.ModelSet{
					(si.currentIndex * 2): {NextMotion: si.sizingSets[si.currentIndex].OriginalVmd}}
				si.mWindow.GetMainGlWindow().ReplaceModelSetChannel <- map[int]*mwidget.ModelSet{
					(si.currentIndex*2 + 1): {NextMotion: si.sizingSets[si.currentIndex].OutputVmd}}
			}()

			si.page.MotionPlayer.SetEnabled(true)
			si.page.MotionPlayer.SetRange(0, si.sizingSets[si.currentIndex].OriginalVmd.GetMaxFrame()+1)
			si.page.MotionPlayer.SetValue(0)

			si.updateOutputPath()
		}
	}
}

// サイジング先モデル読み込み時の処理
func (si *sizingItem) onSizingPmxPathChanged() func(string) {
	return func(path string) {
		isExist, err := mutils.ExistsFile(path)
		if !isExist || err != nil {
			si.outputPmxPicker.PathLineEdit.SetText("")
			return
		}

		if si.sizingPmxPicker.Exists() {
			si.sizingPmxPicker.ClearCache()
			data, err := si.sizingPmxPicker.GetData()
			if err != nil {
				mlog.E(mi18n.T("Pmxファイル読み込みエラー"), err.Error())
				return
			}

			si.sizingSets[si.currentIndex].SizingPmx = data.(*pmx.PmxModel)
			si.sizingSets[si.currentIndex].SizingPmxPath = path

			go func() {
				si.mWindow.GetMainGlWindow().FrameChannel <- 0
				si.mWindow.GetMainGlWindow().IsPlayingChannel <- false
				si.mWindow.GetMainGlWindow().ReplaceModelSetChannel <- map[int]*mwidget.ModelSet{
					(si.currentIndex * 2) + 1: {NextModel: si.sizingSets[si.currentIndex].SizingPmx}}
			}()

			si.updateOutputPath()
		}
	}
}

func (si *sizingItem) updateOutputPath() {
	originalVmdPath := si.sizingSets[si.currentIndex].OriginalVmdPath
	sizingPmxPath := si.sizingSets[si.currentIndex].SizingPmxPath

	if originalVmdPath == "" || sizingPmxPath == "" {
		return
	}

	// オリジナルVMDパス
	_, motionFileName := filepath.Split(originalVmdPath)
	motionFileNameNotExt := motionFileName[:len(motionFileName)-len(filepath.Ext(motionFileName))]

	// サイジング先PMXパス
	_, modelFileName := filepath.Split(sizingPmxPath)
	modelFileNameNotExt := modelFileName[:len(modelFileName)-len(filepath.Ext(modelFileName))]

	// 出力モーションパス
	motionOutputPath := mutils.CreateOutputPath(originalVmdPath, modelFileNameNotExt)
	si.sizingSets[si.currentIndex].OutputVmdPath = motionOutputPath
	si.outputVmdPicker.SetPath(motionOutputPath)

	// 出力モデルパス
	pmxOutputPath := mutils.CreateOutputPath(sizingPmxPath, motionFileNameNotExt)
	si.sizingSets[si.currentIndex].OutputPmxPath = pmxOutputPath
	si.outputPmxPicker.SetPath(pmxOutputPath)
}

func (si *sizingItem) SetEnabled(visible bool) {
	si.Composite.SetEnabled(visible)
	si.originalVmdPicker.SetEnabled(visible)
	si.originalPmxPicker.SetEnabled(visible)
	si.sizingPmxPicker.SetEnabled(visible)
	si.outputPmxPicker.SetEnabled(visible)
	si.outputVmdPicker.SetEnabled(visible)
}

func (si *sizingItem) Dispose() {
	si.Composite.Dispose()
	si.originalVmdPicker.Dispose()
	si.originalPmxPicker.Dispose()
	si.sizingPmxPicker.Dispose()
	si.outputPmxPicker.Dispose()
	si.outputVmdPicker.Dispose()
}

func (si *sizingItem) OnCurrentPageChanged() func() {
	return func() {
		si.originalPmxPicker.OnChanged(si.sizingSets[si.currentIndex].OriginalPmxPath)
		si.originalVmdPicker.OnChanged(si.sizingSets[si.currentIndex].OriginalVmdPath)
		si.sizingPmxPicker.OnChanged(si.sizingSets[si.currentIndex].SizingPmxPath)
		si.outputPmxPicker.SetPath(si.sizingSets[si.currentIndex].OutputPmxPath)
		si.outputVmdPicker.SetPath(si.sizingSets[si.currentIndex].OutputVmdPath)
	}
}
