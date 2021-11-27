// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/emer/auditory/agabor"
	"github.com/emer/etable/etable"
	"github.com/emer/etable/etview"
	"log"
	"math"
	"strings"

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

// Params defines the sound input parameters for auditory processing
type Params struct {
	WinMs       float32 `def:"25" desc:"input window -- number of milliseconds worth of sound to filter at a time"`
	StepMs      float32 `def:"5,10,12.5" desc:"input step -- number of milliseconds worth of sound that the input is stepped along to obtain the next window sample"`
	SegmentMs   float32 `def:"100" desc:"length of full segment's worth of input -- total number of milliseconds to accumulate into a complete segment -- must be a multiple of StepMs -- input will be SegmentMs / StepMs = SegmentSteps wide in the X axis, and number of filters in the Y axis"`
	StrideMs    float32 `def:"100" desc:"how far to move on each trial"`
	BorderSteps int     `def:"6" view:"+" desc:"overlap with previous segment"`
	Channel     int     `viewif:"Channels=1" desc:"specific channel to process, if input has multiple channels, and we only process one of them (-1 = process all)"`
	PadValue    float32

	// these are calculated
	WinSamples        int   `inactive:"+" desc:"number of samples to process each step"`
	StepSamples       int   `inactive:"+" desc:"number of samples to step input by"`
	SegmentSamples    int   `inactive:"+" desc:"number of samples in a segment"`
	StrideSamples     int   `inactive:"+" desc:"number of samples converted from StrideMS"`
	SegmentSteps      int   `inactive:"+" desc:"number of steps in a segment"`
	SegmentStepsTotal int   `inactive:"+" desc:"SegmentSteps plus steps overlapping next segment or for padding if no next segment"`
	Steps             []int `inactive:"+" desc:"pre-calculated start position for each step"`
}

// ParamDefaults initializes the Input
func (sp *SndProcess) ParamDefaults() {
	sp.Params.WinMs = 25.0
	sp.Params.StepMs = 5.0
	sp.Params.SegmentMs = 100.0
	sp.Params.Channel = 0
	sp.Params.PadValue = 0.0
	sp.Params.StrideMs = 100.0
	sp.Params.BorderSteps = 6
}

// Aud encapsulates a specific auditory processing pipeline in
// use in a given case -- can add / modify this as needed
type SndProcess struct {
	Params          Params
	Sound           sound.Wave        `view:"no-inline"`
	Signal          etensor.Float32   `view:"-" desc:" the full sound input obtained from the sound input - plus any added padding"`
	Samples         etensor.Float32   `view:"-" desc:" a window's worth of raw sound input, one channel at a time"`
	Dft             dft.Params        `view:"-" desc:" "`
	Power           etensor.Float32   `view:"-" desc:" power of the dft, up to the nyquist limit frequency (1/2 input.WinSamples)"`
	LogPower        etensor.Float32   `view:"-" desc:" log power of the dft, up to the nyquist liit frequency (1/2 input.WinSamples)"`
	PowerSegment    etensor.Float32   `view:"no-inline" desc:" full segment's worth of power of the dft, up to the nyquist limit frequency (1/2 input.WinSamples)"`
	LogPowerSegment etensor.Float32   `view:"no-inline" desc:" full segment's worth of log power of the dft, up to the nyquist limit frequency (1/2 input.WinSamples)"`
	Mel             mel.Params        `view:"no-inline"`
	MelFBank        etensor.Float32   `view:"no-inline" desc:" mel scale transformation of dft_power, using triangular filters, resulting in the mel filterbank output -- the natural log of this is typically applied"`
	MelFBankSegment etensor.Float32   `view:"no-inline" desc:" full segment's worth of mel feature-bank output"`
	MelFilters      etensor.Float32   `view:"no-inline" desc:" the actual filters"`
	MfccDct         etensor.Float32   `view:"-" desc:" discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
	MfccDctSegment  etensor.Float32   `view:"-" desc:" full segment's worth of discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
	GaborSpecs      []agabor.Filter   `view:" no-inline" desc:"array of params describing each gabor filter"`
	GaborFilters    agabor.FilterSet  `view:"no-inline" desc:"a set of gabor filters with same x and y dimensions"`
	GaborTsr        etensor.Float32   `view:"no-inline" desc:" raw output of Gabor -- full segment's worth of gabor steps"`
	GaborTab        etable.Table      `view:"no-inline" desc:"gabor filter table (view only)"`
	Segment         int               `inactive:"+" desc:"current segment of full sound (zero based)"`
	FftCoefs        []complex128      `view:"-" desc:" discrete fourier transform (fft) output complex representation"`
	Fft             *fourier.CmplxFFT `view:"-" desc:" struct for fast fourier transform"`
	SndFile         gi.FileName       `view:"-" desc:" holds the full path & name of the file to be loaded/processed"`

	// internal state - view:"-"
	FirstStep    bool `view:"-" desc:" if first frame to process -- turns off prv smoothing of dft power"`
	MoreSegments bool `view:"-" desc:" are there more samples to process"`

	// gui
	Win        *gi.Window         `view:"-" desc:"main GUI window"`
	StructView *giv.StructView    `view:"-" desc:"the params viewer"`
	ToolBar    *gi.ToolBar        `view:"-" desc:"the master toolbar"`
	PowerGrid  *etview.TensorGrid `view:"-" desc:"power grid view for the current segment"`
	MelGrid    *etview.TensorGrid `view:"-" desc:"melfbank grid view for the current segment"`
	GaborGrid  *etview.TensorGrid `view:"-" desc:"gabor grid view for the result of applying gabor filters to mel output"`
	SndName    string             `view:"inactive" desc:"just the name, no path"`
}

