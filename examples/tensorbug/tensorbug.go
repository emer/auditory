// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/emer/etable/etensor"
	_ "github.com/emer/etable/etview" // include to get gui views
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
)

// this is the stub main for gogi that calls our actual
// mainrun function, at end of file
func main() {
	gimain.Main(func() {
		mainrun()
	})
}

type Foo struct {
	Atensor etensor.Float32 `view:"no-inline" desc:" generic etensor"`
	Btensor etensor.Float32 `view:"no-inline" desc:" generic etensor"`
}

func (foo *Foo) Config() {
	foo.Atensor.SetShape([]int{2, 6, 2, 2, 6}, nil, nil)
	foo.Atensor.SetZeros()
	foo.Btensor.SetShape([]int{2, 2, 2, 6, 6}, nil, nil)
	foo.Btensor.SetZeros()
}

////////////////////////////////////////////////////////////////////////////////////////////
// 		Gui

// ConfigGui configures the GoGi gui interface for this Aud
func (foo *Foo) ConfigGui() *gi.Window {
	width := 1600
	height := 1200

	gi.SetAppName("Display Tensor Bug")
	gi.SetAppAbout(`This demonstrates crashing when displaying tensor.`)

	win := gi.NewWindow2D("one", "Process Speech ...", width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	tbar := gi.AddNewToolBar(mfr, "tbar")
	tbar.SetStretchMaxWidth()
	// vi.ToolBar = tbar

	split := gi.AddNewSplitView(mfr, "split")
	split.Dim = gi.X
	split.SetStretchMaxWidth()
	split.SetStretchMaxHeight()

	sv := giv.AddNewStructView(split, "sv")
	sv.SetStruct(foo)

	split.SetSplits(1)

	// main menu
	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "Window"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu.AddAppMenu(win)

	emen := win.MainMenu.ChildByName("Edit", 1).(*gi.Action)
	emen.Menu.AddCopyCutPaste(win)

	vp.UpdateEndNoSig(updt)

	win.MainMenuUpdated()
	return win
}

var theFoo Foo

func mainrun() {
	theFoo.Config()

	win := theFoo.ConfigGui()
	win.StartEventLoop()
}
