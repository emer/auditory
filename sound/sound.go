// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sound

import (
	"errors"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/emer/etable/etensor"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

type Endian int32

const (
	BigEndian    = iota // Samples are big endian byte order
	LittleEndian        // Samples are little endian byte order
)

type SoundSampleType int32

const (
	Unknown   = iota // Not set
	SignedInt        // Samples are signed integers
	UnSignedInt
	Float
)

type Wave struct {
	Decoder *wav.Decoder
}

// Load loads the sound file and decodes it
func (snd *Wave) Load(filename string) error {
	inFile, err := os.Open(filename)
	if err != nil {
		fmt.Printf("couldn't open %s %v", filename, err)
		return err
	}
	snd.Decoder = wav.NewDecoder(inFile)

	if snd.Decoder.IsValidFile() != true {
		err := errors.New("Sound.LoadSound: Invalid wav file")
		return err
	}
	fmt.Printf("sample rate: %v\n", snd.Decoder.SampleRate)
	duration, err := snd.Decoder.Duration()
	fmt.Printf("duration: %v\n", duration)
	//defer inFile.Close()

	return err
}

// IsValid returns false if the sound is not a valid sound
func (snd *Wave) IsValid() bool {
	if snd == nil {
		return false
	}
	return snd.Decoder.IsValidFile()
}

// SampleRate returns the sample rate of the sound or 0 is snd is nil
func (snd *Wave) SampleRate() int {
	if snd == nil {
		fmt.Printf("Sound.SampleRate: Sound is nil")
		return 0
	}
	return int(snd.Decoder.SampleRate)
}

// Channels returns the number of channels in the wav data or 0 is snd is nil
func (snd *Wave) Channels() int {
	if snd == nil {
		fmt.Printf("Sound.Channels: Sound is nil")
		return 0
	}
	return int(snd.Decoder.NumChans)
}

// Duration returns the duration in msec of the sound or zero if snd is nil
func (snd *Wave) Duration() time.Duration {
	if snd == nil {
		fmt.Printf("Sound.Duration: Sound is nil")
		return 0
	}
	d, err := snd.Decoder.Duration()
	if err != nil {
		return d
	}
	return 0
}

// todo: return to this
// SampleType
func (snd *Wave) SampleType() SoundSampleType {
	return SignedInt
}

// SoundToTensor converts sound data to floating point etensor with normalized -1..1 values (unless sound is stored as a
// float natively, in which case it is not guaranteed to be normalized) -- for use in signal processing routines --
// can optionally select a specific channel (formats sound_data as a single-dimensional matrix of frames size),
// and -1 gets all available channels (formats sound_data as two-dimensional matrix with outer dimension as
// channels and inner dimension frames
func (snd *Wave) SoundToTensor(samples *etensor.Float32, channel int) bool {
	buf, err := snd.Decoder.FullPCMBuffer()
	if err != nil {
		fmt.Printf("SoundToMatrix error: %v", err)
		return false
	}
	nFrames := buf.NumFrames()

	if channel < 0 && snd.Channels() > 1 { // multiple channels and we process all of them
		shape := make([]int, 2)
		shape[0] = snd.Channels()
		shape[1] = nFrames
		samples.SetShape(shape, nil, nil)
		idx := 0
		for i := 0; i < nFrames; i++ {
			for c := 0; c < snd.Channels(); c, idx = c+1, idx+1 {
				samples.SetFloat([]int{c, i}, float64(snd.GetFloatAtIdx(buf, idx)))
			}
		}
	} else { // only process one channel
		shape := make([]int, 1)
		shape[0] = nFrames
		samples.SetShape(shape, nil, nil)

		if snd.Channels() == 1 { // there is only one channel!
			for i := 0; i < nFrames; i++ {
				samples.SetFloat1D(i, float64(snd.GetFloatAtIdx(buf, i)))
			}
		} else {
			idx := 0
			for i := 0; i < nFrames; i++ { // process a specific channel
				samples.SetFloat1D(i, float64(snd.GetFloatAtIdx(buf, idx+channel)))
				idx += snd.Channels()
			}
		}
	}
	return true
}

// GetFloatAtIdx
func (snd *Wave) GetFloatAtIdx(buf *audio.IntBuffer, idx int) float32 {
	if buf.SourceBitDepth == 32 {
		return float32(buf.Data[idx]) / float32(0x7FFFFFFF)
	} else if buf.SourceBitDepth == 24 {
		return float32(buf.Data[idx]) / float32(0x7FFFFF)
	} else if buf.SourceBitDepth == 16 {
		return float32(buf.Data[idx]) / float32(0x7FFF)
	} else if buf.SourceBitDepth == 8 {
		return float32(buf.Data[idx]) / float32(0x7F)
	}
	return 0
}

// Input defines the sound input parameters for auditory processing
type Params struct {
	WinMs            float32 `def:"25" desc:"input window -- number of milliseconds worth of sound to filter at a time"`
	StepMs           float32 `def:"5,10,12.5" desc:"input step -- number of milliseconds worth of sound that the input is stepped along to obtain the next window sample"`
	SegmentMs        float32 `def:"100" desc:"length of full segment's worth of input -- total number of milliseconds to accumulate into a complete segment -- must be a multiple of StepMs -- input will be SegmentMs / StepMs = SegmentSteps wide in the X axis, and number of filters in the Y axis"`
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
func (sp *Params) Defaults() {
	sp.WinMs = 25.0
	sp.StepMs = 5.0
	sp.SegmentMs = 100.0
	sp.Channel = 0
	sp.PadValue = 0.0
}

// ComputeSamples computes the sample counts based on time and sample rate
// signal padded with zeros to ensure complete segments
func (sp *Params) Config(signalRaw []float32, rate int) (signalPadded []float32) {
	sp.WinSamples = MSecToSamples(sp.WinMs, rate)
	sp.StepSamples = MSecToSamples(sp.StepMs, rate)
	sp.SegmentSamples = MSecToSamples(sp.SegmentMs, rate)
	sp.SegmentSteps = int(math.Round(float64(sp.SegmentMs / sp.StepMs)))
	sp.SegmentStepsPlus = sp.SegmentSteps + int(math.Round(float64(sp.WinSamples/sp.StepSamples)))
	tail := len(signalRaw) % sp.SegmentSamples
	padLen := sp.SegmentStepsPlus*sp.StepSamples - tail
	padLen = padLen + sp.WinSamples
	pad := make([]float32, padLen)
	for i := range pad {
		pad[i] = sp.PadValue
	}
	signalPadded = append(signalRaw, pad...)
	sp.Steps = make([]int, sp.SegmentStepsPlus)
	for i := 0; i < sp.SegmentStepsPlus; i++ {
		sp.Steps[i] = sp.StepSamples * i
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
