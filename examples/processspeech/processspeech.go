// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"os"
	"strings"

	"github.com/emer/auditory/agabor"
	"github.com/emer/auditory/dft"
	"github.com/emer/auditory/mel"
	"github.com/emer/auditory/sound"
	"github.com/emer/etable/etensor"
	_ "github.com/emer/etable/etview" // include to get gui views
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/ki/ki"
	"gonum.org/v1/gonum/dsp/fourier"
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
	Sound           sound.Wave
	SndProcess      sound.Process     `desc:"specifications set and derived for processing the raw auditory input"`
	Signal          etensor.Float32   `inactive:"+" desc:" the full sound input obtained from the sound input - plus any added padding"`
	Samples         etensor.Float32   `inactive:"+" desc:" a window's worth of raw sound input, one channel at a time"`
	Dft             dft.Params        `view:"no-inline"`
	Power           etensor.Float32   `view:"-" desc:" power of the dft, up to the nyquist limit frequency (1/2 input.WinSamples)"`
	LogPower        etensor.Float32   `view:"-" desc:" log power of the dft, up to the nyquist liit frequency (1/2 input.WinSamples)"`
	PowerSegment    etensor.Float32   `view:"no-inline" desc:" full segment's worth of power of the dft, up to the nyquist limit frequency (1/2 input.WinSamples)"`
	LogPowerSegment etensor.Float32   `view:"no-inline" desc:" full segment's worth of log power of the dft, up to the nyquist limit frequency (1/2 input.WinSamples)"`
	Mel             mel.Params        `view:"no-inline"`
	MelFBank        etensor.Float32   `view:"no-inline" desc:" mel scale transformation of dft_power, using triangular filters, resulting in the mel filterbank output -- the natural log of this is typically applied"`
	MelFBankSegment etensor.Float32   `view:"no-inline" desc:" full segment's worth of mel feature-bank output"`
	MelFilters      etensor.Float32   `view:"no-inline" desc:" the actual filters"`
	MfccDct         etensor.Float32   `view:"no-inline" desc:" discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
	MfccDctSegment  etensor.Float32   `view:"no-inline" desc:" full segment's worth of discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
	Gabor           agabor.Params     `viewif:"FBank.On" desc:" full set of frequency / time gabor filters -- first size"`
	GaborFilters    etensor.Float32   `viewif:"On=true" desc:"full gabor filters"`
	GaborTsr        etensor.Float32   `view:"no-inline" desc:" raw output of Gabor -- full segment's worth of gabor steps"`
	Segment         int               `inactive:"+" desc:" the current segment (i.e. one segments worth of samples) - zero is first segment"`
	FftCoefs        []complex128      `view:"-" desc:" discrete fourier transform (fft) output complex representation"`
	Fft             *fourier.CmplxFFT `view:"-" desc:" struct for fast fourier transform"`
	CurSndFile      gi.FileName       `view:"+" desc:" holds the name of the file to be loaded/processed"`

	// internal state - view:"-"
	FirstStep    bool        `view:"-" desc:" if first frame to process -- turns off prv smoothing of dft power"`
	ToolBar      *gi.ToolBar `view:"-" desc:"the master toolbar"`
	MoreSegments bool        `view:"-" desc:" are there more samples to process"`
	SndPath      string      `view:"-" desc:" use to resolve different working directories for IDE and command line execution"`
	PrevSndFile  string      `view:"-" desc:" holds the name of the previous sound file loaded"`
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
	aud.SndProcess.Params.SegmentMs = 100 // set param overrides here before calling config
	aud.SndProcess.Config(aud.Sound.SampleRate())
	aud.Dft.Initialize(aud.SndProcess.Derived.WinSamples)
	aud.Mel.Defaults()
	// override any default Mel values here - then call InitFilters
	aud.Mel.InitFilters(aud.SndProcess.Derived.WinSamples/2+1, aud.Sound.SampleRate(), &aud.MelFilters)
	aud.Samples.SetShape([]int{aud.SndProcess.Derived.WinSamples}, nil, nil)
	aud.Power.SetShape([]int{aud.SndProcess.Derived.WinSamples/2 + 1}, nil, nil)
	aud.LogPower.SetShape([]int{aud.SndProcess.Derived.WinSamples/2 + 1}, nil, nil)
	aud.PowerSegment.SetShape([]int{aud.SndProcess.Derived.SegmentStepsPlus, aud.SndProcess.Derived.WinSamples/2 + 1, aud.Sound.Channels()}, nil, nil)
	if aud.Dft.CompLogPow {
		aud.LogPowerSegment.SetShape([]int{aud.SndProcess.Derived.SegmentStepsPlus, aud.SndProcess.Derived.WinSamples/2 + 1, aud.Sound.Channels()}, nil, nil)
	}

	aud.FftCoefs = make([]complex128, aud.SndProcess.Derived.WinSamples)
	aud.Fft = fourier.NewCmplxFFT(len(aud.FftCoefs))

	aud.MelFBank.SetShape([]int{aud.Mel.FBank.NFilters}, nil, nil)
	aud.MelFBankSegment.SetShape([]int{aud.SndProcess.Derived.SegmentStepsPlus, aud.Mel.FBank.NFilters, aud.Sound.Channels()}, nil, nil)
	if aud.Mel.CompMfcc {
		aud.MfccDctSegment.SetShape([]int{aud.SndProcess.Derived.SegmentStepsPlus, aud.Mel.FBank.NFilters, aud.Sound.Channels()}, nil, nil)
		aud.MfccDct.SetShape([]int{aud.Mel.FBank.NFilters}, nil, nil)
	}

	aud.FirstStep = true
	aud.Segment = -1
	aud.MoreSegments = true

	aud.Gabor.On = true
	if aud.Gabor.On {
		aud.Gabor.Defaults(aud.SndProcess.Derived.SegmentSteps, aud.Mel.FBank.NFilters)
		if aud.SndProcess.Params.SegmentMs == 200 {
			aud.Gabor.SizeTime = 12
			aud.Gabor.SpaceTime = 4
		} // otherwise assume 6 and 2 for 100ms segments
		aud.GaborFilters.SetShape([]int{aud.Gabor.NFilters, aud.Gabor.SizeFreq, aud.Gabor.SizeTime}, nil, nil)
		aud.Gabor.RenderFilters(&aud.GaborFilters)
		tsrX := ((aud.SndProcess.Derived.SegmentSteps - 1) / aud.Gabor.SpaceTime) + 1
		tsrY := ((aud.Mel.FBank.NFilters - aud.Gabor.SizeFreq - 1) / aud.Gabor.SpaceFreq) + 1
		aud.GaborTsr.SetShape([]int{aud.Sound.Channels(), tsrY, tsrX, 2, aud.Gabor.NFilters}, nil, nil)
		aud.GaborTsr.SetMetaData("odd-row", "true")
		aud.GaborTsr.SetMetaData("grid-fill", ".9")
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
	aud.Fft.Reset(aud.SndProcess.Derived.WinSamples)
}

