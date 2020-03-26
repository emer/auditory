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

const         addChar = '+'
const         subChar = '-'
const        multChar = '*'
const         divChar = '/'
const  rightParenChar = ')'
const   leftParenChar = '('

type Equation struct {
	Name string
	Comment string
	FormulaName string
	FormulaRoot *FormulaNode
}

type EqGroup struct {
	Name string
	Equations []Equation
}

type SymbolType int

const (
	SymInvalid = iota
	 
	SymAdd
	 
	SymSub
	 
	SymMult
	
	SymDiv
	
	SymRtParen
	
	SymLftParen
	
	SymTypeN
)
//go:generate stringer -type=SymbolType

var Kit_SymbolType = kit.Enums.AddEnum(SymTypeN, NotBitFlag, nil)

type FormulaNode struct {
	Type 
	Child1 *FormulaNode
	Child2 *FormulaNode
	Value float64
	Symbol Code
}

type FormulaParse struct {
	
}
public:
	FormulaNodeParser(const std::string& s)
				: formulaSymbolMap_(formulaSymbol.codeMap)
				, s_(GS::Text::trim(s))
				, pos_(0)
				, symbolType_(SYMBOL_TYPE_INVALID) {
		if (s_.empty()) {
			THROW_EXCEPTION(GS::TRMControlModelException, "Formula expression parser error: Empty string.")
		}
		nextSymbol();
	}

	FormulaNode_ptr parse();
private:
	enum SymbolType {
		SYMBOL_TYPE_INVALID,
		SYMBOL_TYPE_ADD,
		SYMBOL_TYPE_SUB,
		SYMBOL_TYPE_MULT,
		SYMBOL_TYPE_DIV,
		SYMBOL_TYPE_RIGHT_PAREN,
		SYMBOL_TYPE_LEFT_PAREN,
		SYMBOL_TYPE_STRING
	};

	void throwException(const char* errorDescription) const;
	template<typename T> void throwException(const char* errorDescription, const T& complement) const;
	bool finished() const {
		return pos_ >= s_.size();
	}
	void nextSymbol();
	FormulaNode_ptr parseFactor();
	FormulaNode_ptr parseTerm();
	FormulaNode_ptr parseExpression();
	void skipSpaces();

	static bool isSeparator(char c) {
		switch (c) {
		case rightParenChar: return true;
		case leftParenChar:  return true;
		default:             return std::isspace(c) != 0;
		}
	}

	const FormulaSymbol::CodeMap& formulaSymbolMap_;
	const std::string s_;
	std::string::size_type pos_;
	std::string symbol_;
	SymbolType symbolType_;
};

void
FormulaNodeParser::throwException(const char* errorDescription) const
{
	THROW_EXCEPTION(GS::TRMControlModelException, "Formula expression parser error: "
				<< errorDescription
				<< " at position " << (pos_ - symbol_.size()) << " of string [" << s_ << "].");
}

template<typename T>
void
FormulaNodeParser::throwException(const char* errorDescription, const T& complement) const
{
	THROW_EXCEPTION(GS::TRMControlModelException, "Formula expression parser error: "
				<< errorDescription << complement
				<< " at position " << (pos_ - symbol_.size()) << " of string [" << s_ << "].");
}

void
FormulaNodeParser::skipSpaces()
{
	while (!finished() && std::isspace(s_[pos_])) ++pos_;
}

void
FormulaNodeParser::nextSymbol()
{
	skipSpaces();

	symbol_.resize(0);

	if (finished()) {
		symbolType_ = SYMBOL_TYPE_INVALID;
		return;
	}

	char c = s_[pos_++];
	symbol_ = c;
	switch (c) {
	case addChar:
		symbolType_ = SYMBOL_TYPE_ADD;
		break;
	case subChar:
		symbolType_ = SYMBOL_TYPE_SUB;
		break;
	case multChar:
		symbolType_ = SYMBOL_TYPE_MULT;
		break;
	case divChar:
		symbolType_ = SYMBOL_TYPE_DIV;
		break;
	case rightParenChar:
		symbolType_ = SYMBOL_TYPE_RIGHT_PAREN;
		break;
	case leftParenChar:
		symbolType_ = SYMBOL_TYPE_LEFT_PAREN;
		break;
	default:
		symbolType_ = SYMBOL_TYPE_STRING;
		while ( !finished() && !isSeparator(c = s_[pos_]) ) {
			symbol_ += c;
			++pos_;
		}
	}
}

