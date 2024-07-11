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

	deleteButton.Clicked().Attach(func() {
		// 全削除ボタンが押されたらサイジングセット全削除
		err = fileTabPage.resetSizingSet()
		mwidget.CheckError(err, mWindow, mi18n.T("サイジングセット全削除エラー"))
	})

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

func (ftp *FileTabPage) resetSizingSet() error {
	// 一旦全部削除
	for i := range ftp.navToolBar.Actions().Len() {
		ftp.navToolBar.Actions().RemoveAt(ftp.navToolBar.Actions().Len() - 1)

		go func() {
			ftp.mWindow.GetMainGlWindow().RemoveModelSetIndexChannel <- (i * 2)
			ftp.mWindow.GetMainGlWindow().RemoveModelSetIndexChannel <- (i*2 + 1)
		}()
	}
	ftp.items.sizingSets = make([]*model.SizingSet, 0)
	ftp.items.currentIndex = -1

	// 1セット追加
	err := ftp.addSizingSet()
	if err != nil {
		return err
	}

	return nil
}

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

func (ftp *FileTabPage) setCurrentAction(index int) error {
	ftp.SetFocus()

	// 一旦すべてのチェックを外す
	for i := range len(ftp.items.sizingSets) {
		ftp.navToolBar.Actions().At(i).SetChecked(false)
	}
	// 該当INDEXのみチェックON
	ftp.items.currentIndex = index
	ftp.navToolBar.Actions().At(index).SetChecked(true)
	ftp.currentPageChangedPublisher.Publish()

	return nil
}

func (ftp *FileTabPage) Dispose() {
	ftp.navToolBar.Dispose()
	ftp.MotionPlayer.Dispose()
	ftp.MTabPage.Dispose()
	ftp.items.Dispose()
}
