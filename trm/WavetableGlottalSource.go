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

// 2019-02
// This is a port of the Gnuspeech port to C++ by Marcelo Y. Matuda

package trm

import "math"

// compile with oversampling or plain oscillator
const OversamplingOscillator = 1

// glottal source oscillator table variables
const TableLength = 512
const TableModulus = TableLength - 1

//  oversampling fir filter characteristics
const FirBeta = .2
const FirGamma = .1
const FirCutoff = .00000001

type WaveForm int32

const (
	Pulse = iota
	Sine
)

//go:generate stringer -type=WaveForm

type WavetableGlottalSource struct {
	TableDiv1       int
	TableDiv2       int
	TnLength        float32
	TnDelta         float32
	BasicIncrement  float32
	CurrentPosition float32
	Wavetable       [TableLength]float32
	FirFilter       *FirFilter
}

// Init calculates the initial glottal pulse and stores it in the wavetable, for use in the oscillator.
func (wgs *WavetableGlottalSource) Init(wType WaveForm, sampleRate, tp, tnMin, tnMax float32) {
	wgs.TableDiv1 = int(math.Round(float64(TableLength * (tp / 100.0))))
	wgs.TableDiv2 = int(math.Round(float64(TableLength * ((tp + tnMax) / 100.0))))
	wgs.TnLength = float32(wgs.TableDiv2 - wgs.TableDiv1)
	wgs.TnDelta = float32(math.Round(float64(TableLength * (tnMax - tnMin) / 100.0)))
	wgs.BasicIncrement = float32(TableLength) / sampleRate
	curPos := 0

	// initialize the wavetable with either a glottal pulse or sine tone
	if wType == Pulse {
		// calculate rise portion of wave table
		for i := 0; i < wgs.TableDiv1; i++ {
			x := float32(i) / float32(wgs.TableDiv1)
			x2 := x * x
			x3 := x2 * x
			wgs.Wavetable[i] = (3.0 * x2) - (2.0 * x3)
		}

		// calculate fall portion of wave table
		j := 0
		for i := wgs.TableDiv1; i < wgs.TableDiv2; i++ {
			x := float32(j) / wgs.TnLength
			wgs.Wavetable[i] = 1.0 - (x * x)
			j++
		}

		// set closed portion of wave table
		for i := wgs.TableDiv2; i < TableLength; i++ {
			wgs.Wavetable[i] = 0.0
		}
	} else {
		// sine wave
		for i := 0; i < TableLength; i++ {
			wgs.Wavetable[i] = float32(math.Sin((float64(i) / float64(TableLength) * 2.0 * math.Pi)))
		}
	}
}

// ToDo:
//#if OVERSAMPLING_OSCILLATOR
// firFilter_.reset(new FIRFilter(FIR_BETA, FIR_GAMMA, FIR_CUTOFF));
// #endif

// Reset resets the current position and the Fir Filter
func (wgs *WavetableGlottalSource) Reset() {
	wgs.CurrentPosition = 0
	wgs.FirFilter.Reset()
}

// Update rewrites the changeable part of the glottal pulse according to the amplitude
func (wgs *WavetableGlottalSource) Update(amplitude float32) {
	// calculate new closure point, based on amplitude
	newDiv2 := float64(wgs.TableDiv2) - math.Round(float64(amplitude)*float64(wgs.TnDelta))
	invNewTnLength := 1.0 / (newDiv2 - float64(wgs.TableDiv1))

	//  recalculate the falling portion of the glottal pulse
	x := 0.0
	end := int(newDiv2)
	for i := wgs.TableDiv1; i < end; i++ {
		wgs.Wavetable[i] = float32(1.0) - float32(x*x)
		x += invNewTnLength
	}

	// fill in with closed portion of glottal pulse
	for i := int(newDiv2); i < wgs.TableDiv2; i++ {
		wgs.Wavetable[i] = 0.0
	}

}

// IncrementTablePosition increments the position in the wavetable according to the desired frequency
func (wgs *WavetableGlottalSource) IncrementTablePos(frequency float32) {
	wgs.CurrentPosition = Mod0(wgs.CurrentPosition + (frequency * wgs.BasicIncrement))
}

// ToDo:
/******************************************************************************
 *
 *  function:  oscillator
 *
 *  purpose:   Is a 2X oversampling interpolating wavetable
 *             oscillator.
 *
 ******************************************************************************/
//#if OVERSAMPLING_OSCILLATOR
//float
//WavetableGlottalSource::getSample(float frequency)  /*  2X OVERSAMPLING OSCILLATOR  */
//{
//  int lowerPosition, upperPosition;
//  float interpolatedValue, output;
//
//  for (int i = 0; i < 2; i++) {
//    /*  FIRST INCREMENT THE TABLE POSITION, DEPENDING ON FREQUENCY  */
//    incrementTablePosition(frequency / 2.0);
//
//    /*  FIND SURROUNDING INTEGER TABLE POSITIONS  */
//    lowerPosition = static_cast<int>(currentPosition_);
//    upperPosition = static_cast<int>(mod0(lowerPosition + 1));
//
//    /*  CALCULATE INTERPOLATED TABLE VALUE  */
//    interpolatedValue = wavetable_[lowerPosition] +
//      ((currentPosition_ - lowerPosition) *
//       (wavetable_[upperPosition] - wavetable_[lowerPosition]));
//
//    /*  PUT VALUE THROUGH FIR FILTER  */
//    output = firFilter_->filter(interpolatedValue, i);
//  }
//
//  /*  SINCE WE DECIMATE, TAKE ONLY THE SECOND OUTPUT VALUE  */
//  return output;
//}
//#else
// #endif

// Get sample from plain oscillator
func (wgs *WavetableGlottalSource) GetSample(frequency float32) float32 {
	// first increment the table position, depending on frequency
	wgs.IncrementTablePos(frequency)

	// Find surrounding integer table positions
	lowerPosition := int(wgs.CurrentPosition)
	upperPosition := int(Mod0(float32(lowerPosition + 1)))

	// return interpolated table value
	value := wgs.Wavetable[lowerPosition] +
		((wgs.CurrentPosition - float32(lowerPosition)) * (wgs.Wavetable[upperPosition] - wgs.Wavetable[lowerPosition]))

	return value
}

// Mod0 eturns the modulus of 'value', keeping it in the range 0 -> TableModulus
func Mod0(value float32) float32 {
	if value > TableModulus {
		value -= TableLength
	}
	return value
}
