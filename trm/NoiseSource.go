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

const Factor = 377.0
const InitialSeed = 0.7892347

type NoiseSource struct {
	seed float32
}

func (ns *NoiseSource) Init() {
	ns.Reset()
}

func (ns *NoiseSource) Reset() {
	ns.seed = InitialSeed
}

func (ns *NoiseSource) GetSample() float32 {
	product := ns.seed * Factor
	// C++ code was "seed_ = product - static_cast<int>(product);"
	var product64 float64
	product64 = float64(product)
	ns.seed = float32(product64 - math.Trunc(product64+.5))
	return ns.seed - 0.5
}
