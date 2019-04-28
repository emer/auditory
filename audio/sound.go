// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package audio

import (
	"errors"
	"fmt"
	"github.com/emer/dtable/etensor"
	"github.com/go-audio/wav"
	"os"
	"time"
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

type Sound struct {
	Decoder *wav.Decoder
}

// LoadSound loads the sound file and decodes it
func (snd *Sound) LoadSound(filename string) error {
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
func (snd *Sound) IsValid() bool {
	if snd == nil {
		return false
	}
	return snd.Decoder.IsValidFile()
}

// SampleRate returns the sample rate of the sound or 0 is snd is nil
func (snd *Sound) SampleRate() uint32 {
	if snd == nil {
		fmt.Printf("Sound.SampleRate: Sound is nil")
		return 0
	}
	return snd.Decoder.SampleRate
}

// Channels returns the number of channels in the wav data or 0 is snd is nil
func (snd *Sound) Channels() uint16 {
	if snd == nil {
		fmt.Printf("Sound.Channels: Sound is nil")
		return 0
	}
	return snd.Decoder.NumChans
}

// Duration returns the duration in msec of the sound or zero if snd is nil
func (snd *Sound) Duration() time.Duration {
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
// SoundSampleType
func (snd *Sound) SampleType() SoundSampleType {
	return SignedInt
}

// SoundToMatrix converts sound data to floating point Matrix with normalized -1..1 values (unless sound is stored as a
// float natively, in which case it is not guaranteed to be normalized) -- for use in signal processing routines --
// can optionally select a specific channel (formats sound_data as a single-dimensional matrix of frames size),
// and -1 gets all available channels (formats sound_data as two-dimensional matrix with inner dimension as
// channels and outer dimension frames
func (snd *Sound) SoundToMatrix(soundData *etensor.Float32, channel uint32) bool {

	buf, err := snd.Decoder.FullPCMBuffer()
	if err != nil {
		fmt.Printf("SoundToMatrix error: %v", err)
	}
	nFrames := buf.NumFrames()
	//fmt.Printf("frames: %v\n", strconv.Itoa(int(buf.NumFrames())))

	nChannels := snd.Channels()
	// todo: come back to the sample type as this is needed to know how to access the data correctly
	// todo: look at old code GetFloatAtIdx()
	//st := snd.SampleType()
	//size := snd.Decoder.SampleBitDepth()
	if channel < 0 && nChannels > 1 {
		shape := make([]int, 2)
		shape[0] = int(nChannels)
		shape[1] = nFrames
		soundData.SetShape(shape, nil, nil)
		idx := 0
		for i := 0; i < nFrames; i++ {
			for c := 0; c < int(nChannels); c, idx = c+1, idx+1 {
				soundData.SetFloat([]int{c, i}, float64(buf.Data[idx]))
				fmt.Printf("%v", buf.Data[idx])
			}
		}
	} else {
		shape := make([]int, 1)
		shape[0] = nFrames
		soundData.SetShape(shape, nil, nil)

		if nChannels == 1 {
			for i := 0; i < nFrames; i++ {
				soundData.SetFloat([]int{i}, float64(buf.Data[i]))
				fmt.Printf("%v\n", buf.Data[i])
			}
		} else {
			idx := 0
			for i := 0; i < nFrames; i++ {
				soundData.SetFloat([]int{i}, float64(buf.Data[uint32(idx)+channel]))
				idx += int(nChannels)
			}
		}
	}
	return true
}
