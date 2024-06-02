//go:build windows
// +build windows

package main

import (
	"embed"
	"fmt"
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
	})
}

var env string

//go:embed resources/*
var resourceFiles embed.FS

func main() {
	var mWindow *mwidget.MWindow
	var err error

	appConfig := mconfig.LoadAppConfig(resourceFiles)
	appConfig.Env = env

	if appConfig.IsEnvProd() {
		defer mwidget.RecoverFromPanic(mWindow)
	}

	mWindow, err = mwidget.NewMWindow(resourceFiles, appConfig, true, 512, 768, ui.GetMenuItems)
	mwidget.CheckError(err, nil, mi18n.T("メインウィンドウ生成エラー"))

	motionPlayer := ui.NewFileTabPage(mWindow, resourceFiles)

	glWindow, err := mwidget.NewGlWindow(fmt.Sprintf("%s %s", mWindow.Title(), mi18n.T("ビューワー")),
		512, 768, 0, resourceFiles, nil, motionPlayer)
	mwidget.CheckError(err, mWindow, mi18n.T("ビューワーウィンドウ生成エラー"))
	mWindow.AddGlWindow(glWindow)

	mWindow.Center()
	mWindow.Run()
}
