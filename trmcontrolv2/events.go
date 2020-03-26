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
	"math"
)
const diphone = 2
const triphone = 3
const tetraphone = 4

const intonationConfigFileName = "/intonation"
const eps = 1.0e-6

const toneStatement =    0
const toneExclamation =  1
const toneQuestion =    2
const toneContinuation = 3
const toneSemicolon =    4

cons invalidEvent = math.Inf(1) // positive infinity

type Event struct {
	Events []float64
	Size int
	Time int
	Flat int
}

func (ev *Event) Defaults() {
	size = 36
	time = 0.0
	flag = 0
	ev.Events = make([]float64, ev.Size)
	for i:=0; i < len(ev.Events); i++ {
		ev.Events[i] = invalidEvent 
	}
}

func (ev *Event) Value(idx int) {
	return ev.Events[idx]
}

func (ev *Event) SetValue(v float64, idx int) { 
	if idx < 0 {
		return
	}
	ev.Events[idx] = v
}

type Posture struct {	
	Syllable int
	Onset float64
	RuleTempo float64 // (was 32))
}

func (pos *Posture) Defaults() {
	pos.Syllable = 0
	pos.Onset = 0.0
	pos.RuleTempo = 0.0
}

type Foot struct {
	Onset1 float64
	Onset2 float64
	Tempo float64
	Start int
	End int
	Marked int
	Last int
}

func (ft *Foot) Defaults() {
	ft.Onset1 = 0.0
	ft.Onset2 = 0.0
	ft.Tempo = 1.0
	ft.Start = 0
	ft.End = 0
	ft.Marked = 0
	ft.Last = 0
}

type Tone struct {
	StartFoot int
	EndFoot int
	Type int
}

func (tn *Tone) Defaults() {
	StartFoot = 0
	EndFoot= 0
	Type = 0
}

type Rule struct {
	Number int
	FirstPosture int
	LastPosture int
	Duration float64
	Beat float64
}

func (rl *Rule) Defaults() {
	rl.Number = 0
	rl.FirstPosture = 0
	rl.LastPosture = 0
	rl.Duration = 0.0
	rl.Beat = 0.0
}

type Events struct {
	Model *Model

	ZeroRef int
	ZeroIdx int
	Duration int
	TimeQuant int
	MacroFlag int
	MicroFlag int
	DriftFlag int
	SmoothInton int
	PitchMean float64
	GlobalTempo float64
	Multiplier float64
	IntonParams []float64
	
	UseFixedIntonParams bool
	FixedIntonParams []float64
	
}
-----------------------------------------

func (evs *Events) Defaults() {
	evs.MacroFlag = 0
	evs.MicroFlag = 0
	evs.DriftFlag = 0
	evs.SmoothInton = 1
	evs.GlobalTempo = 1.0
	// evs.TgParams??
	evs.UseFixedIntonParams = false
}

func (evs *Events) Init(confdir string, model *Model) {
	evs.Model = model
	
	setUp();

	list_.reserve(128);

	initToneGroups(configDirPath);

	for (int i = 0; i < 10; ++i) {
		fixedIntonationParameters_[i] = 0.0;
	}
	for (int i = 0; i < TRM::Tube::TOTAL_REGIONS; ++i) {
		radiusCoef[i] = 1.0;
	}


	list_.reserve(128);

	initToneGroups(configDirPath);

	for (int i = 0; i < 10; ++i) {
		fixedIntonationParameters_[i] = 0.0;
	}
	for (int i = 0; i < TRM::Tube::TOTAL_REGIONS; ++i) {
		radiusCoef[i] = 1.0;
	}
	
}

// ToDo: do we need this?
func (evs *Events) setUpDriftGenerator(double deviation, double sampleRate, double lowpassCutoff)
{
	driftGenerator_.setUp(deviation, sampleRate, lowpassCutoff);
}

func (evs *Events) GetPostureAtIndex(idx int) *Posture {
	if index <= evs.CurPosture {
		return evs.Postures[idx]
	}
	return nil
}

const PostureData*
EventList::getPostureDataAtIndex(unsigned int index) const
{
	if (index > currentPosture_) {
		return nullptr;
	} else {
		return &postureData_[index];
	}
}

const RuleData*
EventList::getRuleAtIndex(unsigned int index) const
{
	if (static_cast<int>(index) > currentRule_) {
		return nullptr;
	} else {
		return &ruleData_[index];
	}
}

void
EventList::setFixedIntonationParameters(float notionalPitch, float pretonicRange, float pretonicLift, float tonicRange, float tonicMovement)
{
	fixedIntonationParameters_[1] = notionalPitch;
	fixedIntonationParameters_[2] = pretonicRange;
	fixedIntonationParameters_[3] = pretonicLift;
	fixedIntonationParameters_[5] = tonicRange;
	fixedIntonationParameters_[6] = tonicMovement;
}

void
EventList::setRadiusCoef(const double* values)
{
	for (int i = 0; i < TRM::Tube::TOTAL_REGIONS; ++i) {
		radiusCoef[i] = values[i];
	}
}

