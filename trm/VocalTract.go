// Copyright (c) 2019, The GoKi Authors. All rights reserved.
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
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of         *
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the          *
 *  GNU General Public License for more details.                           *
 *                                                                         *
 *  You should have received a copy of the GNU General Public License      *
 *  along with this program.  If not, see <http://www.gnu.org/licenses/>.  *
 ***************************************************************************/
// 2014-09
// This file was copied from Gnuspeech and modified by Marcelo Y. Matuda.

/******************************************************************************
*
*     Program:       tube
*
*     Description:   Software (non-real-time) implementation of the Tube
*                    Resonance Model for speech production.
*
*     Author:        Leonard Manzara
*
*     Date:          July 5th, 1994
*
******************************************************************************/

// 2019-02
// This is a port to golang of the C++ Gnuspeech port by Marcelo Y. Matuda

package trm

import (
	"fmt"
	"math"
	"strings"

	"github.com/chewxy/math32"
	"github.com/emer/auditory/sound"
	"github.com/emer/etable/etable"
	"github.com/emer/etable/etensor"
	"github.com/go-audio/audio"
	"github.com/goki/gi/gi"
)

/*  COMPILE SO THAT INTERPOLATION NOT DONE FOR SOME CONTROL RATE PARAMETERS  */
//#define MATCH_DSP                 1

const GsTrmTubeMinRadius = 0.001
const InputVectorReserve = 128
const OutputVectorReserve = 1024
const GlottalSourcePulse = 0
const GlottalSourceSine = 1
const PitchBase = 220.0
const PitchOffset = 3
const VolMax = 60
const VtScale = 0.125
const OutputScale = 0.95
const Top = 0
const Bottom = 1

/////////////////////////////////////////////////////
//              TractParams

type TractParams struct {
	Temp         float32
	Loss         float32
	MouthCoef    float32
	NoseCoef     float32
	ThroatCutoff float32
	ThroatVol    float32
	VtlOff       float32
	MixOff       float32
	WaveForm     WaveForm
	NoiseMod     bool
}

// Defaults sets the default values for the vocal tract
func (vtc *TractParams) Defaults() {
	vtc.Temp = 32.0
	vtc.Loss = 0.8
	vtc.MouthCoef = 5000.0
	vtc.NoseCoef = 5000.0
	vtc.ThroatCutoff = 1500.0
	vtc.ThroatVol = 6.0
	vtc.VtlOff = 0.0
	vtc.WaveForm = Pulse
	vtc.NoiseMod = true
	vtc.MixOff = 48.0
}

/////////////////////////////////////////////////////
//              VoiceParams

type AgeGender int32

const (
	Male = iota
	Female
	ChildLg
	ChildSm
	Baby
)

//go:generate stringer -type=Voices

// VoiceParams are the parameters that control the quality of the voice
type VoiceParams struct {
	TractLength      float32
	GlotPulseFallMin float32
	GlotPulseFallMax float32
	GlotPitchRef     float32
	Breathiness      float32
	GlotPulseRise    float32
	ApertureRadius   float32
	NoseRadii        [6]float32 `desc:"fixed nose radii (0 - 3 cm)"`
	//Radius           float32
	NoseRadiusCoef float32 `desc:"global nose radius coefficient"`
	RadiusCoef     float32
}

// DefaultParams are the defaults, some of which don't change
func (vp *VoiceParams) Defaults() {
	vp.GlotPulseRise = 40.0
	vp.ApertureRadius = 3.05
	vp.NoseRadii[0] = 1.35
	vp.NoseRadii[1] = 1.96
	vp.NoseRadii[2] = 1.91
	vp.NoseRadii[3] = 1.3
	vp.NoseRadii[4] = 0.73
	vp.NoseRadii[5] = 0.8 // called Radius_1 in c++ code
	//vp.Radius = 0.8
	vp.NoseRadiusCoef = 1.0
	vp.RadiusCoef = 1.0
	vp.Female() // need to call some AgeGender in case caller doesn't!
}

func (vp *VoiceParams) Male() {
	vp.TractLength = 17.5
	vp.GlotPulseFallMin = 24.0
	vp.GlotPulseFallMax = 24.0
	vp.GlotPitchRef = -12.0
	vp.Breathiness = 0.5
}

func (vp *VoiceParams) Female() {
	vp.TractLength = 15.0
	vp.GlotPulseFallMin = 32.0
	vp.GlotPulseFallMax = 32.0
	vp.GlotPitchRef = 0.0
	vp.Breathiness = 1.5
}

func (vp *VoiceParams) ChildLg() {
	vp.TractLength = 12.5
	vp.GlotPulseFallMin = 24.0
	vp.GlotPulseFallMax = 24.0
	vp.GlotPitchRef = 2.5
	vp.Breathiness = 1.5
}

func (vp *VoiceParams) ChildSm() {
	vp.TractLength = 10.0
	vp.GlotPulseFallMin = 24.0
	vp.GlotPulseFallMax = 24.0
	vp.GlotPitchRef = 5.0
	vp.Breathiness = 1.5
}

func (vp *VoiceParams) Baby() {
	vp.TractLength = 7.5
	vp.GlotPulseFallMin = 24.0
	vp.GlotPulseFallMax = 24.0
	vp.GlotPitchRef = 7.5
	vp.Breathiness = 1.5
}

func (vp *VoiceParams) SetAgeGender(voice AgeGender) {
	switch voice {
	case Male:
		vp.Male()
	case Female:
		vp.Female()
	case ChildLg:
		vp.ChildLg()
	case ChildSm:
		vp.ChildSm()
	case Baby:
		vp.Baby()
	}
}

/////////////////////////////////////////////////////
//              VocalTractCtrl

type CtrlParamIdxs int32

