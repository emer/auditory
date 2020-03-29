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

import ()

type Model struct {
	Categories []Category
	Params     []Param
	//	//Codes []Code
	//	//Postures []Posture
	//	//Rules []Rule
	//	//EquationGrps []EquationGrp
	//	//TransitionGrps []TransitionGrp
	//	//TransitionGrpsSp []TransitionGrp
	Formula Formula
}

// Reset
func (mdl *Model) Reset() {
	mdl.Categories = mdl.Categories[:0]
	mdl.Params = mdl.Params[:0]
	//mdl.Codes = mdl.Codes[:0]
	//mdl.Postures = mdl.Postures[:0]
	//mdl.Rules = mdl.Rules[:0]
	//mdl.EquationGrps = mdl.EquationGrps[:0]
	//mdl.TransitionGrps = mdl.TransitionGrps[:0]
	//mdl.TransitionGrpsSp = mdl.TransitionGrpsSp[:0]
	mdl.Formula.Clear()
}

//// Load
//func (mdl *Model) Load(configDir, configFile string) {
////    Reset()
//// 	fp := configDir + configFile
////
//// 	LOG_DEBUG("Loading xml configuration: " << fp)
//// 	XMLConfigFileReader cfg(*this, fp);
//// 	cfg.loadModel();
//}
//
//// Save
//func (mdl *Model) Save(configDir, configFile string) {
//	fp := configDir + configFile;
//
//	// LOG_DEBUG("Saving xml configuration: " << fp);
//	// XMLConfigFileWriter cfg(*this, fp);
//	cfg.saveModel();
//}
//
////
//// func (mdl *Model) PrintInfo() const {
//// 	no way I am going to port this code - develop print code as needed
//// }
//
//// ParamIdx returns the index or -1 if not found
//func (mdl *Model) ParamIdx(nm string) int {
//	for i, p := range mdl.Params {
//		if p.Name == nm {
//			return i
//		}
//	}
//	return -1
//}
//
//// ParamIdx returns the address of the named category or nil if not found
//func (mdl *Model) CategoryTry(nm string) *Category {
//	for i, c := range mdl.Categories {
//		if c.Name == nm {
//			return &c
//		}
//	}
//	return nil
//}
//
//
////
//func (mdl *Model) evalEquationFormula(const Equation& equation) const
//{
//	return equation.evalFormula(formulaCodebolList_);
//}
//
////
//func (mdl *Model) findEquationGroupName(const std::string& name) const
//{
//	for (const auto& item : equationGroupList_) {
//		if (item.name == name) {
//			return true;
//		}
//	}
//	return false;
//}
//
////
//func (mdl *Model) findEquationName(const std::string& name) const
//{
//	for (const auto& group : equationGroupList_) {
//		for (const auto& item : group.equationList) {
//			if (item->name() == name) {
//				return true;
//			}
//		}
//	}
//	return false;
//}
//
////
//func (mdl *Model) findEquationIndex(const std::string& name, unsigned int& groupIndex, unsigned int& index) const
//{
//	for (unsigned int i = 0, groupListSize = equationGroupList_.size(); i < groupListSize; ++i) {
//		const auto& group = equationGroupList_[i];
//		for (unsigned int j = 0, size = group.equationList.size(); j < size; ++j) {
//			if (group.equationList[j]->name() == name) {
//				groupIndex = i;
//				index = j;
//				return true;
//			}
//		}
//	}
//	return false;
//}
//
//std::shared_ptr<Equation>
//Model::findEquation(const std::string& name)
//{
//	for (const auto& group : equationGroupList_) {
//		for (const auto& equation : group.equationList) {
//			if (equation->name() == name) {
//				return equation;
//			}
//		}
//	}
//	return std::shared_ptr<Equation>();
//}
//
////
//func (mdl *Model) getParameterMinimum(unsigned int parameterIndex) const
//{
//	if (parameterIndex >= parameterList_.size()) {
//		THROW_EXCEPTION(InvalidParameterException, "Invalid parameter index: " << parameterIndex << '.');
//	}
//
//	return parameterList_[parameterIndex].minimum();
//}
//
////
//func (mdl *Model) getParameterMaximum(unsigned int parameterIndex) const
//{
//	if (parameterIndex >= parameterList_.size()) {
//		THROW_EXCEPTION(InvalidParameterException, "Invalid parameter index: " << parameterIndex << '.');
//	}
//
//	return parameterList_[parameterIndex].maximum();
//}
//
//const Parameter&
//Model::getParameter(unsigned int parameterIndex) const
//{
//	if (parameterIndex >= parameterList_.size()) {
//		THROW_EXCEPTION(InvalidParameterException, "Invalid parameter index: " << parameterIndex << '.');
//	}
//
//	return parameterList_[parameterIndex];
//}
//
////
//func (mdl *Model) findParameterName(const std::string& name) const
//{
//	for (const auto& item : parameterList_) {
//		if (item.name() == name) {
//			return true;
//		}
//	}
//	return false;
//}
//
////
//func (mdl *Model) findCodebolName(const std::string& name) const
//{
//	for (const auto& item : symbolList_) {
//		if (item.name() == name) {
//			return true;
//		}
//	}
//	return false;
//}
//
// * Find a Transition with the given name.
// *
//// returns an empty shared_ptr if the Transition was not found.
// */
//const std::shared_ptr<Transition>
//Model::findTransition(const std::string& name) const
//{
//	for (const auto& group : transitionGroupList_) {
//		for (const auto& transition : group.transitionList) {
//			if (transition->name() == name) {
//				return transition;
//			}
//		}
//	}
//	return std::shared_ptr<Transition>();
//}
//
////
//func (mdl *Model) findTransitionGroupName(const std::string& name) const
//{
//	for (const auto& item : transitionGroupList_) {
//		if (item.name == name) {
//			return true;
//		}
//	}
//	return false;
//}
//
////
//func (mdl *Model) findTransitionName(const std::string& name) const
//{
//	for (const auto& group : transitionGroupList_) {
//		for (const auto& item : group.transitionList) {
//			if (item->name() == name) {
//				return true;
//			}
//		}
//	}
//	return false;
//}
//
////
//func (mdl *Model) findTransitionIndex(const std::string& name, unsigned int& groupIndex, unsigned int& index) const
//{
//	for (unsigned int i = 0, groupListSize = transitionGroupList_.size(); i < groupListSize; ++i) {
//		const auto& group = transitionGroupList_[i];
//		for (unsigned int j = 0, size = group.transitionList.size(); j < size; ++j) {
//			if (group.transitionList[j]->name() == name) {
//				groupIndex = i;
//				index = j;
//				return true;
//			}
//		}
//	}
//	return false;
//}
//
// * Find a Special Transition with the given name.
// *
//// returns an empty shared_ptr if the Transition was not found.
// */
//const std::shared_ptr<Transition>
//Model::findSpecialTransition(const std::string& name) const
//{
//	for (const auto& group : specialTransitionGroupList_) {
//		for (const auto& transition : group.transitionList) {
//			if (transition->name() == name) {
//				return transition;
//			}
//		}
//	}
//	return std::shared_ptr<Transition>();
//}
//
////
//func (mdl *Model) findSpecialTransitionGroupName(const std::string& name) const
//{
//	for (const auto& item : specialTransitionGroupList_) {
//		if (item.name == name) {
//			return true;
//		}
//	}
//	return false;
//}
//
////
//func (mdl *Model) findSpecialTransitionName(const std::string& name) const
//{
//	for (const auto& group : specialTransitionGroupList_) {
//		for (const auto& item : group.transitionList) {
//			if (item->name() == name) {
//				return true;
//			}
//		}
//	}
//	return false;
//}
//
////
//func (mdl *Model) findSpecialTransitionIndex(const std::string& name, unsigned int& groupIndex, unsigned int& index) const
//{
//	for (unsigned int i = 0, groupListSize = specialTransitionGroupList_.size(); i < groupListSize; ++i) {
//		const auto& group = specialTransitionGroupList_[i];
//		for (unsigned int j = 0, size = group.transitionList.size(); j < size; ++j) {
//			if (group.transitionList[j]->name() == name) {
//				groupIndex = i;
//				index = j;
//				return true;
//			}
//		}
//	}
//	return false;
//}
//
// * Finds the first Rule that matches the given sequence of Postures.
// */
//const Rule*
//Model::findFirstMatchingRule(const std::vector<const Posture*>& postureSequence, unsigned int& ruleIndex) const
//{
//	if (ruleList_.empty()) {
//		ruleIndex = 0;
//		return nullptr;
//	}
//
//	unsigned int i = 0;
//	for (const auto& r : ruleList_) {
//		if (r->numberOfExpressions() <= postureSequence.size()) {
//			if (r->evalBooleanExpression(postureSequence)) {
//				ruleIndex = i;
//				return r.get();
//			}
//		}
//		++i;
//	}
//
//	ruleIndex = 0;
//	return nullptr;
//}
//
//func (mdl *Model) PostureNameTry(nm string) *Posture {
//   for _, p := range mdl.Postures {}
//		if p.Name == name {
//			return p
//		}
//	return nil
//}
//
