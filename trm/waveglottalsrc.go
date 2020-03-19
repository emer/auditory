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
// This is a port to golang of the C++ Gnuspeech port by Marcelo Y. Matuda

package trm

import (
	"math"
)

// compile with oversampling or plain oscillator
const OversamplingOscillator = true

// glottal source oscillator table variables
const TableLength = 512.0
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
	TnLength        float64
	TnDelta         float64
	BasicIncrement  float64
	CurrentPosition float64
	Wavetable       [TableLength]float64
	FirFilter       FirFilter
}

// Init calculates the initial glottal pulse and stores it in the wavetable, for use in the oscillator.
func (wgs *WavetableGlottalSource) Init(wType WaveForm, sampleRate, tp, tnMin, tnMax float64) {
	wgs.TableDiv1 = int(math.Round(TableLength * (tp / 100.0)))
	wgs.TableDiv2 = int(math.Round(TableLength * ((tp + tnMax) / 100.0)))
	wgs.TnLength = float64(wgs.TableDiv2 - wgs.TableDiv1)
	wgs.TnDelta = math.Round(float64(TableLength * (tnMax - tnMin) / 100.0))
	wgs.BasicIncrement = TableLength / sampleRate
	wgs.CurrentPosition = 0

	if OversamplingOscillator {
		wgs.FirFilter.Init(FirBeta, FirGamma, FirCutoff)
	}

	// initialize the wavetable with either a glottal pulse or sine tone
	if wType == Pulse {
		// calculate rise portion of wave table
		for i := 0; i < wgs.TableDiv1; i++ {
			x := float64(i) / float64(wgs.TableDiv1)
			x2 := x * x
			x3 := x2 * x
			wgs.Wavetable[i] = (3.0 * x2) - (2.0 * x3)
		}

		// calculate fall portion of wave table
		j := 0
		for i := wgs.TableDiv1; i < wgs.TableDiv2; i++ {
			x := float64(j) / wgs.TnLength
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
			wgs.Wavetable[i] = math.Sin((float64(i) / float64(TableLength) * 2.0 * math.Pi))
		}
	}
}

// Reset resets the current position and the Fir Filter
func (wgs *WavetableGlottalSource) Reset() {
	wgs.CurrentPosition = 0
	wgs.FirFilter.Init(FirBeta, FirGamma, FirCutoff)
}

// Update rewrites the changeable part of the glottal pulse according to the amplitude
func (wgs *WavetableGlottalSource) Update(amplitude float64) {
	// calculate new closure point, based on amplitude
	newDiv2 := float64(wgs.TableDiv2) - math.Round(float64(amplitude)*float64(wgs.TnDelta))
	invNewTnLength := 1.0 / (newDiv2 - float64(wgs.TableDiv1))

	//  recalculate the falling portion of the glottal pulse
	x := float64(0.0)
	end := int(newDiv2)
	for i := wgs.TableDiv1; i < end; i++ {
		wgs.Wavetable[i] = float64(1.0) - float64(x*x)
		x += invNewTnLength
	}

	// fill in with closed portion of glottal pulse
	for i := int(newDiv2); i < wgs.TableDiv2; i++ {
		wgs.Wavetable[i] = 0.0
	}
}

// IncrementPosition increments the position in the wavetable according to the specified frequency
func (wgs *WavetableGlottalSource) IncrementPosition(frequency float64) {
	wgs.CurrentPosition = Mod0(wgs.CurrentPosition + (frequency * wgs.BasicIncrement))
}

// GetSample returns sample value from plain oscillator or 2x oversampling oscillator.
func (wgs *WavetableGlottalSource) GetSample(frequency float64) (output float64) {

	if OversamplingOscillator {
		for i := 0; i < 2; i++ {
			// first increment the table position, depending on frequency
			wgs.IncrementPosition(frequency / 2.0)

			// find surrounding integer table positions
			lowerPosition := int(wgs.CurrentPosition)
			upperPosition := int(Mod0(float64(lowerPosition + 1)))

			// calculate interpolated table value
			iv := wgs.Wavetable[lowerPosition] +
				((wgs.CurrentPosition - float64(lowerPosition)) *
					(wgs.Wavetable[upperPosition] - wgs.Wavetable[lowerPosition]))

			// put value through fir filter
			output = float64(wgs.FirFilter.Filter(float64(iv), i == 1))
		}
		// since we decimate, take only the second output value
		return output
	} else { // plain oscillator
		// first increment the table position, depending on frequency
		wgs.IncrementPosition(frequency)

		// Find surrounding integer table positions
		lowerPosition := int(wgs.CurrentPosition)
		upperPosition := int(Mod0(float64(lowerPosition + 1)))

		// return interpolated table value
		output = wgs.Wavetable[lowerPosition] +
			((wgs.CurrentPosition - float64(lowerPosition)) *
				(wgs.Wavetable[upperPosition] - wgs.Wavetable[lowerPosition]))

		return output
	}
}

// Mod0 returns the modulus of 'value', keeping it in the range 0 -> TableModulus
func Mod0(value float64) float64 {
	if value > TableModulus {
		value -= TableLength
	}
	return value
}
