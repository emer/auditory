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

const InitialSeed = 0.7892347
const Factor = 377.0

type Drift struct {
	Deviation float64 `desc:"the amount of drift in semitones above and below the median.  A value around 1 or so should give good results."`
	Offset    float64 `desc:"pitch offset"`
	Seed      float64 `desc:""`
	A0        float64 `desc:""`
	B1        float64 `desc:""`
	PrvSample float64 `desc:""`
}

func (dg *Drift) Defaults() {
	Deviation = 0.0
	Offset = 0.0
	Seed = InitialSeed
	A0 = 0.0
	B1 = 0.0
	PrvSample = 0.0
}

// Setup sets the params - deviation value around 1 should be good,
// sr the sample rate of the system in Hz---this should be the same as the control rate (250 Hz),
// lpcutoff - the cutoff in Hz of the lowpass filter applied to the noise generator.  This value must
// range from 0 Hz to nyquist.  A low value around 1 - 4 Hz should give good results.
func (dg *Drift) SetUp(deviation, sr, lpcutoff float64) {
	// set pitch deviation and offset variables
	dg.Deviation = deviation * 2.0
	dg.Offset = deviation

	// check range of the lowpass cutoff argument
	if lpcutoff < 0.0 {
		lpcutoff = 0.0
	} else if lpcutoff > (sr / 2.0) {
		lpcutoff = sr / 2.0
	}

	// set the filter coefficients
	dg.A0 = (lowpassCutoff * 2.0) / sr
	dg.B1 = 1.0 - dg.A0

	// clear the previous sample memory
	dg.PrvSample = 0.0
}

// Drift returns one sample of the drift signal
func (dg *Drift) Drift() float64 {
	// create random number between 0 and 1
	temp := dg.Seed * factor
	dg.Seed = temp - int(temp) // save for next invocation

	// create random signal with range -deviation to +deviation
	temp = (dg.Seed * dg.Deviation) - dg.Offset

	// lowpass filter the random signal (output is saved for next time)
	dg.PrvSample = (dg.A0 * temp) + (dg.B1 * dg.prvSample)
	return dg.PrvSample
}
