package ui

import (
	"embed"
	"fmt"

	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mwidget"
	"github.com/miu200521358/mlib_go/pkg/pmx"
	"github.com/miu200521358/mlib_go/pkg/vmd"
	"github.com/miu200521358/vmd_sizing_t3/pkg/model"
	"github.com/miu200521358/walk/pkg/walk"
)

type FileTabPage struct {
	*mwidget.MTabPage
	MotionPlayer                *mwidget.MotionPlayer
	mWindow                     *mwidget.MWindow
	navToolBar                  *walk.ToolBar
	sizingPage                  *SizingPage
	currentIndex                int
	currentPageChangedPublisher walk.EventPublisher
	SizingSets                  []*model.SizingSet
}

func NewFileTabPage(mWindow *mwidget.MWindow, resourceFiles embed.FS) (*FileTabPage, error) {
	page, err := mwidget.NewMTabPage(mWindow, mWindow.TabWidget, mi18n.T("ファイル"))
	if err != nil {
		return nil, err
	}
	page.SetLayout(walk.NewVBoxLayout())

	fileTabPage := &FileTabPage{
		MTabPage:     page,
		mWindow:      mWindow,
		currentIndex: -1,
	}

	headerComposite, err := walk.NewComposite(fileTabPage)
	if err != nil {
		return nil, err
	}
	headerComposite.SetLayout(walk.NewVBoxLayout())

	// ボタンBox
	buttonComposite, err := walk.NewComposite(headerComposite)
	if err != nil {
		return nil, err
	}
	buttonComposite.SetLayout(walk.NewHBoxLayout())
	walk.NewHSpacer(buttonComposite)

	// サイジングセット追加ボタン
	addButton, err := walk.NewPushButton(buttonComposite)
	if err != nil {
		return nil, err
	}
	addButton.SetMinMaxSize(walk.Size{Width: 130, Height: 30}, walk.Size{Width: 130, Height: 30})
	addButton.SetText(mi18n.T("サイジングセット追加"))

	// サイジングセット全削除ボタン
	deleteButton, err := walk.NewPushButton(buttonComposite)
	if err != nil {
		return nil, err
	}
	deleteButton.SetMinMaxSize(walk.Size{Width: 130, Height: 30}, walk.Size{Width: 130, Height: 30})
	deleteButton.SetText(mi18n.T("サイジングセット全削除"))

	// プレイヤーBox
	playerComposite, err := walk.NewComposite(headerComposite)
	if err != nil {
		return nil, err
	}
	playerComposite.SetLayout(walk.NewVBoxLayout())

	fileTabPage.MotionPlayer, err = mwidget.NewMotionPlayer(playerComposite, mWindow, resourceFiles)
	if err != nil {
		return nil, err
	}

	walk.NewVSeparator(headerComposite)

	// スクロール
	scrollView, err := walk.NewScrollView(fileTabPage)
	if err != nil {
		return nil, err
	}
	scrollView.SetScrollbars(true, false)
	scrollView.SetLayout(walk.NewHBoxLayout())

	// ナビゲーション用ツールバー
	fileTabPage.navToolBar, err = walk.NewToolBarWithOrientationAndButtonStyle(
		scrollView, walk.Horizontal, walk.ToolBarButtonTextOnly)
	if err != nil {
		return nil, err
	}

	// サイジングページ
	fileTabPage.sizingPage, err = NewSizingPage(mWindow, fileTabPage, nil)
	if err != nil {
		return nil, err
	}

	// 最初の1セットを追加
	err = fileTabPage.addSizingSet()
	if err != nil {
		return nil, err
	}

	addButton.Clicked().Attach(func() {
		// 追加ボタンが押されたらサイジングセット追加
		err = fileTabPage.addSizingSet()
		mwidget.CheckError(err, mWindow, mi18n.T("サイジングセット追加エラー"))
	})

	deleteButton.Clicked().Attach(func() {
		// 全削除ボタンが押されたらサイジングセット全削除
		err = fileTabPage.resetSizingSet()
		mwidget.CheckError(err, mWindow, mi18n.T("サイジングセット全削除エラー"))
	})

	fileTabPage.MotionPlayer.OnPlay = func(isPlaying bool) error {
		// 入力欄は全部再生中は無効化
		addButton.SetEnabled(!isPlaying)
		deleteButton.SetEnabled(!isPlaying)
		fileTabPage.sizingPage.SetEnabled(!isPlaying)
		fileTabPage.MotionPlayer.SetEnabled(!isPlaying)
		fileTabPage.MotionPlayer.PlayButton.SetEnabled(true)
		for _, glWindow := range mWindow.GlWindows {
			glWindow.Play(isPlaying)
		}

		return nil
	}

	return fileTabPage, nil
}

func (ftp *FileTabPage) resetSizingSet() error {
	// 一旦全部削除
	for range ftp.navToolBar.Actions().Len() {
		ftp.navToolBar.Actions().RemoveAt(ftp.navToolBar.Actions().Len() - 1)
	}
	ftp.SizingSets = make([]*model.SizingSet, 0)
	ftp.currentIndex = -1

	// 1セット追加
	err := ftp.addSizingSet()
	if err != nil {
		return err
	}

	return nil
}

func (ftp *FileTabPage) addSizingSet() error {
	action, err := ftp.newPageAction()
	if err != nil {
		return err
	}
	ftp.navToolBar.Actions().Add(action)
	ftp.SizingSets = append(ftp.SizingSets, model.NewSizingSet())

	if len(ftp.SizingSets) > 0 {
		if err := ftp.setCurrentAction(len(ftp.SizingSets) - 1); err != nil {
			return err
		}
	}

	return nil
}