void
EventList::parseGroups(int index, int number, FILE* fp)
{
	char line[256];
	tgParameters_[index].resize(10 * number);
	for (int i = 0; i < number; ++i) {
		fgets(line, 256, fp);
		float* temp = &tgParameters_[index][i * 10];
		sscanf(line, " %f %f %f %f %f %f %f %f %f %f",
			&temp[0], &temp[1], &temp[2], &temp[3], &temp[4],
			&temp[5], &temp[6], &temp[7], &temp[8], &temp[9]);
	}
}

void
EventList::initToneGroups(const char* configDirPath)
{
	FILE* fp;
	char line[256];
	int count = 0;

	std::ostringstream path;
	path << configDirPath << INTONATION_CONFIG_FILE_NAME;
	fp = fopen(path.str().c_str(), "rb");
	if (fp == NULL) {
		THROW_EXCEPTION(IOException, "Could not open the file " << path.str().c_str() << '.');
	}
	while (fgets(line, 256, fp) != NULL) {
		if ((line[0] == '#') || (line[0] == ' ')) {
			// Skip.
		} else if (strncmp(line, "TG", 2) == 0) {
			sscanf(&line[2], " %d", &tgCount_[count]);
			parseGroups(count, tgCount_[count], fp);
			count++;
		} else if (strncmp(line, "RANDOM", 6) == 0) {
			sscanf(&line[6], " %f", &intonationRandom_);
		}
	}
	fclose(fp);

	if (Log::debugEnabled) {
		printToneGroups();
	}
}

void
EventList::printToneGroups()
{
	printf("===== Intonation configuration:\n");
	printf("Intonation random = %f\n", intonationRandom_);
	printf("Tone groups: %d %d %d %d %d\n", tgCount_[0], tgCount_[1], tgCount_[2], tgCount_[3], tgCount_[4]);

	for (int i = 0; i < 5; i++) {
		float* temp = &tgParameters_[i][0];
		printf("Temp [%d] = %p\n", i, temp);
		int j = 0;
		for (int k = 0; k < tgCount_[i]; k++) {
			printf("%f %f %f %f %f %f %f %f %f %f\n",
				temp[j]  , temp[j+1], temp[j+2], temp[j+3], temp[j+4],
				temp[j+5], temp[j+6], temp[j+7], temp[j+8], temp[j+9]);
			j += 10;
		}
	}
}

double
EventList::getBeatAtIndex(int ruleIndex) const
{
	if (ruleIndex > currentRule_) {
		return 0.0;
	} else {
		return ruleData_[ruleIndex].beat;
	}
}

void
EventList::newPostureWithObject(const Posture& p)
{
	if (postureData_[currentPosture_].posture) {
		postureData_.push_back(PostureData());
		postureTempo_.push_back(1.0);
		currentPosture_++;
	}
	postureTempo_[currentPosture_] = 1.0;
	postureData_[currentPosture_].ruleTempo = 1.0;
	postureData_[currentPosture_].posture = &p;
}

void
EventList::replaceCurrentPostureWith(const Posture& p)
{
	if (postureData_[currentPosture_].posture) {
		postureData_[currentPosture_].posture = &p;
	} else {
		postureData_[currentPosture_ - 1].posture = &p;
	}
}

void
EventList::setCurrentToneGroupType(int type)
{
	toneGroups_[currentToneGroup_].type = type;
}

void
EventList::newFoot()
{
	if (currentPosture_ == 0) {
		return;
	}

	feet_[currentFoot_++].end = currentPosture_;
	newPosture();

	feet_.push_back(Foot());
	feet_[currentFoot_].start = currentPosture_;
	feet_[currentFoot_].end = -1;
	feet_[currentFoot_].tempo = 1.0;
}

void
EventList::setCurrentFootMarked()
{
	feet_[currentFoot_].marked = 1;
}

void
EventList::setCurrentFootLast()
{
	feet_[currentFoot_].last = 1;
}

void
EventList::setCurrentFootTempo(double tempo)
{
	feet_[currentFoot_].tempo = tempo;
}

void
EventList::setCurrentPostureTempo(double tempo)
{
	postureTempo_[currentPosture_] = tempo;
}

void
EventList::setCurrentPostureRuleTempo(float tempo)
{
	postureData_[currentPosture_].ruleTempo = tempo;
}

void
EventList::newToneGroup()
{
	if (currentFoot_ == 0) {
		return;
	}

	toneGroups_[currentToneGroup_++].endFoot = currentFoot_;
	newFoot();

	toneGroups_.push_back(ToneGroup());
	toneGroups_[currentToneGroup_].startFoot = currentFoot_;
	toneGroups_[currentToneGroup_].endFoot = -1;
}

void
EventList::newPosture()
{
	if (postureData_[currentPosture_].posture) {
		postureData_.push_back(PostureData());
		postureTempo_.push_back(1.0);
		currentPosture_++;
	}
	postureTempo_[currentPosture_] = 1.0;
}

void
EventList::setCurrentPostureSyllable()
{
	postureData_[currentPosture_].syllable = 1;
}

