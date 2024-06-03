package ui

import (
	"embed"

	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mwidget"
	"github.com/miu200521358/walk/pkg/walk"
)

type FileTabPage struct {
	*mwidget.MultiPageMTabPage
	MotionPlayer *mwidget.MotionPlayer
	mWindow      *mwidget.MWindow
	SizingPages  []*SizingPage
}

func NewFileTabPage(mWindow *mwidget.MWindow, resourceFiles embed.FS) (*FileTabPage, error) {
	page, err := mwidget.NewMultiPageMTabPage(mWindow, mWindow.TabWidget, mi18n.T("ファイル"), true)
	if err != nil {
		return nil, err
	}

	filePage := &FileTabPage{
		MultiPageMTabPage: page,
		mWindow:           mWindow,
		SizingPages:       make([]*SizingPage, 0),
	}

	// ボタンBox
	buttonComposite, err := walk.NewComposite(filePage.Header)
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
	addButton.SetMinMaxSize(walk.Size{Width: 120, Height: 30}, walk.Size{Width: 120, Height: 30})
	addButton.SetText(mi18n.T("サイジングセット追加"))

	// プレイヤーBox
	playerComposite, err := walk.NewComposite(filePage.Header)
	if err != nil {
		return nil, err
	}
	playerComposite.SetLayout(walk.NewVBoxLayout())

	filePage.MotionPlayer, err = mwidget.NewMotionPlayer(playerComposite, mWindow, resourceFiles)
	if err != nil {
		return nil, err
	}

	// 最初の1ページを追加
	filePage.AddSizingPage()

	addButton.Clicked().Attach(func() {
		// 追加ボタンが押されたらサイジングセット追加
		filePage.AddSizingPage()
	})

	return filePage, nil
}

func (ftp *FileTabPage) AddSizingPage() error {
	sizingPage, err := NewSizingPage(ftp.mWindow, ftp, nil)
	mwidget.CheckError(err, ftp.mWindow, mi18n.T("サイジングセット生成エラー"))

	ftp.AddPage(sizingPage.Composite)
	ftp.SizingPages = append(ftp.SizingPages, sizingPage)

	return nil
}

func (ftp *FileTabPage) Dispose() {
	ftp.MultiPageMTabPage.Dispose()
	ftp.MotionPlayer.Dispose()
	for _, sp := range ftp.SizingPages {
		sp.Dispose()
	}
}
