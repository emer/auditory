// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package audio

import (
	"fmt"
	"github.com/go-audio/wav"
	"os"
)

//#include <taFiler>
//#include <float_Matrix>
//#include <QAudioFormat>
//#include <QAudioDeviceInfo>
//#include <taSound_QObj>
//
//#ifdef TA_SNDFILE
//#include <sndfile.hh>

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
	defer inFile.Close()

	return err
}

func (snd *Sound) SampleRate() uint32 {
	return snd.Decoder.SampleRate
}

func (snd *Sound) Channels() uint16 {
	return snd.Decoder.NumChans
}
