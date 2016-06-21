package selector

import (
	"strings"
	"crypto"
	_ "crypto/sha256"
	"encoding/base64"
)

type Selector interface {
	Evaluate(labels map[string]string) bool
	String() string
	UniqueId() string
}

type selectorRoot struct {
	root         node
	cachedString *string
	cachedHash   *string
}

func (sel selectorRoot) Evaluate(labels map[string]string) bool {
	return sel.root.Evaluate(labels)
}

func (sel selectorRoot) String() string {
	if sel.cachedString == nil {
		fragments := sel.root.collectFragments([]string{})
		joined := strings.Join(fragments, "")
		sel.cachedString = &joined
	}
	return *sel.cachedString
}

func (sel selectorRoot) UniqueId() string {
	if sel.cachedHash == nil {
		hash := crypto.SHA224.New()
		hash.Write([]byte(sel.String()))
		hashBytes := hash.Sum(make([]byte, 0, hash.Size()))
		b64hash := base64.RawURLEncoding.EncodeToString(hashBytes)
		sel.cachedHash = &b64hash
	}
	return *sel.cachedHash
}

var _ Selector = (*selectorRoot)(nil)

type node interface {
	Evaluate(labels map[string]string) bool
	collectFragments(fragments []string) []string
}

type LabelEqValueNode struct {
	LabelName string
	Value     string
}

func (node LabelEqValueNode) Evaluate(labels map[string]string) bool {
	if val, ok := labels[node.LabelName]; ok {
		return val == node.Value
	} else {
		return false
	}
}

func (node LabelEqValueNode) collectFragments(fragments []string) []string {
	var quote string
	if strings.Contains(node.Value, `"`) {
		quote = `'`
	}  else {
		quote = `"`
	}
	return append(fragments, node.LabelName, " == ", quote, node.Value, quote)
}

type LabelNeValueNode struct {
	LabelName string
	Value     string
}

func (node LabelNeValueNode) Evaluate(labels map[string]string) bool {
	if val, ok := labels[node.LabelName]; ok {
		return val != node.Value
	} else {
		return true
	}
}

func (node LabelNeValueNode) collectFragments(fragments []string) []string {
	var quote string
	if strings.Contains(node.Value, `"`) {
		quote = `'`
	}  else {
		quote = `"`
	}
	return append(fragments, node.LabelName, " != ", quote, node.Value, quote)
}

type HasNode struct {
	LabelName string
}

func (node HasNode) Evaluate(labels map[string]string) bool {
	if _, ok := labels[node.LabelName]; ok {
		return true
	} else {
		return false
	}
}

func (node HasNode) collectFragments(fragments []string) []string {
	return append(fragments, "has(", node.LabelName, ")")
}

type NotNode struct {
	Operand node
}

func (node NotNode) Evaluate(labels map[string]string) bool {
	return !node.Operand.Evaluate(labels)
}

func (node NotNode) collectFragments(fragments []string) []string {
	fragments = append(fragments, "!")
	return node.Operand.collectFragments(fragments)
}

type AndNode struct {
	Operands []node
}

func (node AndNode) Evaluate(labels map[string]string) bool {
	for _, operand := range node.Operands {
		if !operand.Evaluate(labels) {
			return false
		}
	}
	return true
}

func (node AndNode) collectFragments(fragments []string) []string {
	fragments = append(fragments, "(")
	fragments = node.Operands[0].collectFragments(fragments)
	for _, op := range node.Operands[1:] {
		fragments = append(fragments, " && ")
		fragments = op.collectFragments(fragments)
	}
	fragments = append(fragments, ")")
	return fragments
}

type OrNode struct {
	Operands []node
}

func (node OrNode) Evaluate(labels map[string]string) bool {
	for _, operand := range node.Operands {
		if operand.Evaluate(labels) {
			return true
		}
	}
	return false
}

func (node OrNode) collectFragments(fragments []string) []string {
	fragments = append(fragments, "(")
	fragments = node.Operands[0].collectFragments(fragments)
	for _, op := range node.Operands[1:] {
		fragments = append(fragments, " || ")
		fragments = op.collectFragments(fragments)
	}
	fragments = append(fragments, ")")
	return fragments
}

type AllNode struct {
}

func (node AllNode) Evaluate(labels map[string]string) bool {
	return true
}

func (node AllNode) collectFragments(fragments []string) []string {
	return append(fragments, "all()")
}
