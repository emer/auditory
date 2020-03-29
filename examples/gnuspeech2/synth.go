// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
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
	// vt      trmcontrolv1.VocalTract `view:"noinline"`
	win         *gi.Window
	ToolBar     *gi.ToolBar `view:"-" desc:"the master toolbar"`
	ModelConfig ModelConfig
	TrmConfig   TrmConfig
	// SignalData *etable.Table `desc:"waveform data"`
	// WavePlot   *eplot.Plot2D `view:"-" desc:"waveform plot"`
	// Text       string        `desc:"the text to be synthesized"`
	// Save       bool          `desc:"if true write the synthesized values to .wav file"`
	// Play       bool          `desc:"if true play the sound"`
}

func (syn *Synth) Defaults() {
	// syn.SignalData = &etable.Table{}
	// syn.ConfigSignalData(syn.SignalData)
	// syn.Save = false
	// syn.Play = false
}

func (syn *Synth) Config() {
	syn.ModelConfig.Load("./trmControl.json")
	vfp := "./voice_" + syn.ModelConfig.Voice + ".json"
	syn.TrmConfig.Load("./trm.json", vfp)
}

// ConfigSignalData
// func (syn *Synth) ConfigSignalData(dt *etable.Table) {
// 	dt.SetMetaData("name", "Wave")
// 	dt.SetMetaData("desc", "Waveform values -1 to 1")
// 	dt.SetMetaData("read-only", "true")
// 	dt.SetMetaData("precision", strconv.Itoa(4))
//
// 	sch := etable.Schema{
// 		{"Time", etensor.FLOAT64, nil, nil},
// 		{"Amplitude", etensor.FLOAT64, nil, nil},
// 	}
// 	dt.SetFromSchema(sch, 0)
// }

// func (syn *Synth) ConfigWavePlot(plt *eplot.Plot2D, dt *etable.Table) *eplot.Plot2D {
// 	plt.Params.Title = "Waveform plot"
// 	plt.Params.XAxisCol = "Time"
// 	plt.SetTable(dt)
//
// 	// order of params: on, fixMin, min, fixMax, max
// 	plt.SetColParams("Amplitude", eplot.On, eplot.FixMin, -1, eplot.FloatMax, 1)
//
// 	return plt
// }
//
// func (syn *Synth) GetWaveData() {
// 	syn.SignalData.AddRows(len(syn.vt.SynthOutput))
// 	for i := 0; i < len(syn.vt.SynthOutput); i++ {
// 		syn.SignalData.SetCellFloat("Time", i, float64(i))
// 		syn.SignalData.SetCellFloat("Amplitude", i, float64(syn.vt.Wave[i]))
// 	}
// }
//
// func (syn *Synth) Synthesize() {
// 	if len(syn.Text) == 0 {
// 		gi.PromptDialog(syn.win.Viewport, gi.DlgOpts{Title: "No text to synthesize", Prompt: fmt.Sprintf("Enter the text to synthesize in the Text field.")}, gi.AddOk, gi.NoCancel, nil, nil)
// 		return
// 	}
// 	// if len(syn.Text) > 0 {
// 	_, err := syn.vt.SynthWords(syn.Text, true, true)
// 	if err != nil {
// 		log.Println(err)
// 		gi.PromptDialog(syn.win.Viewport, gi.DlgOpts{Title: "Synthesis error", Prompt: fmt.Sprintf("Synthesis error, see console. Displaying waveform of text to that point.")}, gi.AddOk, gi.NoCancel, nil, nil)
// 	}
// 	syn.GetWaveData()
// 	syn.WavePlot.GoUpdate()
// 	if syn.Save {
// 		fn := syn.Text + ".wav"
// 		err := syn.vt.Buf.WriteWave(fn)
// 		if err != nil {
// 			fmt.Printf("File not found or error opening file: %s (%s)", fn, err)
// 		}
// 	}
// 	// }
// }

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

	// tview := gi.AddNewTabView(split, "tv")

	// plt := tview.AddNewTab(eplot.KiT_Plot2D, "wave").(*eplot.Plot2D)
	// syn.WavePlot = syn.ConfigWavePlot(plt, syn.SignalData)

	// tbar.AddAction(gi.ActOpts{Label: "Update Wave", Icon: "new"}, win.This(),
	// 	func(recv, send ki.Ki, sig int64, data interface{}) {
	// 		syn.GetWaveData()
	// 	})

	// tbar.AddAction(gi.ActOpts{Label: "Synthesize", Icon: "new"}, win.This(),
	// 	func(recv, send ki.Ki, sig int64, data interface{}) {
	// 		syn.Synthesize()
	// 	})

	split.SetSplitsList([]float32{.3, .7})

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

var Synther Synth

func mainrun() {
	Synther.Defaults()
	// Synther.vt.Init()
	Synther.Config()
	Synther.win = Synther.ConfigGui()
	Synther.win.StartEventLoop()
}