func (ftp *FileTabPage) newPageAction() (*walk.Action, error) {
	action := walk.NewAction()
	action.SetCheckable(true)
	action.SetExclusive(true)
	action.SetText(fmt.Sprintf("No. %d", len(ftp.SizingSets)+1))
	index := len(ftp.SizingSets)

	action.Triggered().Attach(func() {
		ftp.setCurrentAction(index)
	})

	return action, nil
}

func (ftp *FileTabPage) saveSizingSet(index int) {
	if index < 0 {
		return
	}

	originalVmd := ftp.sizingPage.OriginalVmdPicker.GetCache()
	if originalVmd != nil {
		ftp.SizingSets[index].OriginalVmd = originalVmd.(*vmd.VmdMotion)
	} else {
		ftp.SizingSets[index].OriginalVmd = nil
	}
	ftp.SizingSets[index].OriginalVmdPath = ftp.sizingPage.OriginalVmdPicker.PathLineEdit.Text()

	originalPmx := ftp.sizingPage.OriginalPmxPicker.GetCache()
	if originalPmx != nil {
		ftp.SizingSets[index].OriginalPmx = originalPmx.(*pmx.PmxModel)
	} else {
		ftp.SizingSets[index].OriginalPmx = nil
	}
	ftp.SizingSets[index].OriginalPmxPath = ftp.sizingPage.OriginalPmxPicker.PathLineEdit.Text()

	sizingPmx := ftp.sizingPage.SizingPmxPicker.GetCache()
	if sizingPmx != nil {
		ftp.SizingSets[index].SizingPmx = sizingPmx.(*pmx.PmxModel)
	} else {
		ftp.SizingSets[index].SizingPmx = nil
	}
	ftp.SizingSets[index].SizingPmxPath = ftp.sizingPage.SizingPmxPicker.PathLineEdit.Text()

	outputPmx := ftp.sizingPage.OutputPmxPicker.GetCache()
	if outputPmx != nil {
		ftp.SizingSets[index].OutputPmx = outputPmx.(*pmx.PmxModel)
	} else {
		ftp.SizingSets[index].OutputPmx = nil
	}
	ftp.SizingSets[index].OutputPmxPath = ftp.sizingPage.OutputPmxPicker.PathLineEdit.Text()

	outputVmd := ftp.sizingPage.OutputVmdPicker.GetCache()
	if outputVmd != nil {
		ftp.SizingSets[index].OutputVmd = outputVmd.(*vmd.VmdMotion)
	} else {
		ftp.SizingSets[index].OutputVmd = nil
	}
	ftp.SizingSets[index].OutputVmdPath = ftp.sizingPage.OutputVmdPicker.PathLineEdit.Text()
}

func (ftp *FileTabPage) restoreSizingSet(index int) {
	if index < 0 {
		return
	}

	originalVmd := ftp.SizingSets[index].OriginalVmd
	if originalVmd != nil {
		ftp.sizingPage.OriginalVmdPicker.PathLineEdit.SetText(originalVmd.Path)
	} else {
		ftp.sizingPage.OriginalVmdPicker.PathLineEdit.SetText(ftp.SizingSets[index].OriginalVmdPath)
	}

	originalPmx := ftp.SizingSets[index].OriginalPmx
	if originalPmx != nil {
		ftp.sizingPage.OriginalPmxPicker.PathLineEdit.SetText(originalPmx.Path)
	} else {
		ftp.sizingPage.OriginalPmxPicker.PathLineEdit.SetText(ftp.SizingSets[index].OriginalPmxPath)
	}

	sizingPmx := ftp.SizingSets[index].SizingPmx
	if sizingPmx != nil {
		ftp.sizingPage.SizingPmxPicker.PathLineEdit.SetText(sizingPmx.Path)
	} else {
		ftp.sizingPage.SizingPmxPicker.PathLineEdit.SetText(ftp.SizingSets[index].SizingPmxPath)
	}

	outputPmx := ftp.SizingSets[index].OutputPmx
	if outputPmx != nil {
		ftp.sizingPage.OutputPmxPicker.PathLineEdit.SetText(outputPmx.Path)
	} else {
		ftp.sizingPage.OutputPmxPicker.PathLineEdit.SetText(ftp.SizingSets[index].OutputPmxPath)
	}

	outputVmd := ftp.SizingSets[index].OutputVmd
	if outputVmd != nil {
		ftp.sizingPage.OutputVmdPicker.PathLineEdit.SetText(outputVmd.Path)
	} else {
		ftp.sizingPage.OutputVmdPicker.PathLineEdit.SetText(ftp.SizingSets[index].OutputVmdPath)
	}
}

func (ftp *FileTabPage) setCurrentAction(index int) error {
	// 切り替える前のページの情報を保存
	ftp.saveSizingSet(ftp.currentIndex)

	ftp.SetFocus()

	for i := range len(ftp.SizingSets) {
		ftp.navToolBar.Actions().At(i).SetChecked(false)
	}
	ftp.currentIndex = index
	ftp.navToolBar.Actions().At(index).SetChecked(true)
	ftp.currentPageChangedPublisher.Publish()

	// 切り替えた後のページの情報を復元
	ftp.restoreSizingSet(index)

	return nil
}

func (ftp *FileTabPage) Dispose() {
	ftp.navToolBar.Dispose()
	ftp.MotionPlayer.Dispose()
	ftp.MTabPage.Dispose()
	ftp.sizingPage.Dispose()
}
