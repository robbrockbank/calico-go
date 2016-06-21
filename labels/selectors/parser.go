package selector

import (
	"errors"
	"github.com/op/go-logging"
	"fmt"
)

var log = logging.MustGetLogger("selector")

// Parse a string representation of a selector expression into a Selector.
func Parse(selector string) (sel Selector, err error) {
	log.Debugf("Parsing %#v", selector)
	tokens, err := Tokenize(selector)
	if err != nil {
		return
	}
	if tokens[0].Kind == TokEof {
		return AllNode{}, nil
	}
	log.Debugf("Tokens %v", tokens)
	// The "||" operator has the lowest precedence so we start with that.
	sel, remTokens, err := parseOrExpression(tokens)
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

// parseOrExpression parses a one or more "&&" terms, separated by "||" operators.
func parseOrExpression(tokens []Token) (sel Selector, remTokens []Token, err error) {
	log.Debugf("Parsing ||s from %v", tokens)
	andNodes := make([]Selector, 0)
	sel, remTokens, err = parseAndExpression(tokens)
	if err != nil {
		return
	}
	andNodes = append(andNodes, sel)
	for {
		switch remTokens[0].Kind {
		case TokOr:
			remTokens = remTokens[1:]
			sel, remTokens, err = parseAndExpression(remTokens)
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

// parseAndExpression parses a one or more operations, separated by "&&" operators.
func parseAndExpression(tokens []Token) (sel Selector, remTokens []Token, err error) {
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

// parseOperations parses a single, possibly negated operation (i.e. ==, !=, has()).
// It also handles calling parseOrExpression recursively for parenthesized expressions.
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
	case TokAll:
		sel = AllNode{}
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
		sel, remTokens, err = parseOrExpression(tokens[1:])
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