Event*
EventList::insertEvent(int number, double time, double value)
{
	time = time * evs.Multiplier;
	if (time < 0.0) {
		return nullptr;
	}
	if (time > (double) (evs.Duration + evs.TimeQuant)) {
		return nullptr;
	}

	int tempTime = ZeroRef + (int) time;
	tempTime = (tempTime >> 2) << 2;
	//if ((tempTime % timeQuantization) != 0) {
	//	tempTime++;
	//}

	if (list_.empty()) {
		std::unique_ptr<Event> tempEvent(new Event());
		tempEvent->time = tempTime;
		if (number >= 0) {
			tempEvent->setValue(value, number);
		}
		list_.push_back(std::move(tempEvent));
		return list_.back().get();
	}

	int i;
	for (i = list_.size() - 1; i >= zeroIndex_; i--) {
		if (list_[i]->time == tempTime) {
			if (number >= 0) {
				list_[i]->setValue(value, number);
			}
			return list_[i].get();
		}
		if (list_[i]->time < tempTime) {
			std::unique_ptr<Event> tempEvent(new Event());
			tempEvent->time = tempTime;
			if (number >= 0) {
				tempEvent->setValue(value, number);
			}
			list_.insert(list_.begin() + (i + 1), std::move(tempEvent));
			return list_[i + 1].get();
		}
	}

	std::unique_ptr<Event> tempEvent(new Event());
	tempEvent->time = tempTime;
	if (number >= 0) {
		tempEvent->setValue(value, number);
	}
	list_.insert(list_.begin() + (i + 1), std::move(tempEvent));
	return list_[i + 1].get();
}

void
EventList::setZeroRef(int newValue)
{
	evs.ZeroRef = newValue;
	zeroIndex_ = 0;

	if (list_.empty()) {
		return;
	}

	for (int i = list_.size() - 1; i >= 0; i--) {
		if (list_[i]->time < newValue) {
			zeroIndex_ = i;
			return;
		}
	}
}

double
EventList::createSlopeRatioEvents(
		const Transition::SlopeRatio& slopeRatio,
		double baseline, double parameterDelta, double min, double max, int eventIndex)
{
	double temp = 0.0, temp1 = 0.0, intervalTime = 0.0, sum = 0.0, factor = 0.0;
	double baseTime = 0.0, endTime = 0.0, totalTime = 0.0, delta = 0.0;
	double startValue;
	double pointTime, pointValue;

	Transition::getPointData(*slopeRatio.pointList.front(), model_, pointTime, pointValue);
	baseTime = pointTime;
	startValue = pointValue;

	Transition::getPointData(*slopeRatio.pointList.back(), model_, pointTime, pointValue);
	endTime = pointTime;
	delta = pointValue - startValue;

	temp = slopeRatio.totalSlopeUnits();
	totalTime = endTime - baseTime;

	int numSlopes = slopeRatio.slopeList.size();
	std::vector<double> newPointValues(numSlopes - 1);
	for (int i = 1; i < numSlopes + 1; i++) {
		temp1 = slopeRatio.slopeList[i - 1]->slope / temp; /* Calculate normal slope */

		/* Calculate time interval */
		intervalTime = Transition::getPointTime(*slopeRatio.pointList[i], model_)
				- Transition::getPointTime(*slopeRatio.pointList[i - 1], model_);

		/* Apply interval percentage to slope */
		temp1 = temp1 * (intervalTime / totalTime);

		/* Multiply by delta and add to last point */
		temp1 = temp1 * delta;
		sum += temp1;

		if (i < numSlopes) {
			newPointValues[i - 1] = temp1;
		}
	}
	factor = delta / sum;
	temp = startValue;

	double value = 0.0;
	for (unsigned int i = 0, size = slopeRatio.pointList.size(); i < size; i++) {
		const Transition::Point& point = *slopeRatio.pointList[i];

		if (i >= 1 && i < slopeRatio.pointList.size() - 1) {
			pointTime = Transition::getPointTime(point, model_);

			pointValue = newPointValues[i - 1];
			pointValue *= factor;
			pointValue += temp;
			temp = pointValue;
		} else {
			Transition::getPointData(point, model_, pointTime, pointValue);
		}

		value = baseline + ((pointValue / 100.0) * parameterDelta);
		if (value < min) {
			value = min;
		} else if (value > max) {
			value = max;
		}
		if (!point.isPhantom) {
			insertEvent(eventIndex, pointTime, value);
		}
	}

	return value;
}

