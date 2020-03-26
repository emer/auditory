// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/***************************************************************************
 *  Copyright 1991, 1992, 1993, 1994, 1995, 1996, 2001, 2002               *
 *    David R. Hill, Leonard Manzara, Craig Schock                         *
 *                                                                         *
 *  This program is free software: you can redistribute it and/or modify   *
 *  it under the terms of the GNU General Public License as published by   *
 *  the Free Software Foundation, either version 3 of the License, or      *
 *  (at your option) any later version.                                    *
 *                                                                         *
 *  This program is distributed in the hope that it will be useful,        *
 *  but WITHOUT ANY WARRANTY without even the implied warranty of         *
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the          *
 *  GNU General Public License for more details.                           *
 *                                                                         *
 *  You should have received a copy of the GNU General Public License      *
 *  along with this program.  If not, see <http://www.gnu.org/licenses/>.  *
 ***************************************************************************/
// 2014-09
// This file was copied from Gnuspeech and modified by Marcelo Y. Matuda.

// 2019-02
// This is a port to golang of the C++ Gnuspeech port by Marcelo Y. Matuda

package trmcontrolv2

import "github.com/goki/ki/kit"

// Intonation
type Intonation int

const (
	//
	IntonationNone Intonation = iota

	//
	IntonationMicro

	//
	IntonationMacro

	//
	IntonationSmooth

	//
	IntonationDrift

	//
	IntonationRandom

	IntonationN
)

//go:generate stringer -type=Intonation

var Kit_Intonation = kit.Enums.AddEnum(IntonationN, NotBitFlag, nil)

type ModelConfig struct {
	ControlRate        float64 `desc:"1.0-1000.0 input tables/second (Hz)"`
	Tempo              float64
	PitchOffset        float64
	DriftDeviation     float64
	DriftLowpassCutoff float64
	Intonation         Intonation

	// Intonation parameters.
	NotionalPitch float64
	PretonicRange float64
	PretonicLift  float64
	TonicRange    float64
	TonicMovement float64

	VoiceName       string
	Dictionary1File string
	Dictionary2File string
	Dictionary3File string
}

func (in *Intonation) Defaults() {
	in.ControlRate = 0.0
	in.Tempo = 0.0
	in.PitchOffset = 0.0
	in.DriftDeviation = 0.0
	in.DriftLowpassCutoff = 0.0
	in.Intonation = 0
	in.NotionalPitch = 0.0
	in.PretonicRange = 0.0
	in.PretonicLift = 0.0
	in.TonicRange = 0.0
	in.TonicMovement = 0.0
}

func (in *Intonation) Load(path string) {

}
