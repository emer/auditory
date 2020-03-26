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

type IntonationPt struct {
	SemiTone float64 `desc:"value of the intonation in semitones"`
	Offset   float64 `desc:"points are timed wrt a beat + this offset"`
	Slope    float64 `desc:"Slope of point"`
	RuleIdx  int     `desc:"Index of posture which is the focus of this point"`
	Events   []Event `desc:"Current events"`
}

func (ip *Intonation) Defaults() {
	ip.SemiTone = 0.0
	ip.Offset = 0.0
	ip.Slope = 0.0
	ip.RuleIdx = 0.0
}

func (ip *IntonationPt) Init(events []Event) {
	for _, e := range events {
		ip.Events = append(ip.Events, e)
	}
}

func (ip *IntonationPt) AbsTime() float64 {
	time := ip.Events.GetBeat(ip.RuleIdx)
	return time + ip.Offset
}

func (ip *IntonationPt) BeatTime() float64 {
	return ip.Events.GetBeat(ip.RuleIdx)
}