/*******************************************************************************
 * FACTOR -> "(" EXPRESSION ")" | SYMBOL | CONST | ADD_OP FACTOR
 */
FormulaNode_ptr
FormulaNodeParser::parseFactor()
{
	switch (symbolType_) {
	case SYMBOL_TYPE_LEFT_PAREN: // (expression)
	{
		nextSymbol();
		FormulaNode_ptr res(parseExpression());
		if (symbolType_ != SYMBOL_TYPE_RIGHT_PAREN) {
			throwException("Right parenthesis not found");
		}
		nextSymbol();
		return res;
	}
	case SYMBOL_TYPE_ADD: // unary plus
		nextSymbol();
		return parseFactor();
	case SYMBOL_TYPE_SUB: // unary minus
		nextSymbol();
		return FormulaNode_ptr(new FormulaMinusUnaryOp(parseFactor()));
	case SYMBOL_TYPE_STRING: // const / symbol
	{
		std::string symbolTmp = symbol_;
		nextSymbol();
		FormulaSymbol::CodeMap::const_iterator iter = formulaSymbolMap_.find(symbolTmp);
		if (iter == formulaSymbolMap_.end()) {
			// It's not a symbol.
			return FormulaNode_ptr(new FormulaConst(GS::Text::parseString<float>(symbolTmp)));
		} else {
			return FormulaNode_ptr(new FormulaSymbolValue(iter->second));
		}
	}
	case SYMBOL_TYPE_RIGHT_PAREN:
		throwException("Unexpected symbol: ", rightParenChar);
	case SYMBOL_TYPE_MULT:
		throwException("Unexpected symbol: ", multChar);
	case SYMBOL_TYPE_DIV:
		throwException("Unexpected symbol: ", divChar);
	default:
		throwException("Invalid symbol");
	}
	return FormulaNode_ptr(); // unreachable
}

/*******************************************************************************
 * TERM -> FACTOR { MULT_OP FACTOR }
 */
FormulaNode_ptr FormulaNodeParser::parseTerm()
{
	FormulaNode_ptr term1 = parseFactor();

	SymbolType type = symbolType_;
	while (type == SYMBOL_TYPE_MULT || type == SYMBOL_TYPE_DIV) {
		nextSymbol();
		FormulaNode_ptr term2 = parseFactor();
		FormulaNode_ptr expr;
		if (type == SYMBOL_TYPE_MULT) {
			expr.reset(new FormulaMultBinaryOp(std::move(term1), std::move(term2)));
		} else {
			expr.reset(new FormulaDivBinaryOp(std::move(term1), std::move(term2)));
		}
		term1.swap(expr);
		type = symbolType_;
	}

	return term1;
}

/*******************************************************************************
 * EXPRESSION -> TERM { ADD_OP TERM }
 */
FormulaNode_ptr
FormulaNodeParser::parseExpression()
{
	FormulaNode_ptr term1 = parseTerm();

	SymbolType type = symbolType_;
	while (type == SYMBOL_TYPE_ADD || type == SYMBOL_TYPE_SUB) {

		nextSymbol();
		FormulaNode_ptr term2 = parseTerm();

		FormulaNode_ptr expr;
		if (type == SYMBOL_TYPE_ADD) {
			expr.reset(new FormulaAddBinaryOp(std::move(term1), std::move(term2)));
		} else {
			expr.reset(new FormulaSubBinaryOp(std::move(term1), std::move(term2)));
		}

		term1 = std::move(expr);
		type = symbolType_;
	}

	return term1;
}

