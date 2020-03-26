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

)
const modelConfigFileName = "/trm_controlmodel.config"
const trmConfigFileName = "/trm.config"
const voiceFilePrefix = "/voice_"

type Control struct {
	Model Model
	Events Events
	ModelConfig ModelConfig
	TrmConfig TrmConfig
}

func (ctrl *Control) Init(path string, model *Model) {
	ctrl.Model = model
	ctrl.Events.Init(path, model)
	ctrl.loadConfigs
}

//
func (ctrl *Control) LoadConfigs(path string) {
	// std::ostringstream trmControlModelConfigFilePath;
	// trmControlModelConfigFilePath << configDirPath << TRM_CONTROL_MODEL_CONFIG_FILE_NAME;
	// trmControlModelConfig_.load(trmControlModelConfigFilePath.str());
// 
	// std::ostringstream trmConfigFilePath;
	// trmConfigFilePath << configDirPath << TRM_CONFIG_FILE_NAME;
// 
	// std::ostringstream voiceFilePath;
	// voiceFilePath << configDirPath << VOICE_FILE_PREFIX << trmControlModelConfig_.voiceName << ".config";
// 
	// trmConfig_.load(trmConfigFilePath.str(), voiceFilePath.str());
}

func (ctrl *Control) InitUtterance(std::ostream& trmParamStream) {
	if (ctrl.TrmConfig.outputRate != 22050.0) && (ctrl.TrmConfig.outputRate != 44100.0) {
		ctrl.TrmConfig.outputRate = 44100.0
	}
	if ((ctrl.TrmConfig.vtlOffset + ctrl.TrmConfig.TractLength) < 15.9) {
		ctrl.TrmConfig.outputRate = 44100.0
	}

	ctrl.Events.PitchMean = ctrl.ModelConfig.pitchOffset + ctrl.TrmConfig.referenceGlottalPitch
	ctrl.Events.SetGlobalTempo(trmControlModelConfig_.tempo)
	setIntonation(trmControlModelConfig_.intonation)
	ctrl.Events.SetUpDriftGenerator(trmControlModelConfig_.driftDeviation, trmControlModelConfig_.controlRate, trmControlModelConfig_.driftLowpassCutoff)
	ctrl.Events.SetRadiusCoef(ctrl.TrmConfig.radiusCoef)

	trmParamStream <<
		ctrl.TrmConfig.outputRate              << '\n' <<
		trmControlModelConfig_.controlRate << '\n' <<
		ctrl.TrmConfig.volume                  << '\n' <<
		ctrl.TrmConfig.channels                << '\n' <<
		ctrl.TrmConfig.balance                 << '\n' <<
		ctrl.TrmConfig.waveform                << '\n' <<
		ctrl.TrmConfig.glottalPulseTp          << '\n' <<
		ctrl.TrmConfig.glottalPulseTnMin       << '\n' <<
		ctrl.TrmConfig.glottalPulseTnMax       << '\n' <<
		ctrl.TrmConfig.breathiness             << '\n' <<
		ctrl.TrmConfig.vtlOffset + ctrl.TrmConfig.TractLength << '\n' << // tube length
		ctrl.TrmConfig.temperature             << '\n' <<
		ctrl.TrmConfig.lossFactor              << '\n' <<
		ctrl.TrmConfig.apertureRadius          << '\n' <<
		ctrl.TrmConfig.mouthCoef               << '\n' <<
		ctrl.TrmConfig.noseCoef                << '\n' <<
		ctrl.TrmConfig.noseRadius[1]           << '\n' <<
		ctrl.TrmConfig.noseRadius[2]           << '\n' <<
		ctrl.TrmConfig.noseRadius[3]           << '\n' <<
		ctrl.TrmConfig.noseRadius[4]           << '\n' <<
		ctrl.TrmConfig.noseRadius[5]           << '\n' <<
		ctrl.TrmConfig.throatCutoff            << '\n' <<
		ctrl.TrmConfig.throatVol               << '\n' <<
		ctrl.TrmConfig.modulation              << '\n' <<
		ctrl.TrmConfig.mixOffset               << '\n'
}

// Chunks are separated by /c.
// There is always one /c at the begin and another at the end of the string.
func (ctrl *Control) calcChunks(text string)
{
	int tmp = 0, idx = 0
	while (text[idx] != '\0') {
		if ((text[idx] == '/') && (text[idx + 1] == 'c')) {
			tmp++
			idx += 2
		} else {
			idx++
		}
	}
	tmp--
	if tmp < 0 tmp = 0
	return tmp
}

// Returns the position of the next /c (the position of the /).
func (ctrl *Control) NextChunk(text string) int
{
	int idx = 0
	while (text[idx] != '\0') {
		if ((text[idx] == '/') && (text[idx + 1] == 'c')) {
			return idx
		} else {
			idx++
		}
	}
	return 0
}

func (ctrl *Control) ValidPosture(token char) int
{
	switch(token[0]) {
	case '0':
	case '1':
	case '2':
	case '3':
	case '4':
	case '5':
	case '6':
	case '7':
	case '8':
	case '9':
		return 1
	default:
		return (model_.postureList().find(token) != nullptr)
	}
}

func (ctrl *Control) SetIntonation(intonation int)
{
	if intonation & Configuration::INTONATION_MICRO {
		ctrl.Events.SetMicroIntonation(1)
	} else {
		ctrl.Events.SetMicroIntonation(0)
	}

	if intonation & Configuration::INTONATION_MACRO {
		ctrl.Events.SetMacroIntonation(1)
		ctrl.Events.SetSmoothIntonation(1) // Macro and not smooth is not working.
	} else {
		ctrl.Events.SetMacroIntonation(0)
		ctrl.Events.SetSmoothIntonation(0) // Macro and not smooth is not working.
	}

	// Macro and not smooth is not working.
//	if (intonation & Configuration::INTONATION_SMOOTH) {
//		ctrl.Events.SetSmoothIntonation(1);
//	} else {
//		ctrl.Events.SetSmoothIntonation(0);
//	}

	if intonation & Configuration::INTONATION_DRIFT {
		ctrl.Events.SetDrift(1)
	} else {
		ctrl.Events.SetDrift(0)
	}

	if intonation & Configuration::INTONATION_RANDOMIZE {
		ctrl.Events.SetTgUseRandom(true)
	} else {
		ctrl.Events.SetTgUseRandom(false)
	}
}