// It is assumed that postureList.size() >= 2.
void
EventList::applyRule(const Rule& rule, const std::vector<const Posture*>& postureList, const double* tempos, int postureIndex)
{
	int cont;
	int currentType;
	double currentValueDelta, value, lastValue;
	double ruleSymbols[5] = {0.0, 0.0, 0.0, 0.0, 0.0};
	double tempTime;
	double targets[4];
	Event* tempEvent = nullptr;

	rule.evaluateExpressionSymbols(tempos, postureList, model_, ruleSymbols);

	evs.Multiplier = 1.0 / (double) (postureData_[postureIndex].ruleTempo);

	int type = rule.numberOfExpressions();
	setDuration((int) (ruleSymbols[0] * evs.Multiplier));

	ruleData_[currentRule_].firstPosture = postureIndex;
	ruleData_[currentRule_].lastPosture = postureIndex + (type - 1);
	ruleData_[currentRule_].beat = (ruleSymbols[1] * evs.Multiplier) + (double) evs.ZeroRef;
	ruleData_[currentRule_++].duration = ruleSymbols[0] * evs.Multiplier;
	ruleData_.push_back(RuleData());

	switch (type) {
	/* Note: Case 4 should execute all of the below, case 3 the last two */
	case 4:
		if (postureList.size() == 4) {
			postureData_[postureIndex + 3].onset = (double) evs.ZeroRef + ruleSymbols[1];
			tempEvent = insertEvent(-1, ruleSymbols[3], 0.0);
			if (tempEvent) tempEvent->flag = 1;
		}
	case 3:
		if (postureList.size() >= 3) {
			postureData_[postureIndex + 2].onset = (double) evs.ZeroRef + ruleSymbols[1];
			tempEvent = insertEvent(-1, ruleSymbols[2], 0.0);
			if (tempEvent) tempEvent->flag = 1;
		}
	case 2:
		postureData_[postureIndex + 1].onset = (double) evs.ZeroRef + ruleSymbols[1];
		tempEvent = insertEvent(-1, 0.0, 0.0);
		if (tempEvent) tempEvent->flag = 1;
		break;
	}

	//tempTargets = (List *) [rule parameterList];

	/* Loop through the parameters */
	for (unsigned int i = 0, size = model_.parameterList().size(); i < size; ++i) {
		/* Get actual parameter target values */
		targets[0] = postureList[0]->getParameterTarget(i);
		targets[1] = postureList[1]->getParameterTarget(i);
		targets[2] = (postureList.size() >= 3) ? postureList[2]->getParameterTarget(i) : 0.0;
		targets[3] = (postureList.size() == 4) ? postureList[3]->getParameterTarget(i) : 0.0;

		/* Optimization, Don't calculate if no changes occur */
		cont = 1;
		switch (type) {
		case DIPHONE:
			if (targets[0] == targets[1]) {
				cont = 0;
			}
			break;
		case TRIPHONE:
			if ((targets[0] == targets[1]) && (targets[0] == targets[2])) {
				cont = 0;
			}
			break;
		case TETRAPHONE:
			if ((targets[0] == targets[1]) && (targets[0] == targets[2]) && (targets[0] == targets[3])) {
				cont = 0;
			}
			break;
		}

		insertEvent(i, 0.0, targets[0]);

		if (cont) {
			currentType = DIPHONE;
			currentValueDelta = targets[1] - targets[0];
			lastValue = targets[0];
			//lastValue = 0.0;

			const std::shared_ptr<Transition> transition = rule.getParamProfileTransition(i);
			if (!transition) {
				THROW_EXCEPTION(UnavailableResourceException, "Rule transition not found: " << i << '.');
			}

			/* Apply lists to parameter */
			for (unsigned int j = 0; j < transition->pointOrSlopeList().size(); ++j) {
				const Transition::PointOrSlope& pointOrSlope = *transition->pointOrSlopeList()[j];
				if (pointOrSlope.isSlopeRatio()) {
					const auto& slopeRatio = dynamic_cast<const Transition::SlopeRatio&>(pointOrSlope);

					if (slopeRatio.pointList[0]->type != currentType) { //TODO: check pointList.size() > 0
						currentType = slopeRatio.pointList[0]->type;
						targets[currentType - 2] = lastValue;
						currentValueDelta = targets[currentType - 1] - lastValue;
					}
					value = createSlopeRatioEvents(
							slopeRatio, targets[currentType - 2], currentValueDelta,
							min_[i], max_[i], i);
				} else {
					const auto& point = dynamic_cast<const Transition::Point&>(pointOrSlope);

					if (point.type != currentType) {
						currentType = point.type;
						targets[currentType - 2] = lastValue;
						currentValueDelta = targets[currentType - 1] - lastValue;
					}
					double pointTime;
					Transition::getPointData(point, model_,
									targets[currentType - 2], currentValueDelta, min_[i], max_[i],
									pointTime, value);
					if (!point.isPhantom) {
						insertEvent(i, pointTime, value);
					}
				}
				lastValue = value;
			}
		}
		//else {
		//	insertEvent(i, 0.0, targets[0]);
		//}
	}

	/* Special Event Profiles */
	for (unsigned int i = 0, size = model_.parameterList().size(); i < size; ++i) {
		const std::shared_ptr<Transition> specialTransition = rule.getSpecialProfileTransition(i);
		if (specialTransition) {
			for (unsigned int j = 0; j < specialTransition->pointOrSlopeList().size(); ++j) {
				const Transition::PointOrSlope& pointOrSlope = *specialTransition->pointOrSlopeList()[j];
				const auto& point = dynamic_cast<const Transition::Point&>(pointOrSlope);

				/* calculate time of event */
				tempTime = Transition::getPointTime(point, model_);

				/* Calculate value of event */
				value = ((point.value / 100.0) * (max_[i] - min_[i]));
				//maxValue = value;

				/* insert event into event list */
				insertEvent(i + 16U, tempTime, value);
			}
		}
	}

	setZeroRef((int) (ruleSymbols[0] * evs.Multiplier) + evs.ZeroRef);
	tempEvent = insertEvent(-1, 0.0, 0.0);
	if (tempEvent) tempEvent->flag = 1;
}