func (sp *SndProcess) Config() {
	sp.Params.SegmentMs = 100 // set param overrides here before calling config
	sr := sp.Sound.SampleRate()
	sp.Params.WinSamples = MSecToSamples(sp.Params.WinMs, sr)
	sp.Params.StepSamples = MSecToSamples(sp.Params.StepMs, sr)
	sp.Params.SegmentSamples = MSecToSamples(sp.Params.SegmentMs, sr)
	sp.Params.SegmentSteps = int(math.Round(float64(sp.Params.SegmentMs / sp.Params.StepMs)))
	sp.Params.SegmentStepsTotal = sp.Params.SegmentSteps + 2*sp.Params.BorderSteps
	sp.Params.StrideSamples = MSecToSamples(sp.Params.StrideMs, sr)

	sp.Dft.Initialize(sp.Params.WinSamples)
	sp.Mel.Defaults()
	// override any default Mel values here - then call InitFilters
	sp.Mel.InitFilters(sp.Params.WinSamples, sp.Sound.SampleRate(), &sp.MelFilters)
	sp.Samples.SetShape([]int{sp.Params.WinSamples}, nil, nil)
	sp.Power.SetShape([]int{sp.Params.WinSamples/2 + 1}, nil, nil)
	sp.LogPower.SetShape([]int{sp.Params.WinSamples/2 + 1}, nil, nil)
	sp.PowerSegment.SetShape([]int{sp.Params.SegmentStepsTotal, sp.Params.WinSamples/2 + 1, sp.Sound.Channels()}, nil, nil)
	if sp.Dft.CompLogPow {
		sp.LogPowerSegment.SetShape([]int{sp.Params.SegmentStepsTotal, sp.Params.WinSamples/2 + 1, sp.Sound.Channels()}, nil, nil)
	}

	sp.FftCoefs = make([]complex128, sp.Params.WinSamples)
	sp.Fft = fourier.NewCmplxFFT(len(sp.FftCoefs))

	sp.MelFBank.SetShape([]int{sp.Mel.FBank.NFilters}, nil, nil)
	sp.MelFBankSegment.SetShape([]int{sp.Params.SegmentStepsTotal, sp.Mel.FBank.NFilters, sp.Sound.Channels()}, nil, nil)
	if sp.Mel.CompMfcc {
		sp.MfccDctSegment.SetShape([]int{sp.Params.SegmentStepsTotal, sp.Mel.FBank.NFilters, sp.Sound.Channels()}, nil, nil)
		sp.MfccDct.SetShape([]int{sp.Mel.FBank.NFilters}, nil, nil)
	}

	sp.FirstStep = true
	sp.Segment = -1
	sp.MoreSegments = true

	sp.GaborSpecs = nil // in case there are some specs already
	spec := agabor.Filter{SizeX: 7, SizeY: 7, WaveLen: 2.0, Orientation: 0, SigmaWidth: 0.6, SigmaLength: 0.2, PhaseOffset: 0, CircleEdge: true}
	sp.GaborSpecs = append(sp.GaborSpecs, spec)
	spec = agabor.Filter{SizeX: 7, SizeY: 7, WaveLen: 2.0, Orientation: 0, SigmaWidth: 0.6, SigmaLength: 0.2, PhaseOffset: 1.5708, CircleEdge: true}
	sp.GaborSpecs = append(sp.GaborSpecs, spec)
	spec = agabor.Filter{SizeX: 7, SizeY: 7, WaveLen: 2.0, Orientation: 45, SigmaWidth: 0.3, SigmaLength: 0.6, PhaseOffset: 0, CircleEdge: true}
	sp.GaborSpecs = append(sp.GaborSpecs, spec)
	spec = agabor.Filter{SizeX: 7, SizeY: 7, WaveLen: 2.0, Orientation: 90, SigmaWidth: 0.3, SigmaLength: 0.6, PhaseOffset: 0, CircleEdge: true}
	sp.GaborSpecs = append(sp.GaborSpecs, spec)
	spec = agabor.Filter{SizeX: 7, SizeY: 7, WaveLen: 2.0, Orientation: 135, SigmaWidth: 0.3, SigmaLength: 0.6, PhaseOffset: 0, CircleEdge: true}
	sp.GaborSpecs = append(sp.GaborSpecs, spec)
	// and a circular one - orientation, phase and sigmaLength will be ignored if specified
	spec = agabor.Filter{SizeX: 7, SizeY: 7, WaveLen: 2.0, SigmaWidth: 0.5, Circular: true}
	sp.GaborSpecs = append(sp.GaborSpecs, spec)

	// filter size is assumed to be consistent and taken from first in the spec list
	sp.GaborFilters.SizeX = sp.GaborSpecs[0].SizeX
	sp.GaborFilters.SizeY = sp.GaborSpecs[0].SizeY
	sp.GaborFilters.StrideX = 3
	sp.GaborFilters.StrideY = 3
	sp.GaborFilters.Gain = 2
	sp.GaborFilters.Distribute = false // the 0 orientation filters will both be centered
	x := sp.GaborFilters.SizeX
	y := sp.GaborFilters.SizeY
	n := len(sp.GaborSpecs)
	sp.GaborFilters.Filters.SetShape([]int{n, y, x}, nil, nil)
	agabor.ToTensor(sp.GaborSpecs, &sp.GaborFilters)
	sp.GaborFilters.ToTable(sp.GaborFilters, &sp.GaborTab) // note: view only, testing

	tmp := sp.Params.SegmentSteps - sp.GaborFilters.SizeX
	tsrX := tmp/sp.GaborFilters.StrideX + 1
	tmp = sp.Mel.FBank.NFilters - sp.GaborFilters.SizeY
	tsrY := tmp/sp.GaborFilters.StrideY + 1
	sp.GaborTsr.SetShape([]int{sp.Sound.Channels(), tsrY, tsrX, 2, n}, nil, nil)
	sp.GaborTsr.SetMetaData("odd-row", "true")
	sp.GaborTsr.SetMetaData("grid-fill", ".9")

	// 2 reasons for this code
	// 1 - the amount of signal handed to the fft has a "border" (some extra signal) to avoid edge effects.
	// On the first step there is no signal to act as the "border" so we pad the data handed on the front.
	// 2 - signals needs to be aligned when the number when multiple signals are input (e.g. 100 and 300 ms)
	// so that the leading edge (right edge) is the same time point.
	// This code does this by generating negative offsets for the start of the processing.
	// Also see SndToWindow for the use of the step values
	strides := int(sp.Params.SegmentMs / sp.Params.StrideMs)
	stepsPerStride := int(sp.Params.StrideMs / sp.Params.StepMs)
	stepsBack := stepsPerStride*(strides-1) + sp.Params.BorderSteps
	sp.Params.Steps = make([]int, sp.Params.SegmentStepsTotal)
	for i := 0; i < sp.Params.SegmentStepsTotal; i++ {
		sp.Params.Steps[i] = sp.Params.StepSamples * (i - stepsBack)
	}
}

