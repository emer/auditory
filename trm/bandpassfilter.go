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

// BandPassFilter
type BandpassFilter struct {
	bpAlpha float64
	bpBeta  float64
	bpGamma float64
	xn1     float64
	xn2     float64
	yn1     float64
	yn2     float64
}

// Reset sets the filter values to zero
func (bf *BandpassFilter) Reset() {
	bf.xn1 = 0.0
	bf.xn2 = 0.0
	bf.yn1 = 0.0
	bf.yn2 = 0.0
}

// Update sets the filter values based on sample rate, bandwidth and center frequency
func (bf *BandpassFilter) Update(sampleRate, bandwidth, centerFreq float64) {
	tanValue := math.Tan((math.Pi * bandwidth) / sampleRate)
	cosValue := math.Cos((2.0 * math.Pi * centerFreq) / sampleRate)
	bf.bpBeta = (1.0 - tanValue) / (2.0 * (1.0 + tanValue))
	bf.bpGamma = (0.5 + bf.bpBeta) * cosValue
	bf.bpAlpha = (0.5 - bf.bpBeta) / 2.0
}

// BandpassFilter is a Frication bandpass filter, with variable center frequency and bandwidth
func (bf *BandpassFilter) Filter(input float64) float64 {
	output := 2.0 * ((bf.bpAlpha * (input - bf.xn2)) + (bf.bpGamma * bf.yn1) - (bf.bpBeta * bf.yn2))

	bf.xn2 = bf.xn1
	bf.xn1 = input
	bf.yn2 = bf.yn1
	bf.yn1 = output
	return output
}