FormulaNode_ptr
FormulaNodeParser::parse()
{
	FormulaNode_ptr formulaRoot = parseExpression();
	if (symbolType_ != SYMBOL_TYPE_INVALID) { // there is a symbol available
		throwException("Invalid text");
	}
	return formulaRoot;
}

} /* namespace */

//==============================================================================

namespace GS {
namespace TRMControlModel {

float
FormulaMinusUnaryOp::eval(const FormulaSymbolList& symbolList) const
{
	return -(child_->eval(symbolList));
}

void
FormulaMinusUnaryOp::print(std::ostream& out, int level) const
{
	std::string prefix(level * 8, ' ');
	out << prefix << "- [\n";

	child_->print(out, level + 1);

	out << prefix << "]" << std::endl;
}

float
FormulaAddBinaryOp::eval(const FormulaSymbolList& symbolList) const
{
	return child1_->eval(symbolList) + child2_->eval(symbolList);
}

void
FormulaAddBinaryOp::print(std::ostream& out, int level) const
{
	std::string prefix(level * 8, ' ');
	out << prefix << "+ [\n";

	child1_->print(out, level + 1);
	child2_->print(out, level + 1);

	out << prefix << "]" << std::endl;
}

float
FormulaSubBinaryOp::eval(const FormulaSymbolList& symbolList) const
{
	return child1_->eval(symbolList) - child2_->eval(symbolList);
}

void
FormulaSubBinaryOp::print(std::ostream& out, int level) const
{
	std::string prefix(level * 8, ' ');
	out << prefix << "- [\n";

	child1_->print(out, level + 1);
	child2_->print(out, level + 1);

	out << prefix << "]" << std::endl;
}

float
FormulaMultBinaryOp::eval(const FormulaSymbolList& symbolList) const
{
	return child1_->eval(symbolList) * child2_->eval(symbolList);
}

void
FormulaMultBinaryOp::print(std::ostream& out, int level) const
{
	std::string prefix(level * 8, ' ');
	out << prefix << "* [\n";

	child1_->print(out, level + 1);
	child2_->print(out, level + 1);

	out << prefix << "]" << std::endl;
}

float
FormulaDivBinaryOp::eval(const FormulaSymbolList& symbolList) const
{
	return child1_->eval(symbolList) / child2_->eval(symbolList);
}

void
FormulaDivBinaryOp::print(std::ostream& out, int level) const
{
	std::string prefix(level * 8, ' ');
	out << prefix << "/ [\n";

	child1_->print(out, level + 1);
	child2_->print(out, level + 1);

	out << prefix << "]" << std::endl;
}

float
FormulaConst::eval(const FormulaSymbolList& /*symbolList*/) const
{
	return value_;
}

void
FormulaConst::print(std::ostream& out, int level) const
{
	out << std::string(level * 8, ' ') << "const=" << value_ << std::endl;
}

float
FormulaSymbolValue::eval(const FormulaSymbolList& symbolList) const
{
	return symbolList[symbol_];
}

void
FormulaSymbolValue::print(std::ostream& out, int level) const
{
	out << std::string(level * 8, ' ') << "symbol=" << symbol_ << std::endl;
}

void
Equation::setFormula(const std::string& formula)
{
	FormulaNodeParser p(formula);
	FormulaNode_ptr tempFormulaRoot = p.parse();

	formula_ = formula;
	std::swap(tempFormulaRoot, formulaRoot_);
}

float
Equation::evalFormula(const FormulaSymbolList& symbolList) const
{
	if (!formulaRoot_) {
		THROW_EXCEPTION(InvalidStateException, "Empty formula.");
	}

	return formulaRoot_->eval(symbolList);
}

std::ostream&
operator<<(std::ostream& out, const Equation& equation)
{
	if (equation.formulaRoot_) {
		equation.formulaRoot_->print(out);
	}
	return out;
}

} /* namespace TRMControlModel */
} /* namespace GS */
