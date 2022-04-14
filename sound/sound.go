// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sound

import (
	"log"
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
	Buf *audio.IntBuffer `inactive:"+"`
}

// Load loads the sound file and decodes it
func (snd *Wave) Load(fn string) error {
	f, err := os.Open(fn)
	if err != nil {
		log.Printf("sound.Load: couldn't open %s %v", fn, err)
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

// WriteWave encodes the signal data and writes it to file using the sample rate and
// other values of the buf object
func (snd *Wave) WriteWave(fn string) error {
	out, err := os.Create(fn)
	if err != nil {
		log.Printf("unable to create %s: %v", fn, err)
		return err
	}

	PCM := 1
	e := wav.NewEncoder(out, snd.SampleRate(), snd.Buf.SourceBitDepth, snd.Channels(), PCM)
	if err = e.Write(snd.Buf); err != nil {
		log.Printf("Encoding failed on write: %v", err)
		return err
	}

	if err = e.Close(); err != nil {
		log.Printf("could not close wav file encoder")
		out.Close()
		return err
	}
	out.Close()
	return nil
}

// SampleRate returns the sample rate of the sound or 0 is snd is nil
func (snd *Wave) SampleRate() int {
	if snd == nil {
		log.Printf("sound.SampleRate: Sound is nil")
		return 0
	}
	return int(snd.Buf.Format.SampleRate)
}

// SampleSize returns the sample rate of the sound or 0 is snd is nil
func (snd *Wave) SampleSize() int {
	if snd == nil {
		log.Printf("sound.SampleSize: Sound is nil")
		return 0
	}
	return 16
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
