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

// Unit
type Unit struct {
	Name    string  `desc:"the CV (e.g. -- da, go, ku ...), or phones (g, ah, ix ...)"`
	Start   float64 `desc:"start time of this unit in milliseconds"`
	End     float64 `desc:"end time of this unit in milliseconds"`
	AStart  float64 `desc:"start time of this unit in milliseconds, adjusted for random start silence and any offset in audio"`
	AEnd    float64 `desc:"end time of this unit in milliseconds, adjusted for random start silence and any offset in audio"`
	Silence bool    `desc:"is silence, i.e. was desginated as a period of silence"`
	Type    string  `desc:"optional info - type of unit, phone, phoneme, word, CV (consonsant-vowel), etc"`
}

// Sequence a sequence of speech units, for example a sequence of phones or words
type Sequence struct {
	File     string  `desc:"full file path"`
	ID       string  `desc:"an id to use if the corpus has subsets or for any other purpose"`
	Sequence string  `desc:"the full sequence of CVs, Phones, Words or whatever the unit"`
	Text     string  `desc:"the full readable transcription"`
	Units    []Unit  `desc:"the units of the sequence"`
	Silence  float64 `desc:"milliseconds of silence added at start of sequence to add variability"`
	Start    float64 `desc:"start of sound in milliseconds, many files have initial silence"`
	Stop     float64 `desc:"start of final silence in milliseconds"`
}