const (
	GlotPitchIdx = iota
	GlotVolIdx
	AspVolIdx
	FricVolIdx
	FricPosIdx
	FricCfIdx
	FricBwIdx
	//Radius2Idx
	//Radius3Idx
	//Radius4Idx
	//Radius5Idx
	//Radius6Idx
	//Radius7Idx
	//Radius8Idx
	VelumIdx
	NCtrlParams
)

//go:generate stringer -type=CtrlParamIdxs

// ToDo: desc for all Radii
type VocalTractCtrl struct {
	GlotPitch float32    `desc:"ranges from -10 for phoneme k to 0 for most, with some being -2 or -1 -- called microInt in gnuspeech data files"`
	GlotVol   float32    `desc:"glottal volume (DB?) typically 60 when present and 0 when not, and sometimes 54, 43.5, 42, "`
	AspVol    float32    `desc:"aspiration volume -- typically 0 when not present and 10 when present"`
	FricVol   float32    `desc:"fricative volume -- typically 0 or .25 .4, .5, .8 but 24 for ph"`
	FricPos   float32    `desc:"ficative position -- varies continuously between 1-7"`
	FricCf    float32    `desc:"fricative center frequency ranges between 864 to 5500 with values around 1770, 2000, 2500, 4500 being common"`
	FricBw    float32    `desc:"fricative bw seems like a frequency -- common intermediate values are 600, 900, 2000, 2600"`
	Radii     [7]float32 `desc:"Radii 2-8 radius of pharynx vocal tract segment as determined by tongue etc -- typically around 1, ranging .5 - 1.7"`
	Velum     float32    `desc:"velum opening -- 1.5 when fully open, .1 when closed, and .25, .5 intermediates used"`
}

func (vtc *VocalTractCtrl) Defaults() {
	vtc.GlotPitch = 0.0
	vtc.GlotVol = 0.0
	vtc.AspVol = 0.0
	vtc.FricVol = 0.0
	vtc.FricPos = 4.0
	vtc.FricCf = 2500.0
	vtc.FricBw = 2000.0
	for i, _ := range vtc.Radii {
		vtc.Radii[i] = 1.0
	}
	vtc.Velum = 0.1
}

// ComputeDeltas computes values in this set of params as deltas from (cur - prv) * ctrl_freq
func (vtc *VocalTractCtrl) ComputeDeltas(curCtrl *VocalTractCtrl, prvCtrl *VocalTractCtrl, deltaMax *VocalTractCtrl, ctrlFreq float32) {
	// todo:
	// for(int i=0; i< N_PARAMS; i++) {
	//  float cval = cur.ParamVal(i);
	//  float pval = prv.ParamVal(i);
	//  float dmax = del_max.ParamVal(i);
	//  float& nval = ParamVal(i);
	//  nval = (cval - pval) * ctrl_freq;
	//  // if(nval > dmax) nval = dmax;
	//  // else if (nval < -dmax) nval = -dmax;
	// }
}

// UpdateFromDeltas updates values in this set of params from deltas
func (vtc *VocalTractCtrl) UpdateFromDeltas(deltas *VocalTractCtrl) {
	// todo:
	//  for(int i=0; i< N_PARAMS; i++) {
	//    float dval = del.ParamVal(i);
	//    float& nval = ParamVal(i);
	//    nval += dval;
	//  }
}

// DefaultMaxDeltas updates the default max delta values in this object (for DeltaMax field in VocalTract)
func (vtc *VocalTractCtrl) DefaultMaxDeltas() {
	cf := float32(1.0 / 501.0) // default control frequency
	// default to entire range ok for now.. fix when glitches encountered.. (comment from c++ code)
	vtc.GlotPitch = 10 * cf
	vtc.GlotVol = 60.0 * cf
	vtc.AspVol = 10.0 * cf
	vtc.FricVol = 24.0 * cf
	vtc.FricPos = 7.0 * cf
	vtc.FricCf = 3000.0 * cf
	vtc.FricBw = 4000.0 * cf
	for i, _ := range vtc.Radii {
		vtc.Radii[i] = 3.0 * cf
	}
	vtc.Velum = 1.5 * cf
}

// SetFromParams fast copy of parameters from other control params
func (vtc *VocalTractCtrl) SetFromParams(vtcSrc *VocalTractCtrl) {
	vtc.GlotPitch = vtcSrc.GlotPitch
	vtc.GlotVol = vtcSrc.GlotVol
	vtc.AspVol = vtcSrc.AspVol
	vtc.FricVol = vtcSrc.FricVol
	vtc.FricPos = vtcSrc.FricPos
	vtc.FricCf = vtcSrc.FricCf
	vtc.FricBw = vtcSrc.FricBw
	for i, _ := range vtc.Radii {
		vtc.Radii[i] = vtcSrc.Radii[i]
	}
	vtc.Velum = vtcSrc.Velum
}

// SetFromValues - order must be preserved!
func (vtc *VocalTractCtrl) SetFromValues(values []float32) {
	vtc.GlotPitch = values[0]
	vtc.GlotVol = values[1]
	vtc.AspVol = values[2]
	vtc.FricVol = values[3]
	vtc.FricPos = values[4]
	vtc.FricCf = values[5]
	vtc.FricBw = values[6]
	for i, _ := range vtc.Radii {
		vtc.Radii[i] = values[i+7]
	}
	vtc.Velum = values[14]
}

func (vtc *VocalTractCtrl) RadiusVal(idx int) float32 {
	if idx <= 0 {
		return 0.8
	}
	return vtc.Radii[idx-1]
	//return (&radius_2)[idx-1]; }
	// get radius value using *zero-based* index value
}

/////////////////////////////////////////////////////
//              VocalTract

