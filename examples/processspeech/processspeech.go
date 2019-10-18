// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"os"
	"strings"

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
	Sound           sound.Params
	Input           input.Params
	Signal          etensor.Float32 `inactive:"+" desc:" the full sound input obtained from the sound input - plus any added padding"`
	WindowIn        etensor.Float32 `inactive:"+" desc:" the raw sound input, one channel at a time"`
	Dft             dft.Params      `view:"no-inline"`
	Power           etensor.Float32 `view:"-" desc:" power of the dft, up to the nyquist limit frequency (1/2 input.WinSamples)"`
	LogPower        etensor.Float32 `view:"-" desc:" log power of the dft, up to the nyquist liit frequency (1/2 input.WinSamples)"`
	PowerSegment    etensor.Float32 `view:"no-inline" desc:" full segment's worth of power of the dft, up to the nyquist limit frequency (1/2 input.WinSamples)"`
	LogPowerSegment etensor.Float32 `view:"no-inline" desc:" full segment's worth of log power of the dft, up to the nyquist limit frequency (1/2 input.WinSamples)"`
	Mel             mel.Params      `view:"no-inline"`
	MelFBankSegment etensor.Float32 `view:"no-inline" desc:" full segment's worth of mel feature-bank output"`
	MfccDctSegment  etensor.Float32 `view:"no-inline" desc:" full segment's worth of discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
	Gabor           agabor.Params   `viewif:"FBank.On" desc:" full set of frequency / time gabor filters -- first size"`
	GaborTsr        etensor.Float32 `view:"no-inline" desc:" raw output of Gabor -- full segment's worth of gabor steps"`
	Segment         int             `inactive:"+" desc:" the current segment (i.e. one segments worth of samples) - zero is first segment"`

	// internal state - view:"-"
	FirstStep    bool        `view:"-" desc:" if first frame to process -- turns off prv smoothing of dft power"`
	ToolBar      *gi.ToolBar `view:"-" desc:"the master toolbar"`
	MoreSegments bool        `view:"-" desc:" are there more samples to process"`
	SndPath      string      `view:"-" desc:" use to resolve different working directories for IDE and command line execution"`
}

func (aud *Aud) SetPath() {
	// this code makes sure that the file is opened regardless of running from goland or terminal
	// as the working directory will be different in the 2 environments
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	aud.SndPath = ""
	if strings.HasSuffix(dir, "auditory") {
		aud.SndPath = dir + "/examples/processspeech/sounds/"
	} else {
		aud.SndPath = dir + "/sounds/"
	}
}

func (aud *Aud) Config() {
	aud.Signal.Values = aud.Input.Config(aud.Signal.Values)
	aud.Dft.Initialize(aud.Input.WinSamples, aud.Input.SampleRate)
	aud.Mel.Initialize(aud.Input.WinSamples/2+1, aud.Input.WinSamples, aud.Input.SampleRate, true)

	aud.WindowIn.SetShape([]int{aud.Input.WinSamples}, nil, nil)
	aud.Power.SetShape([]int{aud.Input.WinSamples/2 + 1}, nil, nil)
	aud.LogPower.SetShape([]int{aud.Input.WinSamples/2 + 1}, nil, nil)
	aud.PowerSegment.SetShape([]int{aud.Input.SegmentStepsPlus, aud.Input.WinSamples/2 + 1, aud.Input.Channels}, nil, nil)
	if aud.Dft.CompLogPow {
		aud.LogPowerSegment.SetShape([]int{aud.Input.SegmentStepsPlus, aud.Input.WinSamples/2 + 1, aud.Input.Channels}, nil, nil)
	}

	aud.MelFBankSegment.SetShape([]int{aud.Input.SegmentStepsPlus, aud.Mel.FBank.NFilters, aud.Input.Channels}, nil, nil)
	if aud.Mel.CompMfcc {
		aud.MfccDctSegment.SetShape([]int{aud.Input.SegmentStepsPlus, aud.Mel.FBank.NFilters, aud.Input.Channels}, nil, nil)
	}
	aud.FirstStep = true
	aud.Segment = -1
	aud.MoreSegments = true

	aud.Gabor.Initialize(aud.Input.SegmentSteps, aud.Mel.FBank.NFilters)
	aud.Gabor.On = true

	if aud.Gabor.On {
		aud.GaborTsr.SetShape([]int{aud.Input.Channels, aud.Gabor.Shape.Y, aud.Gabor.Shape.X, 2, aud.Gabor.NFilters}, nil, nil)
	}
}

// Initialize sets all the tensor result data to zeros
func (aud *Aud) Initialize() {
	aud.Power.SetZeros()
	aud.LogPower.SetZeros()
	aud.PowerSegment.SetZeros()
	aud.LogPowerSegment.SetZeros()
	aud.MelFBankSegment.SetZeros()
	aud.MfccDctSegment.SetZeros()
}

// LoadSound initializes the AuditoryProc with the sound loaded from file by "Sound"
func (aud *Aud) LoadSound(snd *sound.Params) {
	if aud.Input.Channels > 1 {
		snd.SoundToMatrix(&aud.Signal, -1)
	} else {
		snd.SoundToMatrix(&aud.Signal, aud.Input.Channel)
	}
}

// ProcessSegment processes the entire segment's input by processing a small overlapping set of samples on each pass
func (aud *Aud) ProcessSegment() {
	aud.Initialize()
	moreSamples := true
	aud.Segment++
	for ch := int(0); ch < aud.Input.Channels; ch++ {
		for s := 0; s < int(aud.Input.SegmentStepsPlus); s++ {
			moreSamples = aud.ProcessStep(ch, s)
			if !moreSamples {
				aud.MoreSegments = false
				break
			}
		}
	}
	remaining := len(aud.Signal.Values) - aud.Input.SegmentSamples*(aud.Segment+1)
	if remaining < aud.Input.SegmentSamples {
		aud.MoreSegments = false
	}
}

// ProcessStep processes a step worth of sound input from current input_pos, and increment input_pos by input.step_samples
// Process the data by doing a fourier transform and computing the power spectrum, then apply mel filters to get the frequency
// bands that mimic the non-linear human perception of sound
func (aud *Aud) ProcessStep(ch int, step int) bool {
	available := aud.SoundToWindow(aud.Segment, aud.Input.Steps[step], ch)
	aud.Dft.Filter(int(ch), int(step), &aud.WindowIn, aud.FirstStep, aud.Input.WinSamples, &aud.Power, &aud.LogPower, &aud.PowerSegment, &aud.LogPowerSegment)
	aud.Mel.Filter(int(ch), int(step), &aud.WindowIn, &aud.Power, &aud.MelFBankSegment, &aud.MfccDctSegment)
	aud.FirstStep = false
	return available
}

// ApplyGabor convolves the gabor filters with the mel output
func (aud *Aud) ApplyGabor() {
	if aud.Gabor.On {
		for ch := int(0); ch < aud.Input.Channels; ch++ {
			agabor.Conv(ch, aud.Gabor, aud.Input, &aud.GaborTsr, aud.Mel.FBank.NFilters, &aud.MelFBankSegment)
		}
	}
}

// SoundToWindow gets sound from SoundFull at given position and channel, into WindowIn
func (aud *Aud) SoundToWindow(segment, stepOffset int, ch int) bool {
	if aud.Signal.NumDims() == 1 {
		start := segment*aud.Input.SegmentSamples + stepOffset // segment zero based
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

	tbar.AddAction(gi.ActOpts{Label: "Next Segment", Icon: "step-fwd", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(aud.MoreSegments)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		aud.ProcessSegment()
		aud.ApplyGabor()
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
	TheSP.SetPath()
	TheSP.Input.Defaults()
	fn := TheSP.SndPath + "bug.wav"
	TheSP.Sound.Load(fn)
	TheSP.LoadSound(&TheSP.Sound)
	TheSP.Config()
	TheSP.Input.InitFromSound(&TheSP.Sound, TheSP.Input.Channels, 0)
	TheSP.ProcessSegment() // process the first segment of sound
	TheSP.ApplyGabor()
	TheSP.ToolBar.UpdateActions()

	win := TheSP.ConfigGui()
	win.StartEventLoop()
}
