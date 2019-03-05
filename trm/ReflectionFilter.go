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
// This is a port to golang of the C++ Gnuspeech port by Marcelo Y. Matuda

package trm

import (
	"github.com/chewxy/math32"
)

type ReflectionFilter struct {
	A10         float32
	B11         float32
	ReflectionY float32
}

// Init initializes all of the filters struct fields
func (rf *ReflectionFilter) Init(apertureCoef float32) {
	rf.ReflectionY = 0.0
	rf.B11 = -apertureCoef
	rf.A10 = 1.0 - math32.Abs(rf.B11)
}

// Reset set ReflectionY to 0
func (rf *ReflectionFilter) Reset() {
	rf.ReflectionY = 0.0
}

// Filter calculates the output based on input on current values
func (rf *ReflectionFilter) Filter(input float32) float32 {
	output := rf.A10*float32(input) - rf.B11*rf.ReflectionY
	rf.ReflectionY = output
	return output
}
