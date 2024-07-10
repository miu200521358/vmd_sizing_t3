//go:build windows
// +build windows

package main

import (
	"embed"
	"fmt"
	"log"
	"runtime"

	"github.com/miu200521358/walk/pkg/walk"

	"github.com/miu200521358/mlib_go/pkg/mutils/mconfig"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mwidget"
	"github.com/miu200521358/vmd_sizing_t3/pkg/ui"
)

func init() {
	runtime.LockOSThread()

	walk.AppendToWalkInit(func() {
		walk.MustRegisterWindowClass(mwidget.FilePickerClass)
		walk.MustRegisterWindowClass(mwidget.MotionPlayerClass)
		walk.MustRegisterWindowClass(mwidget.ConsoleViewClass)
		walk.MustRegisterWindowClass(ui.SizingPageClass)
	})
}

var env string

//go:embed app/*
var appFiles embed.FS

//go:embed i18n/*
var appI18nFiles embed.FS

func main() {
	var mWindow *mwidget.MWindow
	var err error

	appConfig := mconfig.LoadAppConfig(appFiles)
	appConfig.Env = env
	mi18n.Initialize(appI18nFiles)

	if appConfig.IsEnvProd() || appConfig.IsEnvDev() {
		defer mwidget.RecoverFromPanic(mWindow)
	}

	iconImg, err := mconfig.LoadIconFile(appFiles)
	mwidget.CheckError(err, nil, mi18n.T("アイコン生成エラー"))

	glWindow, err := mwidget.NewGlWindow(512, 768, 0, iconImg, appConfig, nil, nil)
	mwidget.CheckError(err, mWindow, mi18n.T("ビューワーウィンドウ生成エラー"))

	go func() {
		mWindow, err = mwidget.NewMWindow(512, 768, ui.GetMenuItems, iconImg, appConfig, true)
		mwidget.CheckError(err, nil, mi18n.T("メインウィンドウ生成エラー"))

		filePage, err := ui.NewFileTabPage(mWindow)
		mwidget.CheckError(err, nil, mi18n.T("ファイルタブ生成エラー"))

		// コンソールはタブ外に表示
		mWindow.ConsoleView, err = mwidget.NewConsoleView(mWindow, 256, 30)
		mwidget.CheckError(err, mWindow, mi18n.T("コンソール生成エラー"))
		log.SetOutput(mWindow.ConsoleView)

		glWindow.SetMotionPlayer(filePage.MotionPlayer)
		glWindow.SetTitle(fmt.Sprintf("%s %s", mWindow.Title(), mi18n.T("ビューワー")))
		mWindow.AddGlWindow(glWindow)

		mWindow.AsFormBase().Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
			go func() {
				mWindow.GetMainGlWindow().IsClosedChannel <- true
			}()
			mWindow.Close()
		})

		mWindow.Center()
		mWindow.Run()
	}()

	glWindow.Run()
}
