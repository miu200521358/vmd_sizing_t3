//go:build windows
// +build windows

package main

import (
	"embed"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/miu200521358/vmd_sizing_t3/pkg/ui"
	"github.com/miu200521358/walk/pkg/walk"

	"github.com/miu200521358/mlib_go/pkg/interface/app"
	"github.com/miu200521358/mlib_go/pkg/interface/controller"
	"github.com/miu200521358/mlib_go/pkg/interface/controller/widget"
	"github.com/miu200521358/mlib_go/pkg/interface/viewer"
	"github.com/miu200521358/mlib_go/pkg/mutils/mconfig"
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
)

var env string

func init() {
	runtime.LockOSThread()

	// システム上の25%の論理プロセッサを使用する
	runtime.GOMAXPROCS(max(1, int(runtime.NumCPU()/4)))

	walk.AppendToWalkInit(func() {
		walk.MustRegisterWindowClass(widget.ConsoleViewClass)
	})
}

//go:embed app/*
var appFiles embed.FS

//go:embed i18n/*
var appI18nFiles embed.FS

func main() {
	appConfig := mconfig.LoadAppConfig(appFiles)
	appConfig.Env = env
	mi18n.Initialize(appI18nFiles)

	mApp := app.NewMApp(appConfig)
	mApp.RunViewerToControlChannel()
	mApp.RunControlToViewerChannel()

	go func() {
		fmt.Printf("ControlWindow 01: Now[%s]\n", time.Now().Format("2006-01-02 15:04:05.000"))

		// 操作ウィンドウは別スレッドで起動
		controlWindow := controller.NewControlWindow(appConfig, mApp.ControlToViewerChannel(), ui.GetMenuItems, 2)
		mApp.SetControlWindow(controlWindow)

		fmt.Printf("ControlWindow 02: Now[%s]\n", time.Now().Format("2006-01-02 15:04:05.000"))

		controlWindow.InitTabWidget()

		fmt.Printf("ControlWindow 03: Now[%s]\n", time.Now().Format("2006-01-02 15:04:05.000"))

		ui.NewToolState(mApp, controlWindow)

		fmt.Printf("ControlWindow 04: Now[%s]\n", time.Now().Format("2006-01-02 15:04:05.000"))

		consoleView := widget.NewConsoleView(controlWindow.MainWindow, 256, 50)
		log.SetOutput(consoleView)

		fmt.Printf("ControlWindow 05: Now[%s]\n", time.Now().Format("2006-01-02 15:04:05.000"))

		mApp.RunController()
	}()

	mApp.AddViewWindow(viewer.NewViewWindow(
		mApp.ViewerCount(), appConfig, mApp, mApp.ViewerToControlChannel(), mi18n.T("サイジング用ビューワー"), nil))
	mApp.AddViewWindow(viewer.NewViewWindow(
		mApp.ViewerCount(), appConfig, mApp, mApp.ViewerToControlChannel(), mi18n.T("元モデル用ビューワー"),
		mApp.MainViewWindow().GetWindow()))

	mApp.Center()
	mApp.RunViewer()
}
