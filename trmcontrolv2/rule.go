// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source  is governed by a BSD-style
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

const rightParen = ")"
const leftParen = "("
const wildCard = "*"
const orOpSymb = "or"
const notOpSymb = "not"
const xorOpSymb = "xor"
const andOpSymb = "and"

type SymbolType int

const (
	// SymInvalid
	SymInvalid SymbolType = iota

	// 
	SymOrOp

	// 
	SymNotOp
	
	// 
	SymXorOp
	
	//
	SymAndOp
	
	//
	SymRtParen
	
	//
	SymLftParen
	
	//
	SymString
	
	SymTypeN
)
//go:generate stringer -type=SymbolType

var Kit_SymbolType = kit.Enums.AddEnum(SymTypeN, NotBitFlag, nil)

// Parse
type Parse struct {
	Model *Model
	Str string
	Pos int
	Symbol string
	SymType SymbolType
}

func (pars *Parse) Defaults() {
	pars.Pos = 0
	pars.SymType = symInvalid
}

func (pars *Parse) Init(s string, model *Model) error {
	if len(s) == 0 {
		errors.New("empty string parameter not allowed")
	}
	pars.Model = model
	pars.Str = strings.TrimSpace()
	pars.NextSymbol()
	return pars.Parse()
}

func (pars *Parse) Finished() bool {
	return pars.Pos >= len(pars.Str)
}

func (pars *Parse) IsSeparator(c string) bool {	
	if c == rtParens || c == lftParens {
		return true
	}
	if uni.IsSpace() {
		return true
	}
	return false
}

// SkipSpaces moves the index into string past white space
func (pars *Parse) SkipSpaces() {
	for !Finished() && strings.IsSpace(pars.Str[pars.Pos]) {
		pars.Pos++
	}
}

// NextSymbol
func (pars *Parse) NextSymbol() {
	pars.SkipSpaces()
	pars.Symbol = ""
	
	if pars.Finished() {
		pars.SymType = SymInvalid
		return
	}

	c := pars.Str[pars.Pos]
	pars.Pos++
	pars.SymType = c

	switch c {
	case rtParen:
		pars.SymType = SymRtParen
		break
	case lftParen:
		pars.SymType = SymLftParen
		break
	default:
		cnext = pars.Str[pars.Pos]  // notice that Pos has been incremented already
		for !finished() && !isSeparator(cnext)  {
			pars.Sym += cnew
			pars.Pos++
		}
		if pars.Symbol == orOpSymb {
			pars.SymType = SymOrOp
		} else if pars.Symbol == andOpSymb {
			pars.SymType = SymAndOp
		} else if pars.Symbol == notOpSymb {
			pars.SymType = SymNotOp
		} else if pars.Symbol == xorOpSymb {
			pars.SymType = SymXorOp
		} else {
			pars.SymType = SymString
		}
	}
}

// GetNode returns the next boolean node
func (pars *Parse) GetNode() (*Node, error) {
	switch pars.SymType {
		
	case SymLftParen:
	{
		var p Node

		nextSymbol()
		if pars.SymType == SymNotOp {
			// Operand.
			nextSymbol()
			Node* op(GetNode())
			// p.reset(new RuleBooleanNotExpression(std::move(op)))
		} else {
			// 1st operand.
			Node* op1(GetNode())

			// Operator.
			switch pars.SymType {
			case SymOrOp:
			{	// 2nd operand.
				nextSymbol()
				Node* op2(GetNode())
				// p.reset(new RuleBooleanOrExpression(std::move(op1), std::move(op2)))
				break;
			}
			case SymAndOp:
			{	// 2nd operand.
				nextSymbol()
				Node* op2(GetNode())
				// p.reset(new RuleBooleanAndExpression(std::move(op1), std::move(op2)))
				
			}
			case SymXorOp:
			{	// 2nd operand.
				nextSymbol()
				Node* op2(GetNode())
				// p.reset(new RuleBooleanXorExpression(std::move(op1), std::move(op2)))
				
			}
			case SymNotOp:
				return nil, errors.New("Invalid operator")
			default:
				return nil, errors.New("Missing operator")
			}
		}

		if (pars.SymType != SymRtParen) {
			return nil, errors.New("Right parenthesis not found")
		}
		nextSymbol()
		return p, nil
	}
	
	case SymString:
	{
		wild := false;
		if len(pars.Symbol) >= 2 && pars.Symbol[len(pars.Symbol - 1)] == wildCard {
			wild = true;
		}

		var name string
		if wild {
			name = pars.Symbol.substr(0, pars.Symbol.size() - 1)
		} else {
			name = pars.Symbol;
		}

		var category Category
		posture = pars.Model.PostureNameTry(name)
		if posture != nil {
			category = posture.CategoryNameTry(name)
		} else {
			if wild {
				return nil, errors.New("Asterisk at the end of a category name")
			}
			category = pars.Model.CategoryNameTry(name)
		}
		if (!category) {
			return nil, errors.New("Could not find category: ", name)
		}

 		nextSymbol()
		nt := NewNode(NodeTerminal, nil, nil)
		nt.Category = Category
		nt.Wild = wilcard
		return &nt

	}
	case SymOrOp:
		return nil, errors.New("Unexpected OR op.")
	case SymNotOp:
		return nil, errors.New("Unexpected NOT op.")
	case SymXorOp:
		return nil, errors.New("Unexpected XOR op.")
	case SymAndOp:
		return nil, errors.New("Unexpected AND op.")
	case SymRtParen:
		return nil, errors.New("Unexpected right parenthesis")
	default:
		return nil, errors.New("Missing symbol")
	}
}

