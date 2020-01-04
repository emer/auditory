// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sound

import (
	"log"
	"math"
	"os"

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
	Buf *audio.IntBuffer
}

// Load loads the sound file and decodes it
func (snd *Wave) Load(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		log.Printf("sound.Load: couldn't open %s %v", filename, err)
		return err
	}
	defer f.Close()
	d := wav.NewDecoder(f)
	snd.Buf, err = d.FullPCMBuffer()
	if err != nil {
		log.Fatal(err)
	}

	return err
}

// SampleRate returns the sample rate of the sound or 0 is snd is nil
func (snd *Wave) SampleRate() int {
	if snd == nil {
		log.Printf("sound.SampleRate: Sound is nil")
		return 0
	}
	return int(snd.Buf.Format.SampleRate)
}

// Channels returns the number of channels in the wav data or 0 is snd is nil
func (snd *Wave) Channels() int {
	if snd == nil {
		log.Printf("sound.Channels: Sound is nil")
		return 0
	}
	return int(snd.Buf.Format.NumChannels)
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
	nFrames := snd.Buf.NumFrames()

	if channel < 0 && snd.Channels() > 1 { // multiple channels and we process all of them
		shape := make([]int, 2)
		shape[0] = snd.Channels()
		shape[1] = nFrames
		samples.SetShape(shape, nil, nil)
		idx := 0
		for i := 0; i < nFrames; i++ {
			for c := 0; c < snd.Channels(); c, idx = c+1, idx+1 {
				samples.SetFloat([]int{c, i}, float64(snd.GetFloatAtIdx(snd.Buf, idx)))
			}
		}
	} else { // only process one channel
		shape := make([]int, 1)
		shape[0] = nFrames
		samples.SetShape(shape, nil, nil)

		if snd.Channels() == 1 { // there is only one channel!
			for i := 0; i < nFrames; i++ {
				samples.SetFloat1D(i, float64(snd.GetFloatAtIdx(snd.Buf, i)))
			}
		} else {
			idx := 0
			for i := 0; i < nFrames; i++ { // process a specific channel
				samples.SetFloat1D(i, float64(snd.GetFloatAtIdx(snd.Buf, idx+channel)))
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

type Process struct {
	Params  Params
	Derived Derived
}

// Params defines the sound input parameters for auditory processing
type Params struct {
	WinMs     float32 `def:"25" desc:"input window -- number of milliseconds worth of sound to filter at a time"`
	StepMs    float32 `def:"5,10,12.5" desc:"input step -- number of milliseconds worth of sound that the input is stepped along to obtain the next window sample"`
	SegmentMs float32 `def:"100" desc:"length of full segment's worth of input -- total number of milliseconds to accumulate into a complete segment -- must be a multiple of StepMs -- input will be SegmentMs / StepMs = SegmentSteps wide in the X axis, and number of filters in the Y axis"`
	Channel   int     `viewif:"Channels=1" desc:"specific channel to process, if input has multiple channels, and we only process one of them (-1 = process all)"`
	PadValue  float32 `inactive:"+" desc:"use this value for padding signal`
}

// Derived are values calculated from user settable params and auditory input
type Derived struct {
	WinSamples       int   `inactive:"+" desc:"number of samples to process each step"`
	StepSamples      int   `inactive:"+" desc:"number of samples to step input by"`
	SegmentSamples   int   `inactive:"+" desc:"number of samples in a segment"`
	SegmentSteps     int   `inactive:"+" desc:"number of steps in a segment"`
	SegmentStepsPlus int   `inactive:"+" desc:"SegmentSteps plus steps overlapping next segment or for padding if no next segment"`
	Steps            []int `inactive:"+" desc:"pre-calculated start position for each step"`
}

//
// Defaults initializes the Input
func (sp *Process) Defaults() {
	sp.Params.WinMs = 25.0
	sp.Params.StepMs = 5.0
	sp.Params.SegmentMs = 100.0
	sp.Params.Channel = 0
	sp.Params.PadValue = 0.0
}

// Config computes the sample counts based on time and sample rate
// Start and End of non-silent signal found
// Signal padded with zeros to ensure complete segments
func (sp *Process) Config(rate int) {
	sp.Derived.WinSamples = MSecToSamples(sp.Params.WinMs, rate)
	sp.Derived.StepSamples = MSecToSamples(sp.Params.StepMs, rate)
	sp.Derived.SegmentSamples = MSecToSamples(sp.Params.SegmentMs, rate)
	sp.Derived.SegmentSteps = int(math.Round(float64(sp.Params.SegmentMs / sp.Params.StepMs)))
	sp.Derived.SegmentStepsPlus = sp.Derived.SegmentSteps + int(math.Round(float64(sp.Derived.WinSamples/sp.Derived.StepSamples)))
}

// MSecToSamples converts milliseconds to samples, in terms of sample_rate
func MSecToSamples(ms float32, rate int) int {
	return int(math.Round(float64(ms) * 0.001 * float64(rate)))
}

// SamplesToMSec converts samples to milliseconds, in terms of sample_rate
func SamplesToMSec(samples int, rate int) float32 {
	return 1000.0 * float32(samples) / float32(rate)
}

// Trim finds "silence" at the start and the end of the signal
// and returns the trimmed signal including the lead and tail silence up to maxSilence.
// if the sum over the duration is greater than the threshold value we have found the start/end
// duration is the number of samples to sum.
// maxSilence is the amount of silence to leave if lead or tail silence is longer than max.
func Trim(signal []float32, rate int, threshold float32, duration int, maxSilence int) (trimmed []float32) {
	siglen := len(signal)
	start := 0
	end := siglen
	sum := float32(0.0)
	for s := 0; s < duration; s++ {
		sum += signal[s]
	}
	if sum > threshold {
		start = 0 // the start
	} else { // keep looking
		front := 0
		back := front + duration
		for i := front; i < siglen; i++ {
			sum = sum + signal[back] - signal[front]
			if sum > threshold {
				start = front
				break
			}
			front++
			back++
		}
	}
	if start > maxSilence { // otherwise start is original start of signal
		start = start - maxSilence
	}

	// calculate where we really want to end
	sum = float32(0.0)
	for s := siglen - duration; s < siglen; s++ {
		sum += signal[s]
	}
	if sum > threshold {
		end = siglen - 1 // the very end
	} else { // keep looking
		front := siglen - duration - 1
		back := front + duration
		for i := front; i > start; i-- {
			sum = sum + signal[front] - signal[back]
			if sum > threshold {
				end = back
				break
			}
			front--
			back--
		}
	}
	if siglen-end > maxSilence { // otherwise end is original end of signal
		end = end + maxSilence
	}

	return signal[start:end]
}

// Pad pads the signal so that the length of signal divided by segment has no remainder
func (sp *Process) Pad(signal []float32) (padded []float32) {
	siglen := len(signal)
	tail := siglen % sp.Derived.SegmentSamples

	padLen := 0
	if tail < sp.Derived.WinSamples { // less than one window remaining - cut it off
		padLen = 0
	} else {
		padLen = sp.Derived.SegmentStepsPlus*sp.Derived.StepSamples - tail // more than one window remaining - keep and pad
	}

	padLen = padLen + sp.Derived.WinSamples
	pad := make([]float32, padLen)
	for i := range pad {
		pad[i] = sp.Params.PadValue
	}
	padded = append(signal, pad...)
	sp.Derived.Steps = make([]int, sp.Derived.SegmentStepsPlus)
	for i := 0; i < sp.Derived.SegmentStepsPlus; i++ {
		sp.Derived.Steps[i] = sp.Derived.StepSamples * i
	}
	return padded
}
