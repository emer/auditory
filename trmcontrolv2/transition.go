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
// This is a port to golang of the C++ Gnuspeech port by Marcelo Y. Matuda{

package trmcontrolv2

import "github.com/goki/ki/kit"

type PointOrSlope struct {	
	
}

func (pos *PointOrSlope) IsSlopeRatio() {
	
}

// TransitionType 
type TransitionType int

const (
	// TransInvalid restricts the list of symbols to the active file
	TransInvalid TransitionType = iota

	// TransDiPhone
	TransDiPhone = 2

	// TransTriPhone
	TransTriPhone
	
	// TransTetraPhone
	TransTetraPhone
	
	TransTypeN
)

//go:generate stringer -type=TransitionType

var Kit_TransitionType = kit.Enums.AddEnum(TransTypeN, kit.NotBitFlag, nil)

// Transition
type Transition struct {
	
}

//
func(trn *Transition) SlopeRatio::totalSlopeUnits() float64
{
	temp := 0.0
	for (const auto& slope : slopeList) {
		temp += slope->slope
	}
	return temp
}

//
func(trn *Transition) GetPointTime(point, model *Model) float64
{
	if (!point.timeExpression) {
		return point.freeTime
	} else {
		return model.evalEquationFormula(*point.timeExpression)
	}
}

//
func(trn *Transition) getPointData(const Transition::Point& point, const Model& model,
				double& time, double& value)
{
	if (!point.timeExpression) {
		time = point.freeTime
	} else {
		time = model.evalEquationFormula(*point.timeExpression)
	}

	value = point.value
}

//
func(trn *Transition) getPointData(const Transition::Point& point, const Model& model,
				double baseline, double delta, double min, double max,
				double& time, double& value)
{
	if (!point.timeExpression) {
		time = point.freeTime
	} else {
		time = model.evalEquationFormula(*point.timeExpression)
	}

	value = baseline + ((point.value / 100.0) * delta)
	if (value < min) {
		value = min
	} else if (value > max) {
		value = max
	}
}
