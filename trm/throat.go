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

// Throat
type Throat struct {
	Gain float64
	tb1  float64
	ta0  float64
	Y    float64
}

// Init initializes the throat lowpass filter coefficients according to the throatCutoff value,
// and also the throatGain, according to the throatVol value.
func (thr *Throat) Init(sampleRate, cutoff, gain float64) {
	thr.Gain = gain
	thr.ta0 = (cutoff * 2.0) / sampleRate
	thr.tb1 = 1.0 - thr.ta0
}

// Reset sets Y to 0
func (thr *Throat) Reset() {
	thr.Y = 0.0
}

// Process simulates the radiation of sound through the walls of the throat.
// Note that this form of the filter uses addition instead of subtraction for the econd term,
// since tb1 has reversed sign.
func (thr *Throat) Process(input float64) float64 {
	output := (thr.ta0 * input) + (thr.tb1 * thr.Y)
	thr.Y = output
	return output * thr.Gain
}
