// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package input

import (
	"fmt"
	"math"

	"github.com/chewxy/math32"
	"github.com/emer/auditory/sound"
)

// Input defines the sound input parameters for auditory processing
type Params struct {
	WinMs            float32 `def:"25" desc:"input window -- number of milliseconds worth of sound to filter at a time"`
	StepMs           float32 `def:"5,10,12.5" desc:"input step -- number of milliseconds worth of sound that the input is stepped along to obtain the next window sample"`
	SegmentMs        float32 `def:"100" desc:"length of full segment's worth of input -- total number of milliseconds to accumulate into a complete segment -- must be a multiple of StepMs -- input will be SegmentMs / StepMs = SegmentSteps wide in the X axis, and number of filters in the Y axis"`
	SampleRate       int     `desc:"rate of sampling in our sound input (e.g., 16000 = 16Khz) -- can initialize this from a taSound object using InitFromSound method"`
	Channels         int     `desc:"total number of channels to process"`
	Channel          int     `viewif:"Channels=1" desc:"specific channel to process, if input has multiple channels, and we only process one of them (-1 = process all)"`
	WinSamples       int     `inactive:"+" desc:"number of samples to process each step"`
	StepSamples      int     `inactive:"+" desc:"number of samples to step input by"`
	SegmentSamples   int     `inactive:"+" desc:"number of samples in a segment"`
	SegmentSteps     int     `inactive:"+" desc:"number of steps in a segment"`
	SegmentStepsPlus int     `inactive:"+" desc:"SegmentSteps plus steps overlapping next segment or for padding if no next segment"`
	Steps            []int   `inactive:"+" desc:"pre-calculated start position for each step"`
	PadValue         float32 `view:"-" desc:" use this value for padding signal`
}

//Defaults initializes the Input
func (ais *Params) Defaults() {
	ais.WinMs = 25.0
	ais.StepMs = 5.0
	ais.SegmentMs = 100.0
	ais.SampleRate = 44100
	ais.Channels = 1
	ais.Channel = 0
	ais.PadValue = 0.0
}

// ComputeSamples computes the sample counts based on time and sample rate
// signal padded with zeros to ensure complete segments
func (ais *Params) Config(signalRaw []float32) (signalPadded []float32) {
	ais.WinSamples = MSecToSamples(ais.WinMs, ais.SampleRate)
	ais.StepSamples = MSecToSamples(ais.StepMs, ais.SampleRate)
	ais.SegmentSamples = MSecToSamples(ais.SegmentMs, ais.SampleRate)
	ais.SegmentSteps = int(math.Round(float64(ais.SegmentMs / ais.StepMs)))
	ais.SegmentStepsPlus = ais.SegmentSteps + int(math.Round(float64(ais.WinSamples/ais.StepSamples)))
	tail := len(signalRaw) % ais.SegmentSamples
	padLen := ais.SegmentStepsPlus*ais.StepSamples - tail
	padLen = padLen + ais.WinSamples
	pad := make([]float32, padLen)
	for i := range pad {
		pad[i] = ais.PadValue
	}
	signalPadded = append(signalRaw, pad...)
	ais.Steps = make([]int, ais.SegmentStepsPlus)
	for i := 0; i < ais.SegmentStepsPlus; i++ {
		ais.Steps[i] = ais.StepSamples * i
	}
	return signalPadded
}

// MSecToSamples converts milliseconds to samples, in terms of sample_rate
func MSecToSamples(ms float32, rate int) int {
	return int(math.Round(float64(ms) * 0.001 * float64(rate)))
}

// SamplesToMSec converts samples to milliseconds, in terms of sample_rate
func SamplesToMSec(samples int, rate int) float32 {
	return 1000.0 * float32(samples) / float32(rate)
}

// InitFromSound loads a sound and sets the Input channel vars and sample rate
func (in *Params) InitFromSound(snd *sound.Params, nChannels int, channel int) {
	if snd == nil {
		fmt.Printf("InitFromSound: sound nil")
		return
	}
	in.SampleRate = int(snd.SampleRate())
	if nChannels < 1 {
		in.Channels = int(snd.Channels())
	} else {
		in.Channels = int(math32.Min(float32(nChannels), float32(in.Channels)))
	}
	if in.Channels > 1 {
		in.Channel = channel
	} else {
		in.Channel = 0
	}
}