// Initialize sets all the tensor result data to zeros
func (sp *SndProcess) Initialize() {
	sp.Power.SetZeros()
	sp.LogPower.SetZeros()
	sp.PowerSegment.SetZeros()
	sp.LogPowerSegment.SetZeros()
	sp.MelFBankSegment.SetZeros()
	sp.MfccDctSegment.SetZeros()
	sp.Fft.Reset(sp.Params.WinSamples)
}

// LoadSound initializes the AuditoryProc with the sound loaded from file by "Sound"
func (sp *SndProcess) LoadSound(snd *sound.Wave) {
	if sp.Sound.Channels() > 1 {
		snd.SoundToTensor(&sp.Signal, -1)
	} else {
		snd.SoundToTensor(&sp.Signal, sp.Params.Channel)
	}
}

// ProcessSound loads a sound from file and intializes for a new sound
// if the sound is more than one segment long call ProcessSegment followed by ApplyGabor for each segment beyond the first
func (sp *SndProcess) ProcessSound(fn string) {
	sp.SndFile = gi.FileName(fn)
	full := string(sp.SndFile)
	i := strings.LastIndex(full, "/")
	sp.SndName = full[i+1 : len(full)]

	err := sp.Sound.Load(fn)
	if err != nil {
		return
	}
	sp.LoadSound(&sp.Sound)
	sp.Config()
	sp.Pad(sp.Signal.Values)
	sp.ProcessSegment()
	sp.ApplyGabor()
	if sp.Win != nil {
		vp := sp.Win.WinViewport2D()
		if sp.ToolBar != nil {
			sp.ToolBar.UpdateActions()
		}
		vp.SetNeedsFullRender()
	}
}

