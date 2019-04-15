// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package audio

import (
	"fmt"
	"math"

	"github.com/chewxy/math32"
)

type AudInputSpec struct {
	WinMsec      float32 `desc:"#DEF_25 input window -- number of milliseconds worth of sound to filter at a time"`
	StepMsec     float32 `desc:"#DEF_5;10;12.5 input step -- number of milliseconds worth of sound that the input is stepped along to obtain the next window sample""`
	TrialMsec    float32 `desc:"#DEF_100 length of a full trial's worth of input -- total number of milliseconds to accumulate into a complete trial of activations to present to a network -- must be a multiple of step_msec -- input will be trial_msec / step_msec = trial_steps wide in the X axis, and number of filters in the Y axis"`
	BorderSteps  uint32  `desc:"number of steps before and after the trial window to preserve -- this is important when applying temporal filters that have greater temporal extent"`
	SampleRate   uint32  `desc:"rate of sampling in our sound input (e.g., 16000 = 16Khz) -- can initialize this from a taSound object using InitFromSound method"`
	Channels     uint16  `desc:"total number of channels to process"`
	Channel      uint16  `desc:"#CONDSHOW_ON_channels:1 specific channel to process, if input has multiple channels, and we only process one of them (-1 = process all)"`
	WinSamples   uint32  `desc:"#READ_ONLY #SHOW total number of samples to process (win_msec * .001 * sample_rate)"`
	StepSamples  uint32  `desc:"#READ_ONLY #SHOW total number of samples to step input by (step_msec * .001 * sample_rate)"`
	TrialSamples uint32  `desc:"#READ_ONLY #SHOW total number of samples in a trial  (trail_msec * .001 * sample_rate)"`
	TrialSteps   uint32  `desc:"#READ_ONLY #SHOW total number of steps in a trial  (trail_msec / step_msec)"`
	TotalSteps   uint32  `desc:"#READ_ONLY #SHOW 2*border_steps + trial_steps -- total in full window"`
}

//Init initializes the the AudInputSpec
func (ais *AudInputSpec) Init() {
	ais.WinMsec = 25.0
	ais.StepMsec = 5.0
	ais.TrialMsec = 100.0
	ais.BorderSteps = 12
	ais.SampleRate = 16000
	ais.Channels = 1
	ais.Channel = 0
	ais.ComputeSamples()
}

// ComputeSamples computes the sample counts based on time and sample rate
func (ais *AudInputSpec) ComputeSamples() {
	ais.WinSamples = MSecToSamples(ais.WinMsec, ais.SampleRate)
	ais.StepSamples = MSecToSamples(ais.StepMsec, ais.SampleRate)
	ais.TrialSamples = MSecToSamples(ais.TrialMsec, ais.SampleRate)
	ais.TrialSteps = uint32(math.Round(float64(ais.TrialMsec / ais.StepMsec)))
	ais.TotalSteps = 2*ais.BorderSteps + ais.TrialSteps
}

// MSecToSamples converts milliseconds to samples, in terms of sample_rate
func MSecToSamples(msec float32, rate uint32) uint32 {
	return uint32(math.Round(float64(msec) * 0.001 * float64(rate)))
}

// SamplesToMSec converts samples to milliseconds, in terms of sample_rate
func SamplesToMSec(samples uint32, rate uint32) float32 {
	return 1000.0 * float32(samples) / float32(rate)
}

// InitFromSound loads a sound and sets the AudInputSpec channel vars and sample rate
func (ais *AudInputSpec) InitFromSound(snd *Sound, nChannels uint16, channel uint16) {
	if snd == nil {
		fmt.Printf("InitFromSound: sound null")
		return
	}

	ais.SampleRate = snd.SampleRate()
	ais.ComputeSamples()
	if nChannels < 1 {
		ais.Channels = snd.Channels()
	} else {
		ais.Channels = uint16(math32.Min(float32(nChannels), float32(ais.Channels)))
	}
	if ais.Channels > 1 {
		ais.Channel = channel
	} else {
		ais.Channel = 0
	}
}
