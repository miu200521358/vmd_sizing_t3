package ui

import (
	"github.com/miu200521358/mlib_go/pkg/infrastructure/state"
	"github.com/miu200521358/mlib_go/pkg/interface/controller"
)

type ToolState struct {
	AppState      state.IAppState
	ControlWindow *controller.ControlWindow
}

func NewToolState(appState state.IAppState, controlWindow *controller.ControlWindow) *ToolState {
	return &ToolState{
		AppState:      appState,
		ControlWindow: controlWindow,
	}
}
