package selector

import (
	"errors"
	"github.com/op/go-logging"
	"strings"
	"fmt"
	"regexp"
)

var log = logging.MustGetLogger("labels.selectors.tests")

type tokenKind uint8

const (
	TokLabel tokenKind = iota + 1
	TokStringLiteral
	TokSetLiteral
	TokEq
	TokNe
	TokIn
	TokNot
	TokNotIn
	TokHas
	TokLParen
	TokRParen
	TokAnd
	TokOr
	TokEof
)

var whitespace = " \t"

type Token struct {
	Kind  tokenKind
	Value interface{}
}

func Parse(selector string) (sel Selector, err error) {
	log.Debugf("Parsing %#v", selector)
	tokens, err := Tokenize(selector)
	if err != nil {
		return
	}
	log.Debugf("Tokens %v", tokens)
	sel, remTokens, err := parseExpression(tokens)
	if err != nil {
		return
	}
	if len(remTokens) != 1 {
		err = errors.New(fmt.Sprint("unexpected content at end of selector ", remTokens))
		sel = nil
		return
	}
	return
}

func parseExpression(tokens []Token) (sel Selector, remTokens []Token, err error) {
	log.Debugf("Parsing expression from %v", tokens)
	sel, remTokens, err = parseDisjunction(tokens)
	if err != nil {
		return
	}
	return
}

func parseDisjunction(tokens []Token) (sel Selector, remTokens []Token, err error) {
	log.Debugf("Parsing ||s from %v", tokens)
	andNodes := make([]Selector, 0)
	sel, remTokens, err = parseConjunction(tokens)
	if err != nil {
		return
	}
	andNodes = append(andNodes, sel)
	for {
		switch remTokens[0].Kind {
		case TokOr:
			remTokens = remTokens[1:]
			sel, remTokens, err = parseConjunction(remTokens)
			if err != nil {
				return
			}
			andNodes = append(andNodes, sel)
		default:
			if len(andNodes) == 1 {
				sel = andNodes[0]
			} else {
				sel = OrNode{andNodes}
			}
			return
		}
	}
}

func parseConjunction(tokens []Token) (sel Selector, remTokens []Token, err error) {
	log.Debugf("Parsing &&s from %v", tokens)
	opNodes := make([]Selector, 0)
	sel, remTokens, err = parseOperation(tokens)
	if err != nil {
		return
	}
	opNodes = append(opNodes, sel)
	for {
		switch remTokens[0].Kind {
		case TokAnd:
			remTokens = remTokens[1:]
			sel, remTokens, err = parseOperation(remTokens)
			if err != nil {
				return
			}
			opNodes = append(opNodes, sel)
		default:
			if len(opNodes) == 1 {
				sel = opNodes[0]
			} else {
				sel = AndNode{opNodes}
			}
			return
		}
	}
}

func parseOperation(tokens []Token) (sel Selector, remTokens []Token, err error) {
	log.Debugf("Parsing op from %v", tokens)
	if len(tokens) == 0 {
		err = errors.New("Unexpected end of string looking for op")
		return
	}
	negated := false
	for {
		if tokens[0].Kind == TokNot {
			negated = !negated
			tokens = tokens[1:]
		} else {
			break
		}
	}

	switch tokens[0].Kind {
	case TokHas:
		sel = HasNode{tokens[0].Value.(string)}
		remTokens = tokens[1:]
	case TokLabel:
		// should have an operator and a literal.
		if len(tokens) < 3 {
			err = errors.New(fmt.Sprint("Unexpected end of string in middle of op", tokens))
			return
		}
		switch tokens[1].Kind {
		case TokEq:
			if tokens[2].Kind == TokStringLiteral {
				sel = LabelEqValueNode{tokens[0].Value.(string), tokens[2].Value.(string)}
				remTokens = tokens[3:]
			} else {
				err = errors.New("Expected string")
			}
		case TokNe:
			if tokens[2].Kind == TokStringLiteral {
				sel = LabelNeValueNode{tokens[0].Value.(string), tokens[2].Value.(string)}
				remTokens = tokens[3:]
			} else {
				err = errors.New("Expected string")
			}
		// TODO in and not in
		default:
			err = errors.New("Expected == or !=")
			return
		}
	case TokLParen:
		sel, remTokens, err = parseExpression(tokens[1:])
		if err != nil {
			return
		}
		if len(remTokens) < 1 || remTokens[0].Kind != TokRParen {
			err = errors.New("Expected )")
			return
		}
		remTokens = remTokens[1:]
	default:
		err = errors.New("Unexpected token")
		return
	}
	if negated && err == nil {
		sel = NotNode{sel}
	}
	return
}