// LoadSound initializes the AuditoryProc with the sound loaded from file by "Sound"
func (aud *Aud) LoadSound(snd *sound.Wave) {
	if aud.Sound.Channels() > 1 {
		snd.SoundToTensor(&aud.Signal, -1)
	} else {
		snd.SoundToTensor(&aud.Signal, aud.SndProcess.Params.Channel)
	}
}

// ProcessSoundFile loads a sound from file and intializes for a new sound
// if the sound is more than one segment long call ProcessSegment followed by ApplyGabor for each segment beyond the first
func (aud *Aud) ProcessSoundFile(fn string) {
	aud.PrevSndFile = string(aud.CurSndFile)
	aud.CurSndFile = gi.FileName(fn)
	err := aud.Sound.Load(fn)
	if err != nil {
		return
	}
	aud.LoadSound(&aud.Sound)
	aud.Config()
	aud.Signal.Values = sound.Trim(aud.Signal.Values, aud.Sound.SampleRate(), 1.0, 100, 300)
	aud.SndProcess.Pad(aud.Signal.Values)
	aud.ProcessSegment()
	aud.ApplyGabor()
	aud.ToolBar.UpdateActions()
}

// ProcessSegment processes the entire segment's input by processing a small overlapping set of samples on each pass
func (aud *Aud) ProcessSegment() {
	cf := string(aud.CurSndFile)
	if strings.Compare(cf, aud.PrevSndFile) != 0 {
		aud.ProcessSoundFile(string(aud.CurSndFile))
	} else if aud.MoreSegments == false {
		aud.ProcessSoundFile(string(aud.CurSndFile)) // start over - same file
	} else {
		moreSamples := true
		aud.Segment++
		for ch := int(0); ch < aud.Sound.Channels(); ch++ {
			for s := 0; s < int(aud.SndProcess.Derived.SegmentStepsPlus); s++ {
				moreSamples = aud.ProcessStep(ch, s)
				if !moreSamples {
					aud.MoreSegments = false
					break
				}
			}
		}
		remaining := len(aud.Signal.Values) - aud.SndProcess.Derived.SegmentSamples*(aud.Segment+1)
		if remaining < aud.SndProcess.Derived.SegmentSamples {
			aud.MoreSegments = false
		}
	}
}

