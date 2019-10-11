// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/emer/auditory/agabor"
	"github.com/emer/auditory/dft"
	"github.com/emer/auditory/input"
	"github.com/emer/auditory/mel"
	"github.com/emer/auditory/sound"
	"github.com/emer/etable/etensor"
	_ "github.com/emer/etable/etview" // include to get gui views
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/ki/ki"
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
	Sound                sound.Sound
	Input                input.Input
	Channels             int             `inactive:"+" desc:" the number of channels in signal"`
	Mel                  mel.Mel         `view:"no-inline"`
	Dft                  dft.Dft         `view:"no-inline"`
	Signal               etensor.Float32 `inactive:"+" desc:" the full sound input obtained from the sound input - plus any added padding"`
	WindowIn             etensor.Float32 `inactive:"+" desc:" [input.win_samples] the raw sound input, one channel at a time"`
	DftPowerTrialData    etensor.Float32 `view:"no-inline" desc:" [dft_use][input.total_steps][input.channels] full trial's worth of power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftLogPowerTrialData etensor.Float32 `view:"no-inline" desc:" [dft_use][input.total_steps][input.channels] full trial's worth of log power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	MelFBankTrialData    etensor.Float32 `view:"no-inline" desc:" [mel.n_filters][input.total_steps][input.channels] full trial's worth of mel feature-bank output"`
	MfccDctTrialData     etensor.Float32 `view:"no-inline" desc:" full trial's worth of discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
	Gabor                agabor.Gabor    `viewif:"FBank.On" desc:" full set of frequency / time gabor filters -- first size"`
	GaborTsr             etensor.Float32 `view:"no-inline" desc:" raw output of Gabor -- full trial's worth of gabor steps"`
	Trial                int             `inactive:"+" desc:" the current trial - zero is first trial"`

	// internal state - view:"-"
	FirstStep  bool        `view:"-" desc:" if first frame to process -- turns off prv smoothing of dft power"`
	ToolBar    *gi.ToolBar `view:"-" desc:"the master toolbar"`
	MoreTrials bool        `view:"-" desc:" are there more samples to process"`
}

func (aud *Aud) Config() {
	aud.Input.Defaults()
	padValue := .0
	aud.Signal.Values = aud.Input.Config(aud.Signal.Values, float32(padValue))
	aud.Dft.Initialize(aud.Input.WinSamples, aud.Input.SampleRate)
	aud.Mel.Initialize(aud.Dft.SizeHalf, aud.Input.WinSamples, aud.Input.SampleRate, true)

	aud.WindowIn.SetShape([]int{aud.Input.WinSamples}, nil, nil)
	aud.DftPowerTrialData.SetShape([]int{aud.Input.TrialStepsPlus, aud.Dft.SizeHalf, aud.Input.Channels}, nil, nil)
	if aud.Dft.CompLogPow {
		aud.DftLogPowerTrialData.SetShape([]int{aud.Input.TrialStepsPlus, aud.Dft.SizeHalf, aud.Input.Channels}, nil, nil)
	}
	aud.MelFBankTrialData.SetShape([]int{aud.Input.TrialStepsPlus, aud.Mel.FBank.NFilters, aud.Input.Channels}, nil, nil)
	if aud.Mel.CompMfcc {
		aud.MfccDctTrialData.SetShape([]int{aud.Input.TrialStepsPlus, aud.Mel.FBank.NFilters, aud.Input.Channels}, nil, nil)
	}
	aud.FirstStep = true
	aud.Trial = -1
	aud.MoreTrials = true

	aud.Gabor.Initialize(aud.Input.TrialSteps, aud.Mel.FBank.NFilters)
	aud.Gabor.On = true

	if aud.Gabor.On {
		aud.GaborTsr.SetShape([]int{aud.Input.Channels, aud.Gabor.Shape.Y, aud.Gabor.Shape.X, 2, aud.Gabor.NFilters}, nil, nil)
	}
}

// Initialize sets all the tensor result data to zeros
func (aud *Aud) Initialize() {
	aud.DftPowerTrialData.SetZeros()
	aud.DftLogPowerTrialData.SetZeros()
	aud.MelFBankTrialData.SetZeros()
	aud.MfccDctTrialData.SetZeros()
}

// LoadSound initializes the AuditoryProc with the sound loaded from file by "Sound"
func (aud *Aud) LoadSound(snd *sound.Sound) {
	if aud.Input.Channels > 1 {
		snd.SoundToMatrix(&aud.Signal, -1)
	} else {
		snd.SoundToMatrix(&aud.Signal, aud.Input.Channel)
	}
}

// ProcessTrial processes the entire trial's input by processing a small overlapping set of samples on each pass
func (aud *Aud) ProcessTrial() {
	aud.Initialize()
	moreSamples := true
	aud.Trial++
	for ch := int(0); ch < aud.Input.Channels; ch++ {
		for s := 0; s < int(aud.Input.TrialStepsPlus); s++ {
			moreSamples = aud.ProcessStep(ch, s)
			if !moreSamples {
				aud.MoreTrials = false
				break
			}
		}
	}
	remaining := len(aud.Signal.Values) - aud.Input.TrialSamples*(aud.Trial+1)
	if remaining < aud.Input.TrialSamples {
		aud.MoreTrials = false
	}
	aud.ApplyGabor()
	aud.ToolBar.UpdateActions()
}

// ProcessStep processes a step worth of sound input from current input_pos, and increment input_pos by input.step_samples
// Process the data by doing a fourier transform and computing the power spectrum, then apply mel filters to get the frequency
// bands that mimic the non-linear human perception of sound
func (aud *Aud) ProcessStep(ch int, step int) bool {
	available := aud.SoundToWindow(aud.Trial, aud.Input.Steps[step], ch)
	aud.Dft.Filter(int(ch), int(step), aud.WindowIn, aud.FirstStep, &aud.DftPowerTrialData, &aud.DftLogPowerTrialData)
	aud.Mel.Filter(int(ch), int(step), aud.WindowIn, &aud.Dft.Power, &aud.MelFBankTrialData, &aud.MfccDctTrialData)
	aud.FirstStep = false
	return available
}

// ApplyGabor convolves the gabor filters with the mel output
func (aud *Aud) ApplyGabor() {
	if aud.Gabor.On {
		for ch := int(0); ch < aud.Input.Channels; ch++ {
			agabor.Conv(ch, aud.Gabor, aud.Input, &aud.GaborTsr, aud.Mel.FBank.NFilters, aud.MelFBankTrialData)
		}
	}
}

// SoundToWindow gets sound from SoundFull at given position and channel, into WindowIn
func (aud *Aud) SoundToWindow(trial, stepOffset int, ch int) bool {
	if aud.Signal.NumDims() == 1 {
		start := trial*aud.Input.TrialSamples + stepOffset // trial zero based
		end := start + aud.Input.WinSamples
		if end > len(aud.Signal.Values) {
			return false
		}
		aud.WindowIn.Values = aud.Signal.Values[start:end]
	} else {
		// ToDo: implement
		fmt.Printf("SoundToWindow: else case not implemented - please report this issue")
	}
	return true
}

////////////////////////////////////////////////////////////////////////////////////////////
// 		Gui

// ConfigGui configures the GoGi gui interface for this Aud
func (aud *Aud) ConfigGui() *gi.Window {
	width := 1600
	height := 1200

	gi.SetAppName("ProcessSpeech")
	gi.SetAppAbout(`This demonstrates processing a wav file with 100ms of speech.`)

	win := gi.NewWindow2D("one", "Process Speech ...", width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	tbar := gi.AddNewToolBar(mfr, "tbar")
	tbar.SetStretchMaxWidth()
	aud.ToolBar = tbar

	split := gi.AddNewSplitView(mfr, "split")
	split.Dim = gi.X
	split.SetStretchMaxWidth()
	split.SetStretchMaxHeight()

	sv := giv.AddNewStructView(split, "sv")
	sv.SetStruct(aud)

	split.SetSplits(1)

	tbar.AddAction(gi.ActOpts{Label: "Next Trial", Icon: "step-fwd", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(aud.MoreTrials)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		aud.ProcessTrial()
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

var TheSP Aud

func mainrun() {
	TheSP.Sound.Load("bug.wav")
	TheSP.Channels = int(TheSP.Sound.Channels())
	TheSP.LoadSound(&TheSP.Sound)
	TheSP.Config()
	TheSP.Input.InitFromSound(&TheSP.Sound, TheSP.Channels, 0)
	TheSP.ProcessTrial() // process the first trial of sound

	win := TheSP.ConfigGui()
	win.StartEventLoop()
}