// ProcessSegment processes the entire segment's input by processing a small overlapping set of samples on each pass
func (sp *SndProcess) ProcessSegment() {
	if sp.MoreSegments == false {
		sp.ProcessSound(string(sp.SndFile)) // start over - same file
	} else {
		moreSamples := true
		sp.Segment++
		for ch := int(0); ch < sp.Sound.Channels(); ch++ {
			for s := 0; s < int(sp.Params.SegmentStepsTotal); s++ {
				moreSamples = sp.ProcessStep(ch, s)
				if !moreSamples {
					sp.MoreSegments = false
					break
				}
			}
		}
		remaining := len(sp.Signal.Values) - sp.Params.SegmentSamples*(sp.Segment+1)
		if remaining < sp.Params.SegmentSamples {
			sp.MoreSegments = false
		}
	}
}

// ProcessStep processes a step worth of sound input from current input_pos, and increment input_pos by input.step_samples
// Process the data by doing a fourier transform and computing the power spectrum, then apply mel filters to get the frequency
// bands that mimic the non-linear human perception of sound
func (sp *SndProcess) ProcessStep(ch, step int) bool {
	available := sp.SoundToWindow(sp.Segment, sp.Params.Steps[step], ch)
	sp.Dft.Filter(int(ch), int(step), &sp.Samples, sp.FirstStep, sp.Params.WinSamples, sp.FftCoefs, sp.Fft, &sp.Power, &sp.LogPower, &sp.PowerSegment, &sp.LogPowerSegment)
	sp.Mel.Filter(int(ch), int(step), &sp.Samples, &sp.MelFilters, &sp.Power, &sp.MelFBankSegment, &sp.MelFBank, &sp.MfccDctSegment, &sp.MfccDct)
	sp.FirstStep = false
	return available
}

// ApplyGabor convolves the gabor filters with the mel output
func (sp *SndProcess) ApplyGabor() {
	for ch := int(0); ch < sp.Sound.Channels(); ch++ {
		agabor.Convolve(ch, sp.Mel.FBank.NFilters, &sp.MelFBankSegment, sp.GaborFilters, &sp.GaborTsr)
	}
}

// SoundToWindow gets sound from SignalRaw at given position and channel
func (sp *SndProcess) SoundToWindow(segment, stepOffset, ch int) bool {
	if sp.Signal.NumDims() == 1 {
		start := segment*sp.Params.SegmentSamples + stepOffset // segment zero based
		end := start + sp.Params.WinSamples

		if end > len(sp.Signal.Values) {
			fmt.Println("SndToWindow: end beyond signal length!!")
			return false
		}
		var pad []float32
		if start < 0 && end <= 0 {
			pad = make([]float32, end-start)
			sp.Samples.Values = pad[0:]
		} else if start < 0 && end > 0 {
			pad = make([]float32, 0-start)
			sp.Samples.Values = pad[0:]
			sp.Samples.Values = append(sp.Samples.Values, sp.Signal.Values[0:end]...)
		} else {
			sp.Samples.Values = sp.Signal.Values[start:end]
		}
	} else {
		// ToDo: implement
		log.Printf("SoundToWindow: else case not implemented - please report this issue")
	}
	return true
}

///////////////////////////////////////////////////////////////////////////////////////////
// 		Utility Code

// Tail returns the number of samples that remain beyond the last full stride
func (sp *SndProcess) Tail(signal []float32) int {
	temp := len(signal) - sp.Params.SegmentSamples
	tail := temp % sp.Params.StrideSamples
	return tail
}