// Parse
func (pars *Parse) Parse() *Node {
	root := GetNode()
	if (root.SymType != SymInvalid) {
		return nil, errors.New("Invalid text") // ToDo: this doesn't make sens
	}
	return root
}

// EvalExpr
func (r *Rule) EvalExpr(tempos float64, postures *[]Posture, model *Model, syms *float64) {
	var localTempos []float64
	
	model.Formula.Clear()
	
	if len(postures) >= 2 {
		pos := postures[0]
		model.Formula[Transition1] = pos.GetSymbolTarget(1)
		model.Formula[Qssa1] = pos.GetSymbolTarget(2)	
		 model.Formula[Qssb1] = pos.GetSymbolTarget(3)
		
		pos := postures[1]
		model.Formula[Transition2] = pos.GetSymbolTarget(1)
		model.Formula[Qssa2] = Posture.GetSymbolTarget(2)
		 model.Formula[Qssb2] = pos.GetSymbolTarget(3)
		localTempos[0] = tempos[0]
		localTempos[1] = tempos[1]
	} else {
		localTempos[0] = 0.0
		locatTempos[1] = 0.0
	}
	
	if len(postures) >= 3 {
				pos := postures[2]
	model.Formula[Transition3] = pos.GetSymbolTarget(1)
		model.Formula[Qssa3] = pos.GetSymbolTarget(2)
		 model.Formula[Qssb3] = pos.GetSymbolTarget(3)
		locatTempos[2] = tempos[2]
	} else {
		localTempos[2] = 0.0
	}
	
	if len(postures) >= 4 {
				pos := postures[3]
	model.Formula[Transition4] = pos.GetSymbolTarget(1)
		model.Formula[Qssa4] = pos.GetSymbolTarget(2)
		 model.Formula[Qssb4] = pos.GetSymbolTarget(3)
		localTempos[3] = tempos[3]		
	} else {
		localTempos[3] = 0.0
	}

	model.Formula[Tempo1] = localTempos[0]
	model.Formula[Tempo2] = localTempos[1]
	model.Formula[Tempo3] = localTempos[2]
	model.Formula[Tempo4] = localTempos[3]
	model.Formula[SYMB_RD]    = syms[0]
	model.Formula[SYMB_BEAT]  = syms[1]
	model.Formula[SYMB_MARK1] = syms[2]
	model.Formula[SYMB_MARK2] = syms[3]
	model.Formula[SYMB_MARK3] = syms[4]	
	
	
	// Execute in this order.
	if (exprSymbolEquations_.ruleDuration) {
		model.setFormulaSymbolValue(FormulaSymbol::SYMB_RD   , model.evalEquationFormula(*exprSymbolEquations_.ruleDuration));
	}
	if (exprSymbolEquations_.mark1) {
		model.setFormulaSymbolValue(FormulaSymbol::SYMB_MARK1, model.evalEquationFormula(*exprSymbolEquations_.mark1));
	}
	if (exprSymbolEquations_.mark2) {
		model.setFormulaSymbolValue(FormulaSymbol::SYMB_MARK2, model.evalEquationFormula(*exprSymbolEquations_.mark2));
	}
	if (exprSymbolEquations_.mark3) {
		model.setFormulaSymbolValue(FormulaSymbol::SYMB_MARK3, model.evalEquationFormula(*exprSymbolEquations_.mark3));
	}
	if (exprSymbolEquations_.beat) {
		model.setFormulaSymbolValue(FormulaSymbol::SYMB_BEAT , model.evalEquationFormula(*exprSymbolEquations_.beat));
	}

	syms[0] = model.Formula[Rd]
	syms[1] = model.Formula[Beat]
	syms[2] = model.Formula[Mark1]
	syms[3] = model.Formula[Mark2]
	syms[4] = model.Formula[Mark3]
}





