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

// FormulaSymbolType
type FormulaSymbolType int

const (
	//
	FormulaSymTransition1 = iota
	FormulaSymTransition2
	FormulaSymTransition3
	FormulaSymTransition4
	FormulaSymQssa1
	FormulaSymQssa2
	FormulaSymQssa3
	FormulaSymQssa4
	FormulaSymQssb1
	FormulaSymQssb2
	FormulaSymQssb3
	FormulaSymQssb4
	FormulaSymTempo1
	FormulaSymTempo2
	FormulaSymTempo3
	FormulaSymTempo4
	FormulaSymRd
	FormulaSymBeat
	FormulaSymMark1
	FormulaSymMark2
	FormulaSymMark3
	FormulaSymTypeN
)

//go:generate stringer -type=FormulaSymbolType

var Kit_FormulaSymbolType = kit.Enums.AddEnum(FormulaSymTypeN, kit.NotBitFlag, nil)

type Formula struct {
	Syms map[FormulaSymbolType]float64
}

func (fc *Formula) Init() {
	fc.Syms = make(map[FormulaSymbolType]float64)
}

func (fc *Formula) Clear() {
	for k, v := range fc {
		v = 0.0
	}
}

func (fc *Formula) Default(tt TransitionType) error {
	fc.Syms[FormulaSymTransition1] = 33.3333
	fc.Syms[FormulaSymTransition2] = 33.3333
	fc.Syms[FormulaSymTransition3] = 33.3333
	fc.Syms[FormulaSymTransition4] = 33.3333
	fc.Syms[FormulaSymQssa1] = 33.3333
	fc.Syms[FormulaSymQssa2] = 33.3333
	fc.Syms[FormulaSymQssa3] = 33.3333
	fc.Syms[FormulaSymQssa4] = 33.3333
	fc.Syms[FormulaSymQssb1] = 33.3333
	fc.Syms[FormulaSymQssb2] = 33.3333
	fc.Syms[FormulaSymQssb3] = 33.3333
	fc.Syms[FormulaSymQssb4] = 33.3333
	fc.Syms[FormulaSymTempo1] = 1.0
	fc.Syms[FormulaSymTempo2] = 1.0
	fc.Syms[FormulaSymTempo3] = 1.0
	fc.Syms[FormulaSymTempo4] = 1.0

	fc.Syms[FormulaSymBeat] = 33.0
	fc.Syms[FormulaSymMark1] = 100.0
	switch tt {
	case TransDiPhone:
		fc.Syms[FormulaSymRd] = 100.0
		fc.Syms[FormulaSymMark2] = 0.0
		fc.Syms[FormulaSymMark3] = 0.0

	case TransTriPhone:
		fc.Syms[FormulaSymRd] = 200.0
		fc.Syms[FormulaSymMark2] = 200.0
		fc.Syms[FormulaSymMark3] = 0.0

	case TransTetraPhone:
		fc.Syms[FormulaSymRd] = 300.0
		fc.Syms[FormulaSymMark2] = 200.0
		fc.Syms[FormulaSymMark3] = 300.0

	default:
		log.Printf("Formula.Default: Unknown TransactionType")
		return errors.New("Formula.Default: Unknown TransactionType")
	}
	return nil
}
