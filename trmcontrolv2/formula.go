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

package trmcontrolv2

import (
	"errors"
	"log"

	"github.com/goki/ki/kit"
)

// TransitionType
type FormulaCode int

const (
	//
	Transition1 = iota
	Transition2
	Transition3
	Transition4
	Qssa1
	Qssa2
	Qssa3
	Qssa4
	Qssb1
	Qssb2
	Qssb3
	Qssb4
	Tempo1
	Tempo2
	Tempo3
	Tempo4
	Rd
	Beat
	Mark1
	Mark2
	Mark3
	CodeN
)

//go:generate stringer -type=FormulaCode

var Kit_FormulaCode = kit.Enums.AddEnum(CodeN, NotBitFlag, nil)

type Formula struct {
	Codes map[FormulaCode]float64
}

func (fc *Formula) Init() {
	fc.Codes = make(map[FormulaCodes]float64)
}

func (fc *Formula) Clear() {
	for k, v := range fc {
		v = 0.0
	}
}

func (fc *Formula) Default(tt TransitionType) error {
	fc.Codes[Transition1] = 33.3333
	fc.Codes[Transition2] = 33.3333
	fc.Codes[Transition3] = 33.3333
	fc.Codes[Transition4] = 33.3333
	fc.Codes[Qssa1] = 33.3333
	fc.Codes[Qssa2] = 33.3333
	fc.Codes[Qssa3] = 33.3333
	fc.Codes[Qssa4] = 33.3333
	fc.Codes[Qssb1] = 33.3333
	fc.Codes[Qssb2] = 33.3333
	fc.Codes[Qssb3] = 33.3333
	fc.Codes[Qssb4] = 33.3333
	fc.Codes[Tempo1] = 1.0
	fc.Codes[Tempo2] = 1.0
	fc.Codes[Tempo3] = 1.0
	fc.Codes[Tempo4] = 1.0

	fc.Codes[Beat] = 33.0
	fc.Codes[Mark1] = 100.0
	switch tt {
	case TransDiPhone:
		fc.Codes[Rd] = 100.0
		fc.Codes[Mark2] = 0.0
		fc.Codes[Mark3] = 0.0

	case TransTriPhone:
		fc.Codes[Rd] = 200.0
		fc.Codes[Mark2] = 200.0
		fc.Codes[Mark3] = 0.0

	case TransTetraPhone:
		fc.Codes[Rd] = 300.0
		fc.Codes[Mark2] = 200.0
		fc.Codes[Mark3] = 300.0

	default:
		log.Printf("Formula.Default: Unknown TransactionType")
		return errors.New("Formula.Default: Unknown TransactionType")
	}
	return nil
}
