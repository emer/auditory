// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"

	"github.com/emer/auditory/input"
	"github.com/emer/auditory/sound"
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

// Aud encapsulates a specific auditory processing pipeline in
// use in a given case -- can add / modify this as needed
type Aud struct {
	Sound    sound.Sound
	Input    input.Input
	Channels int
}

func (aud *Aud) Defaults() {
	aud.Input.Defaults()
}

// LoadSound opens given filename as current sound
func (aud *Aud) LoadSound(filepath string) error {
	err := aud.Sound.Load(filepath)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// InitInput
func (aud *Aud) InitInput() {
	aud.Input.InitFromSound(&aud.Sound, aud.Channels, 0)
	log.Printf("total steps: %v\n", aud.Input.SegmentStepsPlus)
}

////////////////////////////////////////////////////////////////////////////////////////////
// 		Gui

// ConfigGui configures the GoGi gui interface for this Aud
func (aud *Aud) ConfigGui() *gi.Window {
	width := 1600
	height := 1200

	gi.SetAppName("loadsnd")
	gi.SetAppAbout(`This demonstrates loading a sound but will be extended to show more capabilities of the auditory package`)

	win := gi.NewWindow2D("one", "Auditory ...", width, height, true)

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
	sv.SetStruct(aud)

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

var TheAud Aud

func mainrun() {
	TheAud.Defaults()
	TheAud.LoadSound("female_la_100ms.wav")
	TheAud.Channels = int(TheAud.Sound.Channels())
	TheAud.InitInput()

	win := TheAud.ConfigGui()
	win.StartEventLoop()
}
