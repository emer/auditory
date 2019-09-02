// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/chewxy/math32"
	"github.com/emer/auditory/dft"
	"github.com/emer/auditory/input"
	"github.com/emer/auditory/mel"
	"github.com/emer/auditory/sound"
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

// Aud encapsulates a specific auditory processing pipeline in
// use in a given case -- can add / modify this as needed
type Aud struct {
	Sound                sound.Sound
	Input                input.Input
	Channels             int
	Mel                  mel.Mel `view:"no-inline"`
	Dft                  dft.Dft
	SoundFull            etensor.Float32 `inactive:"+" desc:" #NO_SAVE the full sound input obtained from the sound input"`
	WindowIn             etensor.Float32 `inactive:"+" desc:" #NO_SAVE [input.win_samples] the raw sound input, one channel at a time"`
	FirstStep            bool            `inactive:"+" desc:" #NO_SAVE if first frame to process -- turns off prv smoothing of dft power"`
	InputPos             int             `inactive:"+" desc:" #NO_SAVE current position in the SoundFull input -- in terms of sample number"`
	DftPowerTrialData    etensor.Float32 `view:"no-inline" desc:" #NO_SAVE [dft_use][input.total_steps][input.channels] full trial's worth of power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftLogPowerTrialData etensor.Float32 `view:"no-inline" desc:" #NO_SAVE [dft_use][input.total_steps][input.channels] full trial's worth of log power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	MelFBankTrialData    etensor.Float32 `view:"no-inline" desc:" #NO_SAVE [mel.n_filters][input.total_steps][input.channels] full trial's worth of mel feature-bank output -- only if using gabors"`
	MfccDctTrialData     etensor.Float32 `view:"no-inline" desc:" #NO_SAVE full trial's worth of discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
}

func (aud *Aud) Config() {
	aud.Input.Defaults()
	aud.Dft.Initialize(aud.Input.WinSamples, aud.Input.SampleRate)
	aud.Mel.Initialize(aud.Dft.SizeHalf, aud.Input.WinSamples, aud.Input.SampleRate, true)

	aud.WindowIn.SetShape([]int{aud.Input.WinSamples}, nil, nil)
	aud.DftPowerTrialData.SetShape([]int{aud.Dft.SizeHalf, aud.Input.TotalSteps, aud.Input.Channels}, nil, nil)
	if aud.Dft.CompLogPow {
		aud.DftLogPowerTrialData.SetShape([]int{aud.Dft.SizeHalf, aud.Input.TotalSteps, aud.Input.Channels}, nil, nil)
	}
	aud.MelFBankTrialData.SetShape([]int{aud.Input.TotalSteps, aud.Mel.MelFBank.NFilters, aud.Input.Channels}, nil, nil)
	if aud.Mel.CompMfcc {
		aud.MfccDctTrialData.SetShape([]int{aud.Input.TotalSteps, aud.Mel.MelFBank.NFilters, aud.Input.Channels}, nil, nil)
	}
	aud.InputPos = 0
	aud.FirstStep = true
}

// LoadSound initializes the AuditoryProc with the sound loaded from file by "Sound"
func (aud *Aud) LoadSound(snd *sound.Sound) {
	if aud.Input.Channels > 1 {
		snd.SoundToMatrix(&aud.SoundFull, -1)
	} else {
		snd.SoundToMatrix(&aud.SoundFull, aud.Input.Channel)
	}
}

// ProcessSamples processes the entire input by processing a small overlapping set of samples on each pass
func (aud *Aud) ProcessSamples() {
	for ch := int(0); ch < aud.Input.Channels; ch++ {
		for s := 0; s < int(aud.Input.TotalSteps); s++ {
			aud.ProcessStep(ch, s)
		}
	}
}

// ProcessStep processes a step worth of sound input from current input_pos, and increment input_pos by input.step_samples
// Process the data by doing a fourier transform and computing the power spectrum, then apply mel filters to get the frequency
// bands that mimic the non-linear human perception of sound
func (aud *Aud) ProcessStep(ch int, step int) bool {
	aud.SoundToWindow(aud.InputPos, ch)
	aud.Dft.Filter(int(ch), int(step), aud.WindowIn, aud.FirstStep, &aud.DftPowerTrialData, &aud.DftLogPowerTrialData)
	aud.Mel.Filter(int(ch), int(step), aud.WindowIn, &aud.Dft.DftPower, aud.FirstStep, &aud.MelFBankTrialData, &aud.MfccDctTrialData)
	aud.InputPos = aud.InputPos + aud.Input.StepSamples
	aud.FirstStep = false
	return true
}

// SoundToWindow gets sound from SoundFull at given position and channel, into WindowIn -- pads with zeros for any amount not available in the SoundFull input
func (aud *Aud) SoundToWindow(inPos int, ch int) bool {
	samplesAvail := len(aud.SoundFull.Values) - inPos
	samplesCopy := int(math32.Min(float32(samplesAvail), float32(aud.Input.WinSamples)))
	if samplesCopy > 0 {
		if aud.SoundFull.NumDims() == 1 {
			copy(aud.WindowIn.Values, aud.SoundFull.Values[inPos:samplesCopy+inPos])
		} else {
			// todo: comment from c++ version - this is not right
			//memcpy(window_in.el, (void*)&(sound_full.FastEl2d(chan, in_pos)), sz);
			fmt.Printf("SoundToWindow: else case not implemented - please report this issue")
		}
	}
	samplesCopy = int(math32.Max(float32(samplesCopy), 0)) // prevent negatives here -- otherwise overflows
	// pad remainder with zero
	zeroN := int(aud.Input.WinSamples) - int(samplesCopy)
	if zeroN > 0 {
		sz := zeroN * 4 // 4 bytes - size of float32
		copy(aud.WindowIn.Values[samplesCopy:], make([]float32, sz))
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

var TheSP Aud

func mainrun() {
	TheSP.Sound.Load("large_child_la_100ms.wav")
	TheSP.Channels = int(TheSP.Sound.Channels())
	TheSP.Config()
	TheSP.Input.InitFromSound(&TheSP.Sound, TheSP.Channels, 0)
	TheSP.LoadSound(&TheSP.Sound)
	TheSP.ProcessSamples()

	win := TheSP.ConfigGui()
	win.StartEventLoop()
}