// OroPharynxRegions are different regions of the vocal tract
type OroPharynxRegions int32

const (
	OroPharynxReg1 = iota // S1
	OroPharynxReg2        // S2
	OroPharynxReg3        // S3
	OroPharynxReg4        // S4 & S5
	OroPharynxReg5        // S6 & S7
	OroPharynxReg6        // S8
	OroPharynxReg7        // S9
	OroPharynxReg8        // S10
	OroPharynxRegCount
)

//go:generate stringer -type=OroPharynxRegions

// OroPharynxCoefs are the oropharynx scattering junction coefficients (between each region)
type OroPharynxCoefs int32

const (
	OroPharynxCoef1 = iota // R1-R2 (S1-S2)
	OroPharynxCoef2        // R2-R3 (S2-S3)
	OroPharynxCoef3        // R3-R4 (S3-S4)
	OroPharynxCoef4        // R4-R5 (S5-S6)
	OroPharynxCoef5        // R5-R6 (S7-S8)
	OroPharynxCoef6        // R6-R7 (S8-S9)
	OroPharynxCoef7        // R7-R8 (S9-S10)
	OroPharynxCoef8        // R8-Air (S10-Air)
	OroPharynxCoefCount
)

//go:generate stringer -type=OroPharynxCoefs

// OroPharynxCoefs are the oropharynx scattering junction coefficients (between each region)
type OroPharynxSects int32

const (
	OroPharynxSect1  = iota // OroPharynxReg1
	OroPharynxSect2         // OroPharynxReg2
	OroPharynxSect3         // OroPharynxReg3
	OroPharynxSect4         // OroPharynxReg4
	OroPharynxSect5         // OroPharynxReg4
	OroPharynxSect6         // OroPharynxReg5
	OroPharynxSect7         // OroPharynxReg5
	OroPharynxSect8         // OroPharynxReg6
	OroPharynxSect9         // OroPharynxReg7
	OroPharynxSect10        // OroPharynxReg8
	OroPharynxSectCount
)

//go:generate stringer -type=OroPharynxSects

// NasalSections are different sections of the nasal tract
type NasalSections int32

const (
	NasalSect1 = iota
	NasalSect2
	NasalSect3
	NasalSect4
	NasalSect5
	NasalSect6
	NasalSectCount
	Velum = NasalSect1
)

//go:generate stringer -type=NasalSections

// NasalCoefs
type NasalCoefs int32

const (
	NasalCoef1     = NasalSect1 // N1-N2
	NasalCoef2     = NasalSect2 // N2-N3
	NasalCoef3     = NasalSect3 // N3-N4
	NasalCoef4     = NasalSect4 // N4-N5
	NasalCoef5     = NasalSect5 // N5-N6
	NasalCoef6     = NasalSect6 // N6-Air
	NasalCoefCount = NasalSectCount
)

//go:generate stringer -type=NasalCoefs

// ThreeWayJunction for the three-way junction alpha coefficients
type ThreeWayJunction int32

const (
	ThreeWayLeft = iota
	ThreeWayRight
	ThreeWayUpper
	ThreeWayCount
)

//go:generate stringer -type=ThreeWayJunction

// FricationInjCoefs are the oropharynx scattering junction coefficients (between each region)
type FricationInjCoefs int32

const (
	FricationInjCoef1 = iota // S3
	FricationInjCoef2        // S4
	FricationInjCoef3        // S5
	FricationInjCoef4        // S6
	FricationInjCoef5        // S7
	FricationInjCoef6        // S8
	FricationInjCoef7        // S9
	FricationInjCoef8        // S10
	FricationInjCoefCount
)

//go:generate stringer -type=FricationInjCoefs

type VocalTract struct {
	Buf          sound.Wave
	Volume       float32
	Balance      float32
	Duration     float32 // duration of synthesized sound
	Params       TractParams
	Voice        VoiceParams
	CurControl   VocalTractCtrl
	PrevControl  VocalTractCtrl
	DeltaControl VocalTractCtrl
	DeltaMax     VocalTractCtrl
	PhoneTable   etable.Table
	DictTable    etable.Table

	// derived values
	ControlRate      float32 // 1.0-1000.0 input tables/second (Hz)
	ControlPeriod    int
	SampleRate       int
	ActualTubeLength float32 // actual length in cm

	CurrentData VocalTractCtrl // current control data

	// tube and tube coefficients
	Oropharynx      [OroPharynxSectCount][2][2]float32
	OropharynxCoefs [OroPharynxCoefCount]float32
	Nasal           [NasalSectCount][2][2]float32
	NasalCoefs      [NasalCoefCount]float32
	Alpha           [ThreeWayCount]float32
	CurPtr          int
	PrevPtr         int

	// memory for frication taps
	FricationTap [FricationInjCoefCount]float32

	DampingFactor     float32 // calculated
	CrossmixFactor    float32 //  calculated
	BreathinessFactor float32
	PrevGlotAmplitude float32

	OutputData []float32

	SampleRateConverter   RateConverter
	MouthRadiationFilter  RadiationFilter
	MouthReflectionFilter ReflectionFilter
	NasalRadiationFilter  RadiationFilter
	NasalReflectionFilter ReflectionFilter
	Throat                Throat
	GlottalSource         WavetableGlottalSource
	BandpassFilter        BandpassFilter
	NoiseFilter           NoiseFilter
	NoiseSource           NoiseSource
}

