package ui

import (
	"github.com/miu200521358/mlib_go/pkg/mutils/mi18n"
	"github.com/miu200521358/mlib_go/pkg/mwidget"
	"github.com/miu200521358/walk/pkg/declarative"
	"github.com/miu200521358/walk/pkg/walk"
)

type SizingPage struct {
	*walk.Composite
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
	sizingPage := new(SizingPage)

	if err := (declarative.Composite{
		AssignTo: &sizingPage.Composite,
		Layout:   declarative.VBox{},
	}).Create(declarative.NewBuilder(ftp)); err != nil {
		return nil, err
	}

	if err := walk.InitWrapperWindow(sizingPage); err != nil {
		return nil, err
	}

	var err error
	sizingPage.OriginalVmdPicker, err = (mwidget.NewVmdVpdReadFilePicker(
		mWindow,
		sizingPage.Composite,
		"vmd",
		mi18n.T("サイジング対象モーション(Vmd/Vpd)"),
		mi18n.T("サイジング対象モーション(Vmd/Vpd)ファイルを選択してください"),
		mi18n.T("サイジング対象モーションの使い方"),
		func(path string) {}))
	if err != nil {
		return nil, err
	}

	sizingPage.OriginalPmxPicker, err = (mwidget.NewPmxReadFilePicker(
		mWindow,
		sizingPage.Composite,
		"org_pmx",
		mi18n.T("モーション作成元モデル(Pmx)"),
		mi18n.T("モーション作成元モデルPmxファイルを選択してください"),
		mi18n.T("モーション作成元モデルの使い方"),
		func(path string) {}))
	if err != nil {
		return nil, err
	}

	sizingPage.SizingPmxPicker, err = (mwidget.NewPmxReadFilePicker(
		mWindow,
		sizingPage.Composite,
		"rep_pmx",
		mi18n.T("サイジング先モデル(Pmx)"),
		mi18n.T("サイジング先モデルPmxファイルを選択してください"),
		mi18n.T("サイジング先モデルの使い方"),
		func(path string) {}))
	if err != nil {
		return nil, err
	}

	sizingPage.OutputVmdPicker, err = (mwidget.NewVmdSaveFilePicker(
		mWindow,
		sizingPage.Composite,
		mi18n.T("出力モーション(Vmd)"),
		mi18n.T("出力モーション(Vmd)ファイルパスを指定してください"),
		mi18n.T("出力モーションの使い方"),
		func(path string) {}))
	if err != nil {
		return nil, err
	}

	sizingPage.OutputPmxPicker, err = (mwidget.NewPmxSaveFilePicker(
		mWindow,
		sizingPage.Composite,
		mi18n.T("出力モデル(Pmx)"),
		mi18n.T("出力モデル(Pmx)ファイルパスを指定してください"),
		mi18n.T("出力モデルの使い方"),
		func(path string) {}))
	if err != nil {
		return nil, err
	}

	sizingPage.OriginalVmdPicker.PathLineEdit.SetFocus()

	return sizingPage, nil
}

// func (sp *SizingPage) CreateLayoutItem(ctx *walk.LayoutContext) walk.LayoutItem {
// 	return &sizingPageLayoutItem{idealSize: walk.SizeFrom96DPI(walk.Size{Width: 50, Height: 50}, ctx.DPI())}
// }

func (sp *SizingPage) SetVisible(visible bool) {
	sp.Composite.SetVisible(visible)
	sp.OriginalVmdPicker.SetVisible(visible)
	sp.OriginalPmxPicker.SetVisible(visible)
	sp.SizingPmxPicker.SetVisible(visible)
	sp.OutputPmxPicker.SetVisible(visible)
	sp.OutputVmdPicker.SetVisible(visible)
}

func (sp *SizingPage) Dispose() {
	sp.Composite.Dispose()
	sp.OriginalVmdPicker.Dispose()
	sp.OriginalPmxPicker.Dispose()
	sp.SizingPmxPicker.Dispose()
	sp.OutputPmxPicker.Dispose()
	sp.OutputVmdPicker.Dispose()
}

// type sizingPageLayoutItem struct {
// 	walk.ContainerLayoutItemBase
// 	idealSize walk.Size // in native pixels
// }

// func (li *sizingPageLayoutItem) AsContainerLayoutItemBase() *walk.ContainerLayoutItemBase {
// 	return &li.ContainerLayoutItemBase
// }

// func (li *sizingPageLayoutItem) HeightForWidth(width int) int {
// 	return li.MinSizeForSize(walk.Size{width, li.geometry.ClientSize.Height}).Height
// }

// func (li *sizingPageLayoutItem) LayoutFlags() walk.LayoutFlags {
// 	return 0
// }

// func (li *sizingPageLayoutItem) IdealSize() walk.Size {
// 	return li.idealSize
// }