// Pad pads the signal so that the length of signal divided by stride has no remainder
func (sp *SndProcess) Pad(signal []float32) (padded []float32) {
	tail := sp.Tail(signal)
	padLen := sp.Params.SegmentSamples - sp.Params.StepSamples - tail%sp.Params.StepSamples
	pad := make([]float32, padLen)
	for i := range pad {
		pad[i] = sp.Params.PadValue
	}
	padded = append(signal, pad...)
	return padded
}

// MSecToSamples converts milliseconds to samples, in terms of sample_rate
func MSecToSamples(ms float32, rate int) int {
	return int(math.Round(float64(ms) * 0.001 * float64(rate)))
}

// SamplesToMSec converts samples to milliseconds, in terms of sample_rate
func SamplesToMSec(samples int, rate int) float32 {
	return 1000.0 * float32(samples) / float32(rate)
}

////////////////////////////////////////////////////////////////////////////////////////////
// 		Gui

// ConfigGui configures the GoGi gui interface for this Aud
func (sp *SndProcess) ConfigGui() *gi.Window {
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
	sp.ToolBar = tbar

	split := gi.AddNewSplitView(mfr, "split")
	split.Dim = gi.X
	split.SetStretchMaxWidth()
	split.SetStretchMaxHeight()

	sv := giv.AddNewStructView(split, "sv")
	// parent gets signal when file chooser dialog is closed - connect to view updates on parent
	sv.SetStruct(sp)
	sp.StructView = sv

	tv := gi.AddNewTabView(split, "tv")
	split.SetSplits(.3, .7)

	tbar.AddAction(gi.ActOpts{Label: "Load Sound ", Icon: "step-fwd", Tooltip: "Open file dialog for choosing another sound file (must be .wav).", UpdateFunc: func(act *gi.Action) {
		//act.SetActiveStateUpdt(sp.MoreSegments)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		exts := ".wav"
		giv.FileViewDialog(vp, string(sp.SndFile), exts, giv.DlgOpts{Title: "Open wav File", Prompt: "Open a .wav file to load sound for processing."}, nil,
			win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(gi.DialogAccepted) {
					dlg, _ := send.Embed(gi.KiT_Dialog).(*gi.Dialog)
					fn := giv.FileViewDialogValue(dlg)
					sp.SndFile = gi.FileName(fn)
					sp.ProcessSound(string(sp.SndFile))
				}
			})
		vp.FullRender2DTree()
	})

	tbar.AddAction(gi.ActOpts{Label: "Next Segment", Icon: "step-fwd", Tooltip: "Process the next segment of sound", UpdateFunc: func(act *gi.Action) {
		//act.SetActiveStateUpdt(sp.MoreSegments)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		sp.ProcessSegment()
		sp.ApplyGabor()
		vp.FullRender2DTree()
	})

	tbar.AddAction(gi.ActOpts{Label: "Update Gabor Filters", Icon: "step-fwd", Tooltip: "Updates the gabor filters if you change any of the gabor specs. Changes to gabor size require recompile.", UpdateFunc: func(act *gi.Action) {
		//act.SetActiveStateUpdt(sp.MoreSegments)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		sp.ProcessSegment()
		agabor.ToTensor(sp.GaborSpecs, &sp.GaborFilters)
		sp.GaborFilters.ToTable(sp.GaborFilters, &sp.GaborTab) // note: view only, testing
		vp.FullRender2DTree()
	})

	pv := tv.AddNewTab(etview.KiT_TensorGrid, "Power").(*etview.TensorGrid)
	pv.SetStretchMax()
	sp.PowerGrid = pv
	sp.LogPowerSegment.SetMetaData("grid-min", "10")
	pv.SetTensor(&sp.LogPowerSegment)

	mv := tv.AddNewTab(etview.KiT_TensorGrid, "MelFBank").(*etview.TensorGrid)
	mv.SetStretchMax()
	sp.MelGrid = mv
	mv.SetTensor(&sp.MelFBankSegment)

	//gv := tv.AddNewTab(etview.KiT_TensorGrid, "Gabor Filtering Result").(*etview.TensorGrid)
	//gv.SetStretchMax()
	//sp.GaborGrid = gv
	//gv.SetTensor(&sp.GaborTsr)

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

var TheSP SndProcess

func mainrun() {
	TheSP.ParamDefaults()
	win := TheSP.ConfigGui()
	TheSP.Win = win
	TheSP.SndFile = gi.FileName("sounds/bug.wav")
	TheSP.ProcessSound(string(TheSP.SndFile))
	win.StartEventLoop()
}