// Init gets us going - this is the first function to call
func (vt *VocalTract) Init() {
	vt.Volume = 60.0
	vt.Balance = 0.0
	vt.Duration = 25.0
	vt.ControlRate = 0.0
	vt.SampleRate = 44100
	vt.Duration = 25
	// todo	del_max.DefaultMaxDeltas();

	vt.CurrentData.Defaults()
	vt.CurControl.SetFromParams(&vt.CurrentData)
	vt.Voice.Defaults()
	vt.Params.Defaults()
	vt.InitBuffer()
	vt.Reset()
	ctrlRate := 1.0 / (vt.Duration / 1000.0)
	vt.ControlRate = ctrlRate
	vt.InitializeSynthesizer()
	vt.PrevControl.SetFromParams(&vt.CurControl) // no deltas if reset
	vt.CurrentData.SetFromParams(&vt.CurControl)

	vt.DeltaMax.DefaultMaxDeltas()
	// outputData_.reserve(OUTPUT_VECTOR_RESERVE);

}

func (vt *VocalTract) ControlFromTable(col etensor.Tensor, row int, normalized bool) {
	params := col.SubSpace([]int{row}).(*etensor.Float32)
	vt.CurControl.SetFromValues(params.Values)
}

//void VocalTract::SynthFromTable(const Table& table, const Variant& col, int row,
//                                   bool normalized, bool reset_first) {
// float_MatrixPtr mtx;
// mtx = (float_Matrix*)table.GetValAsMatrix(col, row);
// if(TestError(!(bool)mtx, "SynthFromTable", "matrix column not found")) {
//   return;
// }
// if(mtx->dims() == 2 && mtx->dim(0) == VocalTractCtrl::N_PARAMS) {
//   // multi-dim case..
//   int n_outer = mtx->dim(1);
//   for(int i=0; i< n_outer; i++) {
//     float_MatrixPtr frm;
//     frm = (float_Matrix*)mtx->GetFrameSlice(i);
//     CtrlFromMatrix(*frm, normalized);
//     Synthesize(reset_first && (i == 0));
//   }
// }
// else {
//   // one-shot
//   cur_ctrl.SetFromTable(table, col, row, normalized);
//   Synthesize(reset_first);
// }
//}
//

// LoadEnglishPhones loads the file of English phones
func (vt *VocalTract) LoadEnglishPhones() {
	fn := gi.FileName("VocalTractEnglishPhones.dat")
	err := vt.PhoneTable.OpenCSV(fn, '\t')
	if err != nil {
		fmt.Printf("File not found or error opengin file: %s (%s)", fn, err)
		return
	}
}

// LoadEnglishDict loads the English dictionary of words composed of phones and transitions
func (vt *VocalTract) LoadEnglishDict() {
	fn := gi.FileName("VocalTractEnglishDict.dtbl")
	err := vt.DictTable.OpenCSV(fn, '\t')
	if err != nil {
		fmt.Printf("File not found or error opengin file: %s (%s)", fn, err)
		return
	}
}

