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

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type TrmConfig struct {
	OutputRate float64 `desc:"output sample rate (22.05, 44.1)"`
	Volume     float64 `desc:"master volume (0 - 60 dB)"`
	Channels   int     `desc:"# of sound output channels (1, 2)"`
	Balance    float64 `desc:"stereo balance (-1 to +1)"`
	Waveform   float64 `desc:"GS waveform type (0=PULSE, 1=SINE)"`

	VtlOffset   float64 `desc:"tube length offset"`
	Temperature float64 `desc:"tube temperature (25 - 40 C)"`
	LossFactor  float64 `desc:"junction loss factor in (0 - 5 %)"`

	MouthCoef    float64 `desc:"mouth aperture coefficient"`
	NoseCoef     float64 `desc:"nose aperture coefficient"`
	ThroatCutoff float64 `desc:"throat lp cutoff (50 - nyquist Hz)"`
	ThroatVolume float64 `desc:"throat volume (0 - 48 dB)"`

	NoiseModulation int     `desc:"pulse mod. of noise (0=OFF, 1=ON)"`
	MixOffset       float64 `desc:"noise crossmix offset (30 - 60 dB)"`

	// Parameters that depend on the voice.
	GlottalPulseTp    float64 `desc:"% glottal pulse rise time"`
	GlottalPulseTnMin float64 `desc:"% glottal pulse fall time minimum"`
	GlottalPulseTnMax float64 `desc:"% glottal pulse fall time maximum"`
	Breathiness       float64 `desc:"% glottal source breathiness"`
	VocalTractLength  float64
	RefGlottalPitch   float64
	ApertureRadius    float64 `desc:"aperture scl. radius (3.05 - 12 cm)"`

	// ToDo: nose sections should be a lookup
	NoseRadii [6]float64 `desc:"fixed nose radii (0 - 3 cm)"`
	// ToDo: also shouldn't be hardcoded
	RadiusCoefs          [8]float64
	GlobalRadiusCoef     float64
	GlobalNoseRadiusCoef float64
}

func (trm *TrmConfig) Defaults() {

}

// Load will be passed data/en/trm_control_model.config or equivalent file
func (trm *TrmConfig) Load(pathTrm, pathVoice string) {
	fmt.Println("trm config load")
	trm.OpenJSON(pathTrm)
	// ToDo: voice name should be gotten from controlconfig previously loaded
	trm.OpenJSON(pathVoice)

}

// OpenJSON opens model config from a JSON-formatted file (i.e. model params)
func (trm *TrmConfig) OpenJSON(fn string) error {
	b, err := ioutil.ReadFile(string(fn))
	if err != nil {
		return err
	}
	rval := json.Unmarshal(b, trm)
	return rval
}