void
EventList::generateEventList()
{
	for (unsigned int i = 0; i < 16; i++) { //TODO: replace hard-coded value
		const Parameter& param = model_.getParameter(i);
		min_[i] = (double) param.minimum();
		max_[i] = (double) param.maximum();
	}

	/* Calculate Rhythm including regression */
	for (int i = 0; i < currentFoot_; i++) {
		int rus = feet_[i].end - feet_[i].start + 1;
		/* Apply rhythm model */
		double footTempo;
		if (feet_[i].marked) {
			double tempTempo = 117.7 - (19.36 * (double) rus);
			feet_[i].tempo -= tempTempo / 180.0;
			footTempo = evs.GlobalTemp * feet_[i].tempo;
		} else {
			double tempTempo = 18.5 - (2.08 * (double) rus);
			feet_[i].tempo -= tempTempo / 140.0;
			footTempo = evs.GlobalTemp * feet_[i].tempo;
		}
		for (int j = feet_[i].start; j < feet_[i].end + 1; j++) {
			postureTempo_[j] *= footTempo;
			if (postureTempo_[j] < 0.2) {
				postureTempo_[j] = 0.2;
			} else if (postureTempo_[j] > 2.0) {
				postureTempo_[j] = 2.0;
			}
		}
	}

	unsigned int basePostureIndex = 0;
	std::vector<const Posture*> tempPostureList;
	while (basePostureIndex < currentPosture_) {
		tempPostureList.clear();
		for (unsigned int i = 0; i < 4; i++) {
			unsigned int postureIndex = basePostureIndex + i;
			if (postureIndex <= currentPosture_ && postureData_[postureIndex].posture) {
				tempPostureList.push_back(postureData_[postureIndex].posture);
			} else {
				break;
			}
		}
		if (tempPostureList.size() < 2) {
			break;
		}
		unsigned int ruleIndex = 0;
		const Rule* tempRule = model_.findFirstMatchingRule(tempPostureList, ruleIndex);
		if (tempRule == nullptr) {
			THROW_EXCEPTION(UnavailableResourceException, "Could not find a matching rule.");
		}

		ruleData_[currentRule_].number = ruleIndex + 1U;

		applyRule(*tempRule, tempPostureList, &postureTempo_[basePostureIndex], basePostureIndex);

		basePostureIndex += tempRule->numberOfExpressions() - 1;
	}

	//[dataPtr[numElements-1] setFlag:1];
}

void
EventList::setFullTimeScale()
{
	evs.ZeroRef = 0;
	zeroIndex_ = 0;
	evs.Duration = list_.back()->time + 100;
}

