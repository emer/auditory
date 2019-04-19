// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package audio

import (
	"errors"
	"fmt"
	"github.com/go-audio/wav"
	"os"
	"time"
)

//type Endian int32
//
//const (
//  BigEndian    = iota // Samples are big endian byte order
//  LittleEndian        // Samples are little endian byte order
//)
//
//type SoundSampleType int32
//
//const (
//  Unknown    = iota // Not set
//  SignedInt         // Samples are unsigned integers
//  UnSignedInt
//  Float
//)

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
	defer inFile.Close()

	return err
}

func (snd *Sound) SampleRate() (uint32, error) {
	if snd == nil {
		err := errors.New("Sound.SampleRate: Sound is nil")
		return 0, err
	}
	return snd.Decoder.SampleRate, nil
}

func (snd *Sound) Channels() (uint16, error) {
	if snd == nil {
		err := errors.New("Sound.Channels: Sound is nil")
		return 0, err
	}
	return snd.Decoder.NumChans, nil
}

func (snd *Sound) Duration() (time.Duration, error) {
	if snd == nil {
		err := errors.New("Sound.Duration: Sound is nil")
		return -1, err
	}
	d, err := snd.Decoder.Duration()
	if err != nil {
		return d, err
	}
	return d, nil
}
