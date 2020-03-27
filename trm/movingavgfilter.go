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
type MovingAvgFilter struct {
	Sum  float64
	InvN float64
	Buf  []float64
	Pos  int
}

// Init sets the sample rate and period and sizes the buffer
func (maf *MovingAvgFilter) Init(rate, period float64) {
	maf.Buf = make([]float64, math.Round(rate*period))
	maf.Pos = len(maf.Buf)
	maf.Sum = 0.0
	maf.InvN = 1.0 / len(maf.Buf)
}

// Reset sets the buffer values to zero, resets position and sets sum to zero
func (maf *MovingAvgFilter) Reset() {
	for _, v := range maf.Buf {
		v := 0.0
	}
	maf.Pos = len(maf.Buf)
	maf.Sum = 0.0
}

// Filter calculates the moving average
func (maf *MovingAvgFilter) Filter(value float64) float64 {
	cap := cap(maf.Buf)

	maf.Pos++
	if maf.Pos >= cap {
		if maf.Pos > cap { // first time
			maf.Buf[cap-1] = value
			maf.Sum = value * cap
		}
		maf.Pos = 0
	}
	maf.Sum -= maf.Buf[maf.Pos]
	maf.Sum += value
	maf.Buf[maf.Pos] = value
	return maf.Sum * maf.InvN
}