void
EventList::applyIntonation()
{
	int tgRandom;
	int firstFoot, endFoot;
	int ruleIndex = 0, postureIndex;
	int i, j, k;
	double startTime, endTime, pretonicDelta, offsetTime = 0.0;
	double randomSemitone, randomSlope;

	evs.ZeroRef = 0;
	zeroIndex_ = 0;
	evs.Duration = list_.back()->time + 100;

	intonationPoints_.clear();

	std::shared_ptr<const Category> vocoidCategory = model_.findCategory("vocoid");
	if (!vocoidCategory) {
		THROW_EXCEPTION(UnavailableResourceException, "Could not find the category \"vocoid\".");
	}

	std::uniform_int_distribution<> intRandDist0(0, tgCount_[0] > 0 ? tgCount_[0] - 1 : 0);
	std::uniform_int_distribution<> intRandDist1(0, tgCount_[1] > 0 ? tgCount_[1] - 1 : 0);
	std::uniform_int_distribution<> intRandDist2(0, tgCount_[2] > 0 ? tgCount_[2] - 1 : 0);
	std::uniform_int_distribution<> intRandDist3(0, tgCount_[3] > 0 ? tgCount_[3] - 1 : 0);

	for (i = 0; i < currentToneGroup_; i++) {
		firstFoot = toneGroups_[i].startFoot;
		endFoot = toneGroups_[i].endFoot;

		startTime = postureData_[feet_[firstFoot].start].onset;
		endTime = postureData_[feet_[endFoot].end].onset;

		//printf("Tg: %d First: %d  end: %d  StartTime: %f  endTime: %f\n", i, firstFoot, endFoot, startTime, endTime);

		if (useFixedIntonationParameters_) {
			evs.IntonParams = fixedIntonationParameters_;
		} else {
			switch (toneGroups_[i].type) {
			default:
			case TONE_GROUP_TYPE_STATEMENT:
				if (tgUseRandom_) {
					tgRandom = intRandDist0(randSrc_);
				} else {
					tgRandom = 0;
				}
				evs.IntonParams = &tgParameters_[0][tgRandom * 10];
				break;
			case TONE_GROUP_TYPE_EXCLAMATION:
				if (tgUseRandom_) {
					tgRandom = intRandDist0(randSrc_);
				} else {
					tgRandom = 0;
				}
				evs.IntonParams = &tgParameters_[0][tgRandom * 10];
				break;
			case TONE_GROUP_TYPE_QUESTION:
				if (tgUseRandom_) {
					tgRandom = intRandDist1(randSrc_);
				} else {
					tgRandom = 0;
				}
				evs.IntonParams = &tgParameters_[1][tgRandom * 10];
				break;
			case TONE_GROUP_TYPE_CONTINUATION:
				if (tgUseRandom_) {
					tgRandom = intRandDist2(randSrc_);
				} else {
					tgRandom = 0;
				}
				evs.IntonParams = &tgParameters_[2][tgRandom * 10];
				break;
			case TONE_GROUP_TYPE_SEMICOLON:
				if (tgUseRandom_) {
					tgRandom = intRandDist3(randSrc_);
				} else {
					tgRandom = 0;
				}
				evs.IntonParams = &tgParameters_[3][tgRandom * 10];
				break;
			}
		}

		//printf("Intonation Parameters: Type : %d  random: %d\n", toneGroups[i].type, tgRandom);
		//for (j = 0; j<6; j++)
		//	printf("%f ", intonParms[j]);
		//printf("\n");

		const double deltaTime = endTime - startTime;
		pretonicDelta = deltaTime < EPS ? 0.0 : evs.IntonParams[1] / deltaTime;
		//printf("Pretonic Delta = %f time = %f\n", pretonicDelta, (endTime - startTime));

		/* Set up intonation boundary variables */
		for (j = firstFoot; j <= endFoot; j++) {
			postureIndex = feet_[j].start;
			while (!postureData_[postureIndex].posture->isMemberOfCategory(*vocoidCategory)) {
				postureIndex++;
				//printf("Checking posture %s for vocoid\n", [posture[postureIndex].posture symbol]);
				if (postureIndex > feet_[j].end) {
					postureIndex = feet_[j].start;
					break;
				}
			}

			if (!feet_[j].marked) {
				for (k = 0; k < currentRule_; k++) {
					if ((postureIndex >= ruleData_[k].firstPosture) && (postureIndex <= ruleData_[k].lastPosture)) {
						ruleIndex = k;
						break;
					}
				}

				if (tgUseRandom_) {
					randomSemitone = randDist_(randSrc_) * evs.IntonParams[3] - evs.IntonParams[3] / 2.0;
					randomSlope = randDist_(randSrc_) * 0.015 + 0.01;
				} else {
					randomSemitone = 0.0;
					randomSlope = 0.02;
				}

				//printf("postureIndex = %d onsetTime : %f Delta: %f\n", postureIndex,
				//	postures[postureIndex].onset-startTime,
				//	((postures[postureIndex].onset-startTime)*pretonicDelta) + intonParms[1] + randomSemitone);

				addIntonationPoint((postureData_[postureIndex].onset - startTime) * pretonicDelta + evs.IntonParams[1] + randomSemitone,
							offsetTime, randomSlope, ruleIndex);
			} else { /* Tonic */
				if (toneGroups_[i].type == 3) {
					randomSlope = 0.01;
				} else {
					randomSlope = 0.02;
				}

				for (k = 0; k < currentRule_; k++) {
					if ((postureIndex >= ruleData_[k].firstPosture) && (postureIndex <= ruleData_[k].lastPosture)) {
						ruleIndex = k;
						break;
					}
				}

				if (tgUseRandom_) {
					randomSemitone = randDist_(randSrc_) * evs.IntonParams[6] - evs.IntonParams[6] / 2.0;
					randomSlope += randDist_(randSrc_) * 0.03;
				} else {
					randomSemitone = 0.0;
					randomSlope += 0.03;
				}
				addIntonationPoint(evs.IntonParams[2] + evs.IntonParams[1] + randomSemitone,
							offsetTime, randomSlope, ruleIndex);

				postureIndex = feet_[j].end;
				for (k = ruleIndex; k < currentRule_; k++) {
					if ((postureIndex >= ruleData_[k].firstPosture) && (postureIndex <= ruleData_[k].lastPosture)) {
						ruleIndex = k;
						break;
					}
				}

				addIntonationPoint(evs.IntonParams[2] + evs.IntonParams[1] + evs.IntonParams[5],
							0.0, 0.0, ruleIndex);
			}
			offsetTime = -40.0;
		}
	}
	addIntonationPoint(evs.IntonParams[2] + evs.IntonParams[1] + evs.IntonParams[5],
				0.0, 0.0, currentRule_ - 1);
}

