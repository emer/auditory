// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"

	"github.com/emer/auditory/sound"
	_ "github.com/emer/etable/etview" // include to get gui views
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/ki/ki"
	"os"
)

func main() {
	PW.New()
	if len(os.Args) > 1 {
		PW.CmdArgs() // simple assumption is that any args = no gui -- could add explicit arg if you want
	} else {
		PW.Config()
		gimain.Main(func() { // this starts gui -- requires valid OpenGL display connection (e.g., X11)
			guirun()
		})
	}
}

func guirun() {
	win := PW.ConfigGui()
	win.StartEventLoop()
}

// Params
type Params struct {
}

// Play
type Play struct {
	// gui
	Win        *gi.Window      `view:"-" desc:"main GUI window"`
	StructView *giv.StructView `view:"-" desc:"the params viewer"`
	ToolBar    *gi.ToolBar     `view:"-" desc:""`

	// required params
	Rate     int    `desc:"wav sample rate"`
	Channels int    `desc:"number of channels of wav data in file"`
	BitDepth int    `desc:"bit depth in bytes"`
	FileName string `desc:"name of wave file to play"`
}

// New creates new blank elements and initializes defaults
func (pw *Play) New() {
}

// New creates new blank elements and initializes defaults
func (pw *Play) Config() {
	// defaults
	pw.FileName = "female_ba_100ms.wav"
	pw.Rate = 44100
	pw.BitDepth = 2 // in bytes
	pw.Channels = 2
}

func (pw *Play) PlayIt() {
	_, err := os.Stat(pw.FileName)
	if err != nil {
		fmt.Printf("File: %v not found\n", pw.FileName)
	} else {
		sound.Play(pw.FileName, pw.Rate, pw.Channels, pw.BitDepth)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////
// 		Gui

// ConfigGui configures the GoGi gui interface for this Aud
func (p *Play) ConfigGui() *gi.Window {
	width := 1600
	height := 1200

	gi.SetAppName("ProcessSpeech")
	gi.SetAppAbout(`This demonstrates processing a wav file with 100ms of speech.`)

	win := gi.NewMainWindow("one", "Process Speech ...", width, height)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	tbar := gi.AddNewToolBar(mfr, "tbar")
	tbar.SetStretchMaxWidth()
	p.ToolBar = tbar

	split := gi.AddNewSplitView(mfr, "split")
	split.Dim = gi.X
	split.SetStretchMaxWidth()
	split.SetStretchMaxHeight()

	sv := giv.AddNewStructView(split, "sv")
	// parent gets signal when file chooser dialog is closed - connect to view updates on parent
	sv.SetStruct(p)
	p.StructView = sv

	tbar.AddAction(gi.ActOpts{Label: "Load Sound ", Icon: "step-fwd", Tooltip: "Open file dialog for choosing another sound file (must be .wav).", UpdateFunc: func(act *gi.Action) {
		//act.SetActiveStateUpdt(p.MoreSegments)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		exts := ".wav"
		var file gi.FileName
		giv.FileViewDialog(vp, string(file), exts, giv.DlgOpts{Title: "Open wav File", Prompt: "Open a .wav file to load sound for processing."}, nil,
			win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(gi.DialogAccepted) {
					dlg, _ := send.Embed(gi.KiT_Dialog).(*gi.Dialog)
					p.FileName = giv.FileViewDialogValue(dlg)
				}
			})
		vp.FullRender2DTree()
	})

	tbar.AddAction(gi.ActOpts{Label: "Play", Icon: "play", Tooltip: "play the wav file", UpdateFunc: func(act *gi.Action) {
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		p.PlayIt()
		vp.FullRender2DTree()
	})

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

var PW Play

func (pw *Play) CmdArgs() {
	//pw.NoGui = true

	flag.StringVar(&pw.FileName, "file", "", "wave file name (Required)")
	flag.IntVar(&pw.Rate, "rate", 44100, "most typical sample rate is 44100 but could be 11025, or 22050")
	flag.IntVar(&pw.Channels, "channels", 2, "number of channels of wav file data")
	flag.IntVar(&pw.BitDepth, "depth", 2, "bit depth in bytes")
	flag.Parse()

	if pw.FileName == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	fmt.Printf("file: %s, sample rate: %s, bit depth: %s, channels: %s\n", pw.FileName, pw.Rate, pw.BitDepth, pw.Channels)
	pw.PlayIt()
}