Rule::setBooleanExpressionList(const std::vector<std::string>& exprList, const Model& model)
{
	unsigned int size = exprList.size();
	if (size < 2U || size > 4U) {
		THROW_EXCEPTION(InvalidParameterException, "Invalid number of boolean expressions: " << size << '.');
	}

	RuleBooleanNodeList testBooleanNodeList;

	for (unsigned int i = 0; i < size; ++i) {
		Parse p(exprList[i], model);
		testBooleanNodeList.push_back(p.parse());
	}

	booleanExpressionList_ = exprList;
	std::swap(booleanNodeList_, testBooleanNodeList);
}

//////////////////////////////////////////////////
// Rule
//////////////////////////////////////////////////

type ExpSymEquation struct {
	Duration *Equation
	Beat *Equation
	Mark1 *Equation
	Mark2 *Equation
	Mark3 *Equation	
}
type Rule struct {
	BoolExprs []string
	ParamProfileTransitions []Transitions
	SpecialProfileTransitions []Transitions
	ExprSymEquations ExprSymEquations
	Comment string
	Nodes RuleNodes
}

func (r *Rule) Init(nParams int) {
	r.ParamProfileTransitions = make([]Transitions, nParams)
	r.SpecialProfileTransitions = make([]Transitions, nParams)
}

type NodeType int

const (
	NodeAndOp = iota
 
	NodeOrOp
 
	NodeNotOp
 
   NodeXorOp

	NodeTerminal
	
	NodeTypeN
)
//go:generate stringer -type=NodeType

var Kit_NodeType = kit.Enums.AddEnum(NodeTypeN, NotBitFlag, nil)

type Node struct {
	Type 
	Child1 *Node // all but terminal
	Child2 *Node // for and, or, xor
	Category *Category // only for terminal node
	wildCard bool // only for terminal node
}

func NewNode(type NodeType, c1, c2 *Node) *Node {
	ex := &BoolAndExpr{}
	ex.Child1 = c1
	ex.Child2 = c2
	return ex
}

func (nd *Node) Eval(posture *Posture) (result bool, err error) {
	switch nd.Type {
		
	case NodeAndOp:
		if nd.Child1 == nil || nd.Child2 == nil {
			err = "Eval error: One or more of nodes children were nil"
		}
		r1 := nd.Child1.Eval(posture)
		r2 := nd.Child2.Eval(posture)
		return r1 && r2, nil

	case NodeOrOp:
		if nd.Child1 == nil || nd.Child2 == nil {
			err = "Eval error: One or more of nodes children were nil"
		}
		r1 := nd.Child1.Eval(posture)
		r2 := nd.Child2.Eval(posture)
		return r1 || r2, nil
		
	case NodeXorOp:
		if nd.Child1 == nil || nd.Child2 == nil {
			err = "Eval error: One or more of nodes children were nil"
		}
		r1 := nd.Child1.Eval(posture)
		r2 := nd.Child2.Eval(posture)
		return r1 != r2, nil
		
	case NodeNotOp:
		if nd.Child1 == nil {
			err = "Eval error: child1 was nil"
		}
		r1 := nd.Child1.Eval(posture)
		return !r1, nil
		
	case NodeTerminal:
		if posture.IsMemberOfCategory {
			return true, nil
		} else if nd.wildCard {
			return posture.Name == (nd.Category + "\'"), nil
		} else {
			return false, nil
		}
	}
}