// SynthPhone
func (vt *VocalTract) SynthPhone(phon string, stress, doubleStress, syllable, reset bool) bool {
	if vt.PhoneTable.Rows == 0 {
		vt.LoadEnglishPhones()
	}
	if stress {
		phon = phon + "'"
	}
	pcol := vt.PhoneTable.ColByName("phone")
	idx := -1
	for i := 0; i < pcol.Len(); i++ {
		if pcol.StringVal1D(i) == phon {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}

	dcol := vt.PhoneTable.ColByName("duration")
	dur := dcol.FloatVal1D(idx)
	tcol := vt.PhoneTable.ColByName("transition")
	trans := tcol.FloatVal1D(idx)
	totalTime := (dur + trans) * 1.5

	nReps := math.Ceil(totalTime / float64(vt.Duration))
	nReps = math.Max(nReps, 1.0)

	vt.ControlFromTable(vt.PhoneTable.ColByName("phone_data"), idx, false)
	// todo: syllable, double_stress, qsss other params??
	// fmt.Println("saying:", phon, "dur:", String(tot_time), "n_reps:", String(n_reps),
	//              "start pos:", String(outputData_.size()));
	if reset {
		vt.SynthReset(true)
	}
	for i := 0; i < int(nReps); i++ {
		vt.Synthesize(false)
	}
	return true
}

// SynthPhones
func (vt *VocalTract) SynthPhones(phones string, resetFirst, play bool) bool {
	var phone string
	stress := false
	doubleStress := false
	syllable := false
	first := true

	for _, r := range phones {
		c := string(r)
		if c == "'" {
			stress = true
			continue
		}
		if c == "\"" {
			doubleStress = true
			continue
		}
		if c == "%" {
			vt.SynthPhone(phone, stress, doubleStress, syllable, resetFirst && first)
			phone = ""
			first = false
			break // done
		}
		if c == "." { // syllable
			syllable = true
			vt.SynthPhone(phone, stress, doubleStress, syllable, resetFirst && first)
			stress = false
			doubleStress = false
			syllable = false
			phone = ""
			first = false
			continue
		}
		if c == "_" { // reg separator
			vt.SynthPhone(phone, stress, doubleStress, syllable, resetFirst && first)
			stress = false
			doubleStress = false
			syllable = false
			phone = ""
			first = false
			continue
		}
		phone += c
	}
	if len(phone) > 0 {
		vt.SynthPhone(phone, stress, doubleStress, syllable, resetFirst && first)
	}

	if play {
		PlaySound()
	}
	return true
}

// SynthWord
func (vt *VocalTract) SynthWord(word string, resetFirst bool, play bool) bool {
	if vt.DictTable.Rows == 0 {
		vt.LoadEnglishDict()
	}
	col := vt.DictTable.ColByName("word")
	if col == nil {
		fmt.Printf("Column name 'word' not found")
		return false
	}

	var idx = -1
	for i := 0; i < col.Len(); i++ {
		if col.StringVal1D(i) == word {
			idx = i
		}
	}
	if idx == -1 {
		return false
	}
	col = vt.DictTable.ColByName("phones")
	if col == nil {
		fmt.Printf("Column name 'phones' not found")
		return false
	}
	phones := col.StringVal1D(idx)
	return vt.SynthPhones(phones, resetFirst, play)
}

// SynthWords
func (vt *VocalTract) SynthWords(ws string, resetFirst bool, play bool) bool {
	words := strings.Split(ws, " ")
	rval := true
	for i := 0; i < len(words); i++ {
		rval := vt.SynthWord(words[i], (resetFirst && (i == 0)), false)
		if !rval {
			break
		}
		if i < len(words)-1 {
			vt.SynthPhone("#", false, false, false, false)
		}
	}
	if play {
		PlaySound()
	}
	return rval
}

// Reset reset all vocal tract values
func (vt *VocalTract) Reset() {
	vt.ControlPeriod = 0
	vt.ActualTubeLength = 0.0
	for i := 0; i < OroPharynxSectCount; i++ {
		for j := 0; j < 2; j++ {
			for k := 0; k < 2; k++ {
				vt.Oropharynx[i][j][k] = 0.0
			}
		}
	}
	for i := 0; i < OroPharynxCoefCount; i++ {
		vt.OropharynxCoefs[i] = 0.0
	}

	for i := 0; i < NasalSectCount; i++ {
		for j := 0; j < 2; j++ {
			for k := 0; k < 2; k++ {
				vt.Nasal[i][j][k] = 0.0
			}
		}
	}
	for i := 0; i < NasalCoefCount; i++ {
		vt.NasalCoefs[i] = 0.0
	}
	for i := 0; i < ThreeWayCount; i++ {
		vt.Alpha[i] = 0.0
	}
	for i := 0; i < FricationInjCoefCount; i++ {
		vt.FricationTap[i] = 0.0
	}
	vt.CurPtr = 1
	vt.PrevPtr = 0
	vt.DampingFactor = 0.0
	vt.CrossmixFactor = 0.0
	vt.BreathinessFactor = 0.0
	vt.PrevGlotAmplitude = -1.0
	vt.OutputData = vt.OutputData[:0]

	vt.SampleRateConverter.Reset()
	vt.MouthRadiationFilter.Reset()
	vt.MouthReflectionFilter.Reset()
	vt.NasalRadiationFilter.Reset()
	vt.NasalReflectionFilter.Reset()
	vt.Throat.Reset()
	vt.GlottalSource.Reset()
	vt.BandpassFilter.Reset()
	vt.NoiseFilter.Reset()
	vt.NoiseSource.Reset()
}

// SpeedOfSound returns the speed of sound according to the value of the temperature (in Celsius degrees)
func SpeedOfSound(temp float32) float32 {
	return 331.4 + (0.6 * temp)
}

//InitializeSynthesizer initializes all variables so that the synthesis can be run
func (vt *VocalTract) InitializeSynthesizer() {
	var nyquist float32

	// calculate the sample rate, based on nominal tube length and speed of sound
	if vt.Voice.TractLength > 0.0 {
		c := SpeedOfSound(vt.Params.Temp)
		vt.ControlPeriod = int(math.Round(float64(c*OroPharynxSectCount*100.0) / float64(vt.Voice.TractLength*vt.ControlRate)))
		vt.SampleRate = int(vt.ControlRate * float32(vt.ControlPeriod))
		vt.ActualTubeLength = float32(c*OroPharynxSectCount*100.0) / float32(vt.SampleRate)
		nyquist = float32(vt.SampleRate) / 2.0
	} else {
		nyquist = 1.0
		fmt.Println("Illegal tube length")
	}

	vt.BreathinessFactor = vt.Voice.Breathiness / 100.0
	vt.CrossmixFactor = 1.0 / Amplitude(vt.Params.MixOff)
	vt.DampingFactor = (1.0 - (vt.Params.Loss / 100.0))

	// initialize the wave table
	//vt.Voice.SetAgeGender(Female)
	gs := WavetableGlottalSource{}
	vt.GlottalSource = gs
	vt.GlottalSource.Init(GlottalSourcePulse, float32(vt.SampleRate), vt.Voice.GlotPulseRise, vt.Voice.GlotPulseFallMin, vt.Voice.GlotPulseFallMax)
	vt.GlottalSource.Reset()

	mouthApertureCoef := (nyquist - vt.Params.MouthCoef) / nyquist
	vt.MouthRadiationFilter.Init(mouthApertureCoef)
	vt.MouthRadiationFilter.Reset()
	vt.MouthReflectionFilter.Init(mouthApertureCoef)
	vt.MouthReflectionFilter.Reset()

	nasalApertureCoef := (nyquist - vt.Params.NoseCoef) / nyquist
	vt.NasalRadiationFilter.Init(nasalApertureCoef)
	vt.NasalRadiationFilter.Reset()
	vt.NasalReflectionFilter.Init(nasalApertureCoef)
	vt.NasalReflectionFilter.Reset()

	vt.InitNasal()
	vt.Throat.Init(float32(vt.SampleRate), vt.Params.ThroatCutoff, Amplitude(vt.Params.ThroatVol))
	vt.Throat.Reset()

	vt.SampleRateConverter.Init(vt.SampleRate, OutputRate, &vt.OutputData)
	vt.SampleRateConverter.Reset()
	vt.OutputData = vt.OutputData[:0]

	vt.BandpassFilter.Reset()
	vt.NoiseFilter.Reset()
	vt.NoiseSource.Reset()
}

func (vt *VocalTract) InitSynth() {
	vt.InitBuffer()
	vt.Reset()
	vt.ControlRate = 1.0 / (vt.Duration / 1000.0)
	vt.InitializeSynthesizer()
	vt.PrevControl.SetFromParams(&vt.CurControl)
	vt.CurrentData.SetFromParams(&vt.CurControl)
}

// InitBuffer
func (vt *VocalTract) InitBuffer() {
	frames := (vt.Duration / 1000.0) * float32(vt.SampleRate)
	format := &audio.Format{
		NumChannels: 1,
		SampleRate:  vt.SampleRate,
	}
	vt.Buf.Buf = &audio.IntBuffer{Data: make([]int, int(frames)), Format: format, SourceBitDepth: 16}
}

// SynthReset
func (vt *VocalTract) SynthReset(initBuffer bool) {
	vt.InitSynth()
	if initBuffer {
		vt.InitBuffer()
	}
}

// Synthesize
func (vt *VocalTract) Synthesize(resetFirst bool) {
	ctrlRate := 1.0 / (vt.Duration / 1000.0)
	if ctrlRate != vt.ControlRate { // todo: if ctrlRate != vt.ControlRate || !IsValid()
		vt.InitSynth()
	} else if resetFirst {
		vt.SynthReset(true)
	}

	controlFreq := 1.0 / float32(vt.ControlPeriod)
	fmt.Printf("control period: %v, freq: %v", vt.ControlPeriod, controlFreq)

	vt.DeltaControl.ComputeDeltas(&vt.CurControl, &vt.PrevControl, &vt.DeltaMax, float32(controlFreq))

	for j := 0; j < vt.ControlPeriod; j++ {
		vt.SynthesizeImpl()
		vt.CurrentData.UpdateFromDeltas(&vt.DeltaControl)
	}
	vt.PrevControl.SetFromParams(&vt.CurrentData) // prev is where we actually got, not where we wanted to get..

	//ss := vt.Buf.SampleSize()
	//st := vt.Buf.SampleType()
	//
	// todo:
	// nFrames := len(vt.OutputData
	// if FrameCount() < nFrames {
	//	InitBuffer(nFrames, SampleRate(), ChannelCount(), sampleSize, sType)
	// }

	// todo:
	// scale := vt.CalculateMonoScale()

	// #if (QT_VERSION >= 0x050000)
	// void* buf = q_buf.data();
	// for(int i=0; i < n_frm; i++) {
	//   WriteFloatAtIdx(outputData_[i] * scale, buf, i, stype, samp_size);
	// }
	// #endif
	// SigEmitUpdated();
}

// SynthesizeImpl
func (vt *VocalTract) SynthesizeImpl() {
	// convert parameters here
	f0 := Frequency(vt.CurrentData.GlotPitch)
	ax := Amplitude(vt.CurrentData.GlotVol)
	ah1 := Amplitude(vt.CurrentData.AspVol)
	vt.TubeCoefficients()
	vt.SetFricationTaps()
	vt.BandpassFilter.Update(float32(vt.SampleRate), vt.CurrentData.FricBw, vt.CurrentData.FricCf)

	// do synthesis here
	// create low-pass filtered noise
	lpNoise := vt.NoiseFilter.Filter(vt.NoiseSource.GetSample())
	fmt.Printf("%f\n", lpNoise)

	// update the shape of the glottal pulse, if necessary
	if vt.Params.WaveForm == Pulse {
		if ax != vt.PrevGlotAmplitude {
			vt.GlottalSource.Update(ax)
		}
	}

	//  create glottal pulse (or sine tone)
	pulse := vt.GlottalSource.GetSample(f0)

	pulsedNoise := lpNoise * pulse

	// create noisy glottal pulse
	pulse = ax * ((pulse * (1.0 - vt.BreathinessFactor)) + (pulsedNoise * vt.BreathinessFactor))

	var signal float32
	// cross-mix pure noise with pulsed noise
	if vt.Params.NoiseMod {
		crossmix := ax * vt.CrossmixFactor
		if crossmix >= 1.0 {
			crossmix = 1.0
		}
		signal = (pulsedNoise * crossmix) + (lpNoise * (1.0 - crossmix))
	} else {
		signal = lpNoise
	}
	fmt.Printf("%f\n", signal)

	return

	// put signal through vocal tract
	signal = vt.Update(((pulse + (ah1 * signal)) * VtScale), vt.BandpassFilter.Filter(signal))

	// put pulse through throat
	signal += vt.Throat.Process(pulse * VtScale)
	//fmt.Printf("%f\n", signal);

	// output sample here
	vt.SampleRateConverter.DataFill(signal)

	vt.PrevGlotAmplitude = ax
}

// InitNasalCavity
func (vt *VocalTract) InitNasal() {
	var radA2, radB2 float32

	// calculate coefficients for internal fixed sections of nasal cavity
	for i, j := NasalSect2, NasalCoef2; i < NasalSect6; i, j = i+1, j+1 {
		radA2 = vt.Voice.NoseRadii[i]
		radA2 *= radA2
		radB2 = vt.Voice.NoseRadii[i+1]
		radB2 *= radB2
		vt.NasalCoefs[j] = (radA2 - radB2) / (radA2 + radB2)
	}

	// calculate the fixed coefficient for the nose aperture
	radA2 = vt.Voice.NoseRadii[NasalSect6] // zero based
	radA2 *= radA2
	radB2 = vt.Voice.ApertureRadius * vt.Voice.ApertureRadius
	vt.NasalCoefs[NasalCoef6] = (radA2 - radB2) / (radA2 + radB2)
}

// TubeCoefficients
func (vt *VocalTract) TubeCoefficients() {
	var radA2, radB2 float32
	// calculate coefficients for the oropharynx
	for i := 0; i < OroPharynxRegCount-1; i++ {
		radA2 = vt.CurrentData.RadiusVal(i)
		radA2 *= radA2
		radB2 = vt.CurrentData.RadiusVal(i + 1)
		radB2 *= radB2
		vt.OropharynxCoefs[i] = (radA2 - radB2) / (radA2 + radB2)
	}

	// calculate the coefficient for the mouth aperture
	radA2 = vt.CurrentData.RadiusVal(OroPharynxReg8)
	radA2 *= radA2
	radB2 = vt.Voice.ApertureRadius * vt.Voice.ApertureRadius
	vt.OropharynxCoefs[OroPharynxCoef8] = (radA2 - radB2) / (radA2 + radB2)

	// calculate alpha coefficients for 3-way junction
	// note:  since junction is in middle of region 4, r0_2 = r1_2
	r0_2 := vt.CurrentData.RadiusVal(OroPharynxReg4)
	r0_2 *= r0_2
	r1_2 := r0_2
	r2_2 := vt.CurrentData.Velum * vt.CurrentData.Velum
	sum := 2.0 / (r0_2 + r1_2 + r2_2)
	vt.Alpha[ThreeWayLeft] = sum * r0_2
	vt.Alpha[ThreeWayRight] = sum * r1_2
	vt.Alpha[ThreeWayUpper] = sum * r2_2

	// and 1st nasal passage coefficient
	radA2 = vt.CurrentData.Velum * vt.CurrentData.Velum
	radB2 = vt.Voice.NoseRadii[NasalSect2]
	radB2 *= radB2
	vt.NasalCoefs[NasalCoef1] = (radA2 - radB2) / (radA2 + radB2)
}

// SetFricationTaps Sets frication taps according to the current position and amplitude of frication
func (vt *VocalTract) SetFricationTaps() {
	fricationAmplitude := Amplitude(vt.CurrentData.FricVol)

	integerPart := int(vt.CurrentData.FricPos)
	complement := vt.CurrentData.FricPos - float32(integerPart)
	remainder := 1.0 - complement

	for i := FricationInjCoef1; i < FricationInjCoefCount; i++ {
		if i == int(integerPart) {
			vt.FricationTap[i] = remainder * fricationAmplitude
			if (i + 1) < FricationInjCoefCount {
				i += 1
				vt.FricationTap[i] = complement * fricationAmplitude
			}
		} else {
			vt.FricationTap[i] = 0.0
		}
	}
	//#if 0
	// /*  PRINT OUT  */
	// printf("fricationTaps:  ");
	// for (i = FC1; i < TOTAL_FRIC_COEFFICIENTS; i++)
	//   printf("%.6f  ", fricationTap[i]);
	// printf("\n");
	//#endif
	//}
}

// Update updates the pressure wave throughout the vocal tract, and returns
// the summed output of the oral and nasal cavities.  Also injects frication appropriately
func (vt *VocalTract) Update(input, frication float32) float32 {
	vt.CurPtr += 1
	if vt.CurPtr > 1 {
		vt.CurPtr = 0
	}

	vt.PrevPtr += 1
	if vt.PrevPtr > 1 {
		vt.PrevPtr = 0
	}
	// input to top of tube
	vt.Oropharynx[OroPharynxSect1][Top][vt.CurPtr] =
		(vt.Oropharynx[OroPharynxSect1][Bottom][vt.PrevPtr] * vt.DampingFactor) + input

	// calculate the scattering junctions for s1-s2
	delta := vt.OropharynxCoefs[OroPharynxCoef1] *
		(vt.Oropharynx[OroPharynxSect1][Top][vt.PrevPtr] - vt.Oropharynx[OroPharynxSect2][Bottom][vt.PrevPtr])
	vt.Oropharynx[OroPharynxSect2][Top][vt.CurPtr] =
		(vt.Oropharynx[OroPharynxSect1][Top][vt.PrevPtr] + delta) * vt.DampingFactor
	vt.Oropharynx[OroPharynxSect1][Bottom][vt.CurPtr] =
		(vt.Oropharynx[OroPharynxSect2][Bottom][vt.PrevPtr] + delta) * vt.DampingFactor

	// calculate the scattering junctions for s2-s3 and s3-s4
	for i, j, k := OroPharynxSect2, OroPharynxCoef2, FricationInjCoef1; i < OroPharynxSect4; i, j, k = i+1, j+1, k+1 {
		delta = vt.OropharynxCoefs[j] *
			(vt.Oropharynx[i][Top][vt.PrevPtr] - vt.Oropharynx[i+1][Bottom][vt.PrevPtr])
		vt.Oropharynx[i+1][Top][vt.CurPtr] =
			((vt.Oropharynx[i][Top][vt.PrevPtr] + delta) * vt.DampingFactor) +
				(vt.FricationTap[k] * frication)
		vt.Oropharynx[i][Bottom][vt.CurPtr] =
			(vt.Oropharynx[i+1][Bottom][vt.PrevPtr] + delta) * vt.DampingFactor
	}

	// update 3-way junction between the middle of R4 and nasal cavity
	junctionPressure := (vt.Alpha[ThreeWayLeft] * vt.Oropharynx[OroPharynxSect4][Top][vt.PrevPtr]) +
		(vt.Alpha[ThreeWayRight] * vt.Oropharynx[OroPharynxSect5][Bottom][vt.PrevPtr]) +
		(vt.Alpha[ThreeWayUpper] * vt.Nasal[Velum][Bottom][vt.PrevPtr])
	vt.Oropharynx[OroPharynxSect4][Bottom][vt.CurPtr] =
		(junctionPressure - vt.Oropharynx[OroPharynxSect4][Top][vt.PrevPtr]) * vt.DampingFactor
	vt.Oropharynx[OroPharynxSect5][Top][vt.CurPtr] =
		((junctionPressure - vt.Oropharynx[OroPharynxSect5][Bottom][vt.PrevPtr]) * vt.DampingFactor) + (vt.FricationTap[FricationInjCoef3] * frication)
	vt.Nasal[Velum][Top][vt.CurPtr] =
		(junctionPressure - vt.Nasal[Velum][Bottom][vt.PrevPtr]) * vt.DampingFactor

	// calculate junction between R4 and R5 (S5-S6)
	delta = vt.OropharynxCoefs[OroPharynxCoef4] *
		(vt.Oropharynx[OroPharynxSect5][Top][vt.PrevPtr] - vt.Oropharynx[OroPharynxSect6][Bottom][vt.PrevPtr])
	vt.Oropharynx[OroPharynxSect6][Top][vt.CurPtr] =
		((vt.Oropharynx[OroPharynxSect5][Top][vt.PrevPtr] + delta) * vt.DampingFactor) +
			(vt.FricationTap[FricationInjCoef4] * frication)
	vt.Oropharynx[OroPharynxSect5][Bottom][vt.CurPtr] =
		(vt.Oropharynx[OroPharynxSect6][Bottom][vt.PrevPtr] + delta) * vt.DampingFactor

	// Calculate junction inside R5 (S6-S7) (pure delay with damping)
	vt.Oropharynx[OroPharynxSect7][Top][vt.CurPtr] =
		(vt.Oropharynx[OroPharynxSect6][Top][vt.PrevPtr] * vt.DampingFactor) +
			(vt.FricationTap[FricationInjCoef5] * frication)
	vt.Oropharynx[OroPharynxSect6][Bottom][vt.CurPtr] =
		vt.Oropharynx[OroPharynxSect7][Bottom][vt.PrevPtr] * vt.DampingFactor

	// calculate last 3 internal junctions (S7-S8, S8-S9, S9-S10)
	for i, j, k := OroPharynxSect7, OroPharynxCoef5, FricationInjCoef6; i < OroPharynxSect10; i, j, k = i+1, j+1, k+1 {
		delta = vt.OropharynxCoefs[j] *
			(vt.Oropharynx[i][Top][vt.PrevPtr] - vt.Oropharynx[i+1][Bottom][vt.PrevPtr])
		vt.Oropharynx[i+1][Top][vt.CurPtr] =
			((vt.Oropharynx[i][Top][vt.PrevPtr] + delta) * vt.DampingFactor) +
				(vt.FricationTap[k] * frication)
		vt.Oropharynx[i][Bottom][vt.CurPtr] =
			(vt.Oropharynx[i+1][Bottom][vt.PrevPtr] + delta) * vt.DampingFactor
	}

	// reflected signal at mouth goes through a lowpass filter
	vt.Oropharynx[OroPharynxSect10][Bottom][vt.CurPtr] = vt.DampingFactor *
		vt.MouthReflectionFilter.Filter(vt.OropharynxCoefs[OroPharynxCoef8]*
			vt.Oropharynx[OroPharynxSect10][Top][vt.PrevPtr])

	// output from mouth goes through a highpass filter
	output := vt.MouthRadiationFilter.Filter((1.0 + vt.OropharynxCoefs[OroPharynxCoef8]) *
		vt.Oropharynx[OroPharynxSect10][Top][vt.PrevPtr])

	//  update nasal cavity
	for i, j := Velum, NasalCoef1; i < NasalCoef6; i, j = i+1, j+1 {
		delta = vt.NasalCoefs[j] *
			(vt.Nasal[i][Top][vt.PrevPtr] - vt.Nasal[i+1][Bottom][vt.PrevPtr])
		vt.Nasal[i+1][Top][vt.CurPtr] =
			(vt.Nasal[i][Top][vt.PrevPtr] + delta) * vt.DampingFactor
		vt.Nasal[i][Bottom][vt.CurPtr] =
			(vt.Nasal[i+1][Bottom][vt.PrevPtr] + delta) * vt.DampingFactor
	}

	// reflected signal at nose goes through a lowpass filter
	vt.Nasal[NasalSect6][Bottom][vt.CurPtr] = vt.DampingFactor *
		vt.NasalReflectionFilter.Filter(vt.NasalCoefs[NasalCoef6]*vt.Nasal[NasalCoef6][Top][vt.PrevPtr])

	// output from nose goes through a highpass filter
	output += vt.MouthRadiationFilter.Filter((1.0 + vt.NasalCoefs[NasalCoef6]) *
		vt.Nasal[NasalSect6][Top][vt.PrevPtr])

	// return summed output from mouth and nose
	return output
}

// MonoScale
func (vt *VocalTract) MonoScale() float32 {
	return (OutputScale / (vt.SampleRateConverter.MaxSampleVal()) * Amplitude(vt.Volume))
}

// StereoScale
func (vt *VocalTract) StereoScale(leftScale,
	rightScale *float32) {
	*leftScale = (-((vt.Balance / 2.0) - 0.5))
	*rightScale = (-((vt.Balance / 2.0) + 0.5))

	scale := leftScale
	if vt.Balance > 0.0 {
		scale = rightScale
	}
	newMax := (vt.SampleRateConverter.MaxSampleVal() * (*scale))
	*scale = (OutputScale / (newMax * Amplitude(vt.Volume)))
	*leftScale *= *scale
	*rightScale *= *scale
}

// Amplitude  converts dB value to amplitude value
func Amplitude(decibelLevel float32) float32 {
	decibelLevel -= VolMax

	if decibelLevel <= -VolMax {
		return 0
	}

	if decibelLevel >= 0.0 {
		return 1.0
	}

	return math32.Pow(10.0, decibelLevel/20.0)
}

// Frequency converts a given pitch (0 = middle C) to the corresponding frequency
func Frequency(pitch float32) float32 {
	return PitchBase * math32.Pow(2.0, (pitch+PitchOffset)/12.0)
}

func PlaySound() {

}