// ProcessStep processes a step worth of sound input from current input_pos, and increment input_pos by input.step_samples
// Process the data by doing a fourier transform and computing the power spectrum, then apply mel filters to get the frequency
// bands that mimic the non-linear human perception of sound
func (aud *Aud) ProcessStep(ch, step int) bool {
	available := aud.SoundToWindow(aud.Segment, aud.SndProcess.Derived.Steps[step], ch)
	aud.Dft.Filter(int(ch), int(step), &aud.Samples, aud.FirstStep, aud.SndProcess.Derived.WinSamples, aud.FftCoefs, aud.Fft, &aud.Power, &aud.LogPower, &aud.PowerSegment, &aud.LogPowerSegment)
	aud.Mel.Filter(int(ch), int(step), &aud.Samples, &aud.MelFilters, &aud.Power, &aud.MelFBankSegment, &aud.MelFBank, &aud.MfccDctSegment, &aud.MfccDct)
	aud.FirstStep = false
	return available
}

// ApplyGabor convolves the gabor filters with the mel output
func (aud *Aud) ApplyGabor() {
	if aud.Gabor.On {
		for ch := int(0); ch < aud.Sound.Channels(); ch++ {
			agabor.Conv(ch, aud.Gabor, aud.SndProcess.Derived.SegmentStepsPlus, &aud.GaborTsr, aud.Mel.FBank.NFilters, &aud.GaborFilters, &aud.MelFBankSegment)
		}
	}
}

// SoundToWindow gets sound from SignalRaw at given position and channel
func (aud *Aud) SoundToWindow(segment, stepOffset, ch int) bool {
	if aud.Signal.NumDims() == 1 {
		start := segment*aud.SndProcess.Derived.SegmentSamples + stepOffset // segment zero based
		end := start + aud.SndProcess.Derived.WinSamples
		aud.Samples.Values = aud.Signal.Values[start:end]
	} else {
		// ToDo: implement
		log.Printf("SoundToWindow: else case not implemented - please report this issue")
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

	win := gi.NewMainWindow("one", "Process Speech ...", width, height)

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
	// parent gets signal when file chooser dialog is closed - connect to view updates on parent
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
	TheSP.SndProcess.Defaults()
	TheSP.CurSndFile = gi.FileName(TheSP.SndPath + "bug.wav")
	TheSP.ProcessSoundFile(string(TheSP.CurSndFile))

	win := TheSP.ConfigGui()
	win.StartEventLoop()
}