const (
	identifierExpr = `[a-zA-Z_./-][a-zA-Z_./-0-9]*`
	hasExpr = `has\(\s*(` + identifierExpr + `)\s*\)`
	notInExpr = `not\s*in\b`
	inExpr = `in\b`
)

var (
	identifierRegex, _ = regexp.Compile("^" + identifierExpr)
	hasRegex, _ = regexp.Compile("^" + hasExpr)
	notInRegex, _ = regexp.Compile("^" + notInExpr)
	inRegex, _ = regexp.Compile("^" + inExpr)
)

func Tokenize(input string) (tokens []Token, err error) {
	for {
		log.Debug("Remaining input: ", input)
		startLen := len(input)
		input = strings.TrimLeft(input, whitespace)
		if len(input) == 0 {
			tokens = append(tokens, Token{TokEof, nil})
			return
		}
		switch input[0] {
		case '(':
			tokens = append(tokens, Token{TokLParen, nil})
			input = input[1:]
		case ')':
			tokens = append(tokens, Token{TokRParen, nil})
			input = input[1:]
		case '"':
			input = input[1:]
			index := strings.Index(input, `"`)
			if index == -1 {
				return nil, errors.New("unterminated string")
			}
			value := input[0:index]
			tokens = append(tokens, Token{TokStringLiteral, value})
			input = input[index + 1:]
		case '\'':
			input = input[1:]
			index := strings.Index(input, `'`)
			if index == -1 {
				return nil, errors.New("unterminated string")
			}
			value := input[0:index]
			tokens = append(tokens, Token{TokStringLiteral, value})
			input = input[index + 1:]
		case '=':
			if input[1] == '=' {
				tokens = append(tokens, Token{TokEq, nil})
				input = input[2:]
			} else {
				return nil, errors.New("invalid operator")
			}
		case '!':
			if input[1] == '=' {
				tokens = append(tokens, Token{TokNe, nil})
				input = input[2:]
			} else {
				tokens = append(tokens, Token{TokNot, nil})
				input = input[1:]
			}
		case '&':
			if input[1] == '&' {
				tokens = append(tokens, Token{TokAnd, nil})
				input = input[2:]
			} else {
				return nil, errors.New("invalid operator")
			}
		case '|':
			if input[1] == '|' {
				tokens = append(tokens, Token{TokOr, nil})
				input = input[2:]
			} else {
				return nil, errors.New("invalid operator")
			}
		default:
			if idxs := hasRegex.FindStringSubmatchIndex(input); idxs != nil {
				// Found "has(label)"
				wholeMatchEnd := idxs[1]
				labelNameMatchStart := idxs[2]
				labelNameMatchEnd := idxs[3]
				labelName := input[labelNameMatchStart:labelNameMatchEnd]
				tokens = append(tokens, Token{TokHas, labelName})
				input = input[wholeMatchEnd:]
			} else if idxs := notInRegex.FindStringIndex(input); idxs != nil {
				// Found "not in"
				tokens = append(tokens, Token{TokNotIn, nil})
				input = input[idxs[1]:]
			} else if idxs := inRegex.FindStringIndex(input); idxs != nil {
				// Found "in"
				tokens = append(tokens, Token{TokIn, nil})
				input = input[idxs[1]:]
			} else if idxs := identifierRegex.FindStringIndex(input); idxs != nil {
				// Found "label"
				endIndex := idxs[1]
				identifier := input[:endIndex]
				log.Debug("Identifier ", identifier)
				tokens = append(tokens, Token{TokLabel, identifier})
				input = input[endIndex:]
			} else {
				err = errors.New("unexpected characters")
				return
			}
		}
		if len(input) >= startLen {
			err = errors.New("infinite loop detected in tokenizer")
			return
		}
	}
}
