// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"math/rand"
	"strconv"

	"github.com/emer/auditory/trm"
	"github.com/emer/etable/eplot"
	"github.com/emer/etable/etable"
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

// Synth encapsulates
type Synth struct {
	vt         trm.VocalTract
	ToolBar    *gi.ToolBar   `view:"-" desc:"the master toolbar"`
	SignalData *etable.Table `desc:"waveform data"`
	WavePlot   *eplot.Plot2D `view:"-" desc:"waveform plot"`
}

func (syn *Synth) Defaults() {
	syn.SignalData = &etable.Table{}
	syn.ConfigSignalData(syn.SignalData)
	for i := 0; i < 100; i++ {
		syn.SignalData.AddRows(1)
		syn.SignalData.SetCellFloat("Time", i, float64(i))
		syn.SignalData.SetCellFloat("Amplitude", i, rand.Float64())
	}
}

// ConfigSignalData
func (syn *Synth) ConfigSignalData(dt *etable.Table) {
	dt.SetMetaData("name", "Wave")
	dt.SetMetaData("desc", "Waveform values -1 to 1")
	dt.SetMetaData("read-only", "true")
	dt.SetMetaData("precision", strconv.Itoa(4))

	sch := etable.Schema{
		{"Time", etensor.FLOAT64, nil, nil},
		{"Amplitude", etensor.FLOAT64, nil, nil},
	}
	dt.SetFromSchema(sch, 0)
}

func (syn *Synth) ConfigWavePlot(plt *eplot.Plot2D, dt *etable.Table) *eplot.Plot2D {
	plt.Params.Title = "Waveform plot"
	plt.Params.XAxisCol = "Time"
	plt.SetTable(dt)

	// order of params: on, fixMin, min, fixMax, max
	plt.SetColParams("Amplitude", eplot.On, eplot.FixMin, -1, eplot.FloatMax, 1)

	return plt
}

////////////////////////////////////////////////////////////////////////////////////////////
// 		Gui

// ConfigGui configures the GoGi gui interface for this Aud
func (syn *Synth) ConfigGui() *gi.Window {
	width := 1600
	height := 1200

	gi.SetAppName("Synth")
	gi.SetAppAbout(`This demonstrates synthesizing a sound (phone or word)`)

	win := gi.NewMainWindow("one", "Auditory ...", width, height)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	tbar := gi.AddNewToolBar(mfr, "tbar")
	tbar.SetStretchMaxWidth()
	syn.ToolBar = tbar

	split := gi.AddNewSplitView(mfr, "split")
	split.Dim = gi.X
	split.SetStretchMax()

	sv := giv.AddNewStructView(split, "sv")
	sv.SetStruct(syn)

	tview := gi.AddNewTabView(split, "tv")

	plt := tview.AddNewTab(eplot.KiT_Plot2D, "wave").(*eplot.Plot2D)
	syn.WavePlot = syn.ConfigWavePlot(plt, syn.SignalData)

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

var TheSyn Synth

func mainrun() {
	TheSyn.Defaults()
	TheSyn.vt.Init()
	TheSyn.vt.LoadEnglishPhones()
	TheSyn.vt.SynthPhones("ee", true, true)

	win := TheSyn.ConfigGui()
	TheSyn.WavePlot.GoUpdate()
	win.StartEventLoop()
}