void
EventList::applyIntonationSmooth()
{
	setFullTimeScale();
	//tempPoint = [[IntonationPoint alloc] initWithEventList: self];
	//[tempPoint setSemitone: -20.0];
	//[tempPoint setSemitone: -20.0];
	//[tempPoint setRuleIndex: 0];
	//[tempPoint setOffsetTime: 10.0 - [self getBeatAtIndex:(int) 0]];

	//[intonationPoints insertObject: tempPoint at:0];

	for (unsigned int j = 0; j < intonationPoints_.size() - 1; j++) {
		const IntonationPoint& point1 = intonationPoints_[j];
		const IntonationPoint& point2 = intonationPoints_[j + 1];

		double x1 = point1.absoluteTime() / 4.0;
		double y1 = point1.semitone() + 20.0;
		double m1 = point1.slope();

		double x2 = point2.absoluteTime() / 4.0;
		double y2 = point2.semitone() + 20.0;
		double m2 = point2.slope();

		double x12 = x1 * x1;
		double x13 = x12 * x1;

		double x22 = x2 * x2;
		double x23 = x22 * x2;

		double denominator = x2 - x1;
		denominator = denominator * denominator * denominator;

//		double d = ( -(y2 * x13) + 3 * y2 * x12 * x2 + m2 * x13 * x2 + m1 * x12 * x22 - m2 * x12 * x22 - 3 * x1 * y1 * x22 - m1 * x1 * x23 + y1 * x23 )
//			/ denominator;
		double c = ( -(m2 * x13) - 6 * y2 * x1 * x2 - 2 * m1 * x12 * x2 - m2 * x12 * x2 + 6 * x1 * y1 * x2 + m1 * x1 * x22 + 2 * m2 * x1 * x22 + m1 * x23 )
			/ denominator;
		double b = ( 3 * y2 * x1 + m1 * x12 + 2 * m2 * x12 - 3 * x1 * y1 + 3 * x2 * y2 + m1 * x1 * x2 - m2 * x1 * x2 - 3 * y1 * x2 - 2 * m1 * x22 - m2 * x22 )
			/ denominator;
		double a = ( -2 * y2 - m1 * x1 - m2 * x1 + 2 * y1 + m1 * x2 + m2 * x2) / denominator;

		insertEvent(32, point1.absoluteTime(), point1.semitone());
		//printf("Inserting Point %f\n", [point1 semitone]);
		double yTemp = (3.0 * a * x12) + (2.0 * b * x1) + c;
		insertEvent(33, point1.absoluteTime(), yTemp);
		yTemp = (6.0 * a * x1) + (2.0 * b);
		insertEvent(34, point1.absoluteTime(), yTemp);
		yTemp = 6.0 * a;
		insertEvent(35, point1.absoluteTime(), yTemp);
	}
	//[intonationPoints removeObjectAt:0];

	//[self insertEvent:32 atTime: 0.0 withValue: -20.0]; /* A value of -20.0 in bin 32 should produce a
	//							    linear interp to -20.0 */
}

void
EventList::addIntonationPoint(double semitone, double offsetTime, double slope, int ruleIndex)
{
	if (ruleIndex > currentRule_) {
		return;
	}

	IntonationPoint iPoint(this);
	iPoint.setRuleIndex(ruleIndex);
	iPoint.setOffsetTime(offsetTime);
	iPoint.setSemitone(semitone);
	iPoint.setSlope(slope);

	double time = iPoint.absoluteTime();
	for (unsigned int i = 0; i < intonationPoints_.size(); i++) {
		if (time < intonationPoints_[i].absoluteTime()) {
			intonationPoints_.insert(intonationPoints_.begin() + i, iPoint);
			return;
		}
	}

	intonationPoints_.push_back(iPoint);
}

