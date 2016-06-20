package selector

import (
	"github.com/op/go-logging"
	"strings"
	"errors"
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
	TokNotIn
	TokHas
	TokLParen
	TokRParen
	TokAnd
	TokOr
)

var whitespace = " \t"

type Token struct {
	Kind  tokenKind
	Value interface{}
}

func Parse(selector string) (Selector, error) {
	log.Debugf("Parsing %#v", selector)
	ands := strings.Split(selector, "&&")
	operands := make([]Selector, 0, len(ands))
	for _, chunk := range ands {
		eqParts := strings.Split(chunk, "==")
		label := strings.Trim(eqParts[0], " ")
		literal := strings.Trim(eqParts[1], " \"'")
		eqNode := LabelEqValueNode{
			LabelName: label,
			Value:     literal,
		}
		operands = append(operands, eqNode)
	}
	if len(operands) == 1 {
		return operands[0], nil
	}
	return AndNode{Operands: operands}, nil
}

func Tokenize(input string) ([]Token, error) {
	tokens := make([]Token, 0)
	for {
		startLen := len(input)
		input = strings.TrimLeft(input, whitespace)
		if len(input) == 0 {
			return tokens, nil
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
			input = input[index+1:]
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
				return nil, errors.New("invalid operator")
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
			if strings.HasPrefix(input, "has(") {
				input = input[4:]
				index := strings.Index(input, ")")
				tokens = append(tokens,
					Token{TokHas, input[:index]})
				input = input[index+1:]
			} else if strings.HasPrefix(input, "not"){
				input = strings.TrimLeft(input[3:], whitespace)
				if input[:2] == "in" && input[2] == ' ' {
					tokens = append(tokens, Token{TokNotIn, nil})
					input = input[2:]
				} else {
					return nil, errors.New("invalid operator")
				}
			} else if len(input) > 2 && input[:2] == "in" && input[2] == ' ' {
				tokens = append(tokens, Token{TokIn, nil})
				input = input[2:]
			} else {
				index := strings.IndexAny(input, whitespace)
				if index == -1 {
					tokens = append(tokens,
						Token{TokLabel, input})
					input = ""
				} else {
					tokens = append(tokens,
						Token{TokLabel, input[:index]})
					input = input[index+1:]
				}
			}
		}
		if len(input) >= startLen {
			log.Panicf("Failed to reduce size of input %#v", input)
		}
	}
}
