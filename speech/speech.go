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

// Unit
type Unit struct {
	Name   string  `desc:"the CV (e.g. -- da, go, ku ...), or phones (g, ah, ix ...)"`
	Start  float64 `desc:"start time of this unit in a particular sequence in milliseconds"`
	End    float64 `desc:"end time of this unit in a particular sequence in milliseconds"`
	AStart float64 `desc:"start time of this unit in a particular sequence in milliseconds, adjusted for random start silence and any offset in audio"`
	AEnd   float64 `desc:"end time of this unit in a particular sequence in milliseconds, adjusted for random start silence and any offset in audio"`
	Type   string  `desc:"optional info - type of unit, phone, phoneme, word, CV (consonsant-vowel), etc"`
}

// Sequence a sequence of speech units, for example a sequence of phones or words
type Sequence struct {
	File     string  `desc:""`
	ID       string  `desc:"an id to use if the corpus has subsets"`
	Sequence string  `desc:"the full sequence of CVs, Phones, Words or whatever the unit"`
	Units    []Unit  `desc:"the units of the sequence"`
	Silence  float64 `desc:"milliseconds of silence added at start of sequence to add variability"`
	TimeCur  float64 `desc:"current time in milliseconds since start of sequence"`
	TimeStop float64 `desc:"start of final silence in milliseconds"`
}
