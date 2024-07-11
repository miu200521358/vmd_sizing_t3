package ui

import (
	"fmt"

	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mwidget"
	"github.com/miu200521358/vmd_sizing_t3/pkg/model"
	"github.com/miu200521358/walk/pkg/walk"
)

type FileTabPage struct {
	*mwidget.MTabPage
	MotionPlayer                *mwidget.MotionPlayer
	mWindow                     *mwidget.MWindow
	navToolBar                  *walk.ToolBar
	items                       *sizingItem
	currentPageChangedPublisher walk.EventPublisher
}

func NewFileTabPage(mWindow *mwidget.MWindow) (*FileTabPage, error) {
	page, err := mwidget.NewMTabPage(mWindow, mWindow.TabWidget, mi18n.T("ファイル"))
	if err != nil {
		return nil, err
	}
	page.SetLayout(walk.NewVBoxLayout())

	fileTabPage := &FileTabPage{
		MTabPage: page,
		mWindow:  mWindow,
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

	fileTabPage.MotionPlayer, err = mwidget.NewMotionPlayer(playerComposite, mWindow)
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
	fileTabPage.items, err = NewSizingItem(mWindow, fileTabPage, nil)
	if err != nil {
		return nil, err
	}
	// ページ変更時の処理をアタッチ
	fileTabPage.CurrentPageChanged().Attach(fileTabPage.items.OnCurrentPageChanged())

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

	// deleteButton.Clicked().Attach(func() {
	// 	// 全削除ボタンが押されたらサイジングセット全削除
	// 	err = fileTabPage.resetSizingSet()
	// 	mwidget.CheckError(err, mWindow, mi18n.T("サイジングセット全削除エラー"))
	// })

	fileTabPage.MotionPlayer.OnPlay = func(isPlaying bool) error {

		fileTabPage.items.SetEnabled(!isPlaying)
		fileTabPage.MotionPlayer.PlayButton.SetEnabled(true)
		go func() {
			mWindow.GetMainGlWindow().IsPlayingChannel <- isPlaying
		}()

		return nil
	}

	return fileTabPage, nil
}

// func (ftp *FileTabPage) resetSizingSet() error {
// 	// 一旦全部削除
// 	for range ftp.navToolBar.Actions().Len() {
// 		ftp.navToolBar.Actions().RemoveAt(ftp.navToolBar.Actions().Len() - 1)
// 	}
// 	ftp.currentIndex = -1

// 	// 1セット追加
// 	err := ftp.addSizingSet()
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

func (ftp *FileTabPage) CurrentPageChanged() *walk.Event {
	return ftp.currentPageChangedPublisher.Event()
}

func (ftp *FileTabPage) addSizingSet() error {
	action, err := ftp.newPageAction()
	if err != nil {
		return err
	}
	ftp.navToolBar.Actions().Add(action)
	ftp.items.sizingSets = append(ftp.items.sizingSets, model.NewSizingSet(len(ftp.items.sizingSets)))

	if len(ftp.items.sizingSets) > 0 {
		if err := ftp.setCurrentAction(len(ftp.items.sizingSets) - 1); err != nil {
			return err
		}
	}

	return nil
}

func (ftp *FileTabPage) newPageAction() (*walk.Action, error) {
	action := walk.NewAction()
	action.SetCheckable(true)
	action.SetExclusive(true)
	action.SetText(fmt.Sprintf("No. %d", len(ftp.items.sizingSets)+1))
	index := len(ftp.items.sizingSets)

	action.Triggered().Attach(func() {
		ftp.setCurrentAction(index)
	})

	return action, nil
}

// func (ftp *FileTabPage) saveSizingSet(index int) {
// 	if index < 0 {
// 		return
// 	}

// 	originalVmd := ftp.items.OriginalVmdPicker.GetCache()
// 	if originalVmd != nil {
// 		ftp.items.sizingSets[index].OriginalVmd = originalVmd.(*vmd.VmdMotion)
// 	} else {
// 		ftp.items.sizingSets[index].OriginalVmd = nil
// 	}
// 	ftp.items.sizingSets[index].OriginalVmdPath = ftp.items.OriginalVmdPicker.PathLineEdit.Text()

// 	originalPmx := ftp.items.OriginalPmxPicker.GetCache()
// 	if originalPmx != nil {
// 		ftp.items.sizingSets[index].OriginalPmx = originalPmx.(*pmx.PmxModel)
// 	} else {
// 		ftp.items.sizingSets[index].OriginalPmx = nil
// 	}
// 	ftp.items.sizingSets[index].OriginalPmxPath = ftp.items.OriginalPmxPicker.PathLineEdit.Text()

// 	sizingPmx := ftp.items.SizingPmxPicker.GetCache()
// 	if sizingPmx != nil {
// 		ftp.items.sizingSets[index].SizingPmx = sizingPmx.(*pmx.PmxModel)
// 	} else {
// 		ftp.items.sizingSets[index].SizingPmx = nil
// 	}
// 	ftp.items.sizingSets[index].SizingPmxPath = ftp.items.SizingPmxPicker.PathLineEdit.Text()

// 	outputPmx := ftp.items.OutputPmxPicker.GetCache()
// 	if outputPmx != nil {
// 		ftp.items.sizingSets[index].OutputPmx = outputPmx.(*pmx.PmxModel)
// 	} else {
// 		ftp.items.sizingSets[index].OutputPmx = nil
// 	}
// 	ftp.items.sizingSets[index].OutputPmxPath = ftp.items.OutputPmxPicker.PathLineEdit.Text()

// 	outputVmd := ftp.items.OutputVmdPicker.GetCache()
// 	if outputVmd != nil {
// 		ftp.items.sizingSets[index].OutputVmd = outputVmd.(*vmd.VmdMotion)
// 	} else {
// 		ftp.items.sizingSets[index].OutputVmd = nil
// 	}
// 	ftp.items.sizingSets[index].OutputVmdPath = ftp.items.OutputVmdPicker.PathLineEdit.Text()
// }

// func (ftp *FileTabPage) restoreSizingSet(index int) {
// 	if index < 0 {
// 		return
// 	}

// 	originalVmd := ftp.items.sizingSets[index].OriginalVmd
// 	if originalVmd != nil {
// 		ftp.items.OriginalVmdPicker.PathLineEdit.SetText(originalVmd.Path)
// 	} else {
// 		ftp.items.OriginalVmdPicker.PathLineEdit.SetText(ftp.items.sizingSets[index].OriginalVmdPath)
// 	}

// 	originalPmx := ftp.items.sizingSets[index].OriginalPmx
// 	if originalPmx != nil {
// 		ftp.items.OriginalPmxPicker.PathLineEdit.SetText(originalPmx.Path)
// 	} else {
// 		ftp.items.OriginalPmxPicker.PathLineEdit.SetText(ftp.items.sizingSets[index].OriginalPmxPath)
// 	}

// 	sizingPmx := ftp.items.sizingSets[index].SizingPmx
// 	if sizingPmx != nil {
// 		ftp.items.SizingPmxPicker.PathLineEdit.SetText(sizingPmx.Path)
// 	} else {
// 		ftp.items.SizingPmxPicker.PathLineEdit.SetText(ftp.items.sizingSets[index].SizingPmxPath)
// 	}

// 	outputPmx := ftp.items.sizingSets[index].OutputPmx
// 	if outputPmx != nil {
// 		ftp.items.OutputPmxPicker.PathLineEdit.SetText(outputPmx.Path)
// 	} else {
// 		ftp.items.OutputPmxPicker.PathLineEdit.SetText(ftp.items.sizingSets[index].OutputPmxPath)
// 	}

// 	outputVmd := ftp.items.sizingSets[index].OutputVmd
// 	if outputVmd != nil {
// 		ftp.items.OutputVmdPicker.PathLineEdit.SetText(outputVmd.Path)
// 	} else {
// 		ftp.items.OutputVmdPicker.PathLineEdit.SetText(ftp.items.sizingSets[index].OutputVmdPath)
// 	}
// }

func (ftp *FileTabPage) setCurrentAction(index int) error {
	// // 切り替える前のページの情報を保存
	// ftp.saveSizingSet(ftp.currentIndex)

	ftp.SetFocus()

	// 一旦すべてのチェックを外す
	for i := range len(ftp.items.sizingSets) {
		ftp.navToolBar.Actions().At(i).SetChecked(false)
	}
	// 該当INDEXのみチェックON
	ftp.items.currentIndex = index
	ftp.navToolBar.Actions().At(index).SetChecked(true)
	ftp.currentPageChangedPublisher.Publish()

	// // 切り替えた後のページの情報を復元
	// ftp.restoreSizingSet(index)

	return nil
}

func (ftp *FileTabPage) Dispose() {
	ftp.navToolBar.Dispose()
	ftp.MotionPlayer.Dispose()
	ftp.MTabPage.Dispose()
	ftp.items.Dispose()
}