void
EventList::generateOutput(std::ostream& trmParamStream)
{
	double currentValues[36];
	double currentDeltas[36];
	double temp;
	float table[16];

	if (list_.empty()) {
		return;
	}

	for (int i = 0; i < 16; i++) {
		unsigned int j = 1;
		while ((temp = list_[j]->getValue(i)) == GS_EVENTLIST_INVALID_EVENT_VALUE) {
			j++;
			if (j >= list_.size()) break;
		}
		currentValues[i] = list_[0]->getValue(i);
		if (j < list_.size()) {
			currentDeltas[i] = ((temp - currentValues[i]) / (double) (list_[j]->time)) * 4.0;
		} else {
			currentDeltas[i] = 0.0;
		}
	}
	for (int i = 16; i < 36; i++) {
		currentValues[i] = currentDeltas[i] = 0.0;
	}

	if (smoothIntonation_) {
		unsigned int j = 0;
		while ((temp = list_[j]->getValue(32)) == GS_EVENTLIST_INVALID_EVENT_VALUE) {
			j++;
			if (j >= list_.size()) break;
		}
		if (j < list_.size()) {
			currentValues[32] = list_[j]->getValue(32);
		} else {
			currentValues[32] = 0.0;
		}
		currentDeltas[32] = 0.0;
	} else {
		unsigned int j = 1;
		while ((temp = list_[j]->getValue(32)) == GS_EVENTLIST_INVALID_EVENT_VALUE) {
			j++;
			if (j >= list_.size()) break;
		}
		currentValues[32] = list_[0]->getValue(32);
		if (j < list_.size()) {
			currentDeltas[32] = ((temp - currentValues[32]) / (double) (list_[j]->time)) * 4.0;
		} else {
			currentDeltas[32] = 0.0;
		}
		currentValues[32] = -20.0;
	}

	unsigned int index = 1;
	int currentTime = 0;
	int nextTime = list_[1]->time;
	while (index < list_.size()) {

		for (int j = 0; j < 16; j++) {
			table[j] = (float) currentValues[j] + (float) currentValues[j + 16];
		}
		if (!evs.MicroFlag_) table[0] = 0.0;
		if (driftFlag_)  table[0] += static_cast<float>(driftGenerator_.drift());
		if (evs.MacroFlag)  table[0] += static_cast<float>(currentValues[32]);

		table[0] += static_cast<float>(evs.PitchMean);

		//trmParamStream << std::fixed << std::setprecision(3);
		trmParamStream << table[0];
		for (int k = 1; k < 7; ++k) {
			trmParamStream << ' ' << table[k];
		}
		for (int k = 7; k < 15; ++k) { // R1 - R8
			trmParamStream << ' ' << table[k] * radiusCoef[k - 7];
		}
		trmParamStream << ' ' << table[15];
		trmParamStream << '\n';

		for (int j = 0; j < 32; j++) {
			if (currentDeltas[j]) {
				currentValues[j] += currentDeltas[j];
			}
		}

		if (smoothIntonation_) {
			currentDeltas[34] += currentDeltas[35];
			currentDeltas[33] += currentDeltas[34];
			currentValues[32] += currentDeltas[33];
		} else {
			if (currentDeltas[32]) {
				currentValues[32] += currentDeltas[32];
			}
		}
		currentTime += 4;

		if (currentTime >= nextTime) {
			++index;
			if (index == list_.size()) {
				break;
			}
			nextTime = list_[index]->time;
			for (int j = 0; j < 33; j++) { /* 32? 33? */
				if (list_[index - 1]->getValue(j) != GS_EVENTLIST_INVALID_EVENT_VALUE) {
					unsigned int k = index;
					while ((temp = list_[k]->getValue(j)) == GS_EVENTLIST_INVALID_EVENT_VALUE) {
						if (k >= list_.size() - 1U) {
							currentDeltas[j] = 0.0;
							break;
						}
						k++;
					}
					if (temp != GS_EVENTLIST_INVALID_EVENT_VALUE) {
						currentDeltas[j] = (temp - currentValues[j]) /
									(double) (list_[k]->time - currentTime) * 4.0;
					}
				}
			}
			if (smoothIntonation_) {
				if (list_[index - 1]->getValue(33) != GS_EVENTLIST_INVALID_EVENT_VALUE) {
					currentValues[32] = list_[index - 1]->getValue(32);
					currentDeltas[32] = 0.0;
					currentDeltas[33] = list_[index - 1]->getValue(33);
					currentDeltas[34] = list_[index - 1]->getValue(34);
					currentDeltas[35] = list_[index - 1]->getValue(35);
				}
			}
		}
	}

	if (Log::debugEnabled) {
		printDataStructures();
	}
}

void
EventList::clearMacroIntonation()
{
	for (unsigned int i = 0, size = list_.size(); i < size; ++i) {
		auto& event = list_[i];
		for (unsigned int j = 32; j < 36; ++j) {
			event->setValue(GS_EVENTLIST_INVALID_EVENT_VALUE, j);
		}
	}
}

void
EventList::printDataStructures()
{
	printf("Tone Groups %d\n", currentToneGroup_);
	for (int i = 0; i < currentToneGroup_; i++) {
		printf("%d  start: %d  end: %d  type: %d\n", i, toneGroups_[i].startFoot, toneGroups_[i].endFoot,
			toneGroups_[i].type);
	}

	printf("\nFeet %d\n", currentFoot_);
	for (int i = 0; i < currentFoot_; i++) {
		printf("%d  tempo: %f start: %d  end: %d  marked: %d last: %d onset1: %f onset2: %f\n", i, feet_[i].tempo,
			feet_[i].start, feet_[i].end, feet_[i].marked, feet_[i].last, feet_[i].onset1, feet_[i].onset2);
	}

	printf("\nPostures %d\n", currentPosture_);
	for (unsigned int i = 0; i < currentPosture_; i++) {
		printf("%u  \"%s\" tempo: %f syllable: %d onset: %f ruleTempo: %f\n",
			 i, postureData_[i].posture->name().c_str(), postureTempo_[i], postureData_[i].syllable, postureData_[i].onset, postureData_[i].ruleTempo);
	}

	printf("\nRules %d\n", currentRule_);
	for (int i = 0; i < currentRule_; i++) {
		printf("Number: %d  start: %d  end: %d  duration %f\n", ruleData_[i].number, ruleData_[i].firstPosture,
			ruleData_[i].lastPosture, ruleData_[i].duration);
	}
#if 0
	printf("\nEvents %lu\n", list_.size());
	for (unsigned int i = 0; i < list_.size(); i++) {
		const Event& event = *list_[i];
		printf("  Event: time=%d flag=%d\n    Values: ", event.time, event.flag);

		for (int j = 0; j < 16; j++) {
			printf("%.3f ", event.getValue(j));
		}
		printf("\n            ");
		for (int j = 16; j < 32; j++) {
			printf("%.3f ", event.getValue(j));
		}
		printf("\n            ");
		for (int j = 32; j < Event::EVENTS_SIZE; j++) {
			printf("%.3f ", event.getValue(j));
		}
		printf("\n");
	}
#endif
}

} /* namespace TRMControlModel */
} /* namespace GS */
