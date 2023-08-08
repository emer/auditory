// Copyright (c) 2021, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package speech

import (
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Units is a collection of speech units
type Units []*Unit

var KiT_Units = kit.Types.AddType(&Units{}, UnitProps)

// UnitProps - none at this time
var UnitProps = ki.Props{}

// ToDo: consider making Type an enum with values of phone, phoneme, word, etc

// Unit some unit of sound of whatever type
type Unit struct {

	// the CV (e.g. -- da, go, ku ...), or phones (g, ah, ix ...)
	Name string `desc:"the CV (e.g. -- da, go, ku ...), or phones (g, ah, ix ...)"`

	// start time of this unit in milliseconds
	Start float64 `desc:"start time of this unit in milliseconds"`

	// end time of this unit in milliseconds
	End float64 `desc:"end time of this unit in milliseconds"`

	// start time of this unit in milliseconds, adjusted for random start silence and any offset in audio
	AStart float64 `desc:"start time of this unit in milliseconds, adjusted for random start silence and any offset in audio"`

	// end time of this unit in milliseconds, adjusted for random start silence and any offset in audio
	AEnd float64 `desc:"end time of this unit in milliseconds, adjusted for random start silence and any offset in audio"`

	// is silence, i.e. was desginated as a period of silence
	Silence bool `desc:"is silence, i.e. was desginated as a period of silence"`

	// optional info - type of unit, phone, phoneme, word, CV (consonsant-vowel), etc
	Type string `desc:"optional info - type of unit, phone, phoneme, word, CV (consonsant-vowel), etc"`
}

// Sequence a sequence of speech units, for example a sequence of phones or words
type Sequence struct {

	// full file path
	File string `desc:"full file path"`

	// an id to use if the corpus has subsets or for any other purpose
	ID string `desc:"an id to use if the corpus has subsets or for any other purpose"`

	// the full sequence of CVs, Phones, Words or whatever the unit
	Sequence string `desc:"the full sequence of CVs, Phones, Words or whatever the unit"`

	// the full readable transcription
	Text string `desc:"the full readable transcription"`

	// the units of the sequence
	Units []Unit `desc:"the units of the sequence"`

	// milliseconds of silence added at start of sequence to add variability
	Silence float64 `desc:"milliseconds of silence added at start of sequence to add variability"`

	// start of sound in milliseconds, many files have initial silence
	Start float64 `desc:"start of sound in milliseconds, many files have initial silence"`

	// start of final silence in milliseconds
	Stop float64 `desc:"start of final silence in milliseconds"`

	// amount to adjust for random silence added (or subtracted) at start of sequence. Negative value means the existing silence was less than the random amount to be added
	Offset int `desc:"amount to adjust for random silence added (or subtracted) at start of sequence. Negative value means the existing silence was less than the random amount to be added"`

	// current time in ms as we stride through the sound sequence
	CurTime float64 `desc:"current time in ms as we stride through the sound sequence"`

	// time in ms that it will be after processing the next segment
	NextTime float64 `desc:"time in ms that it will be after processing the next segment"`
}

func (seq *Sequence) Init() {
	seq.Units = []Unit{}
}
