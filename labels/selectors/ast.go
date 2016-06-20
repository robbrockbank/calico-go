package selector

import "github.com/projectcalico/calico-go/labels"

type Selector interface {
	Evaluate(labels labels.Labels) bool
}

type Label struct {
	LabelName string
}

type LabelEqValue struct {
	LabelName string
	Value string
}

func (node LabelEqValue) Evaluate(labels labels.Labels) bool {
	if val, ok := labels[node.LabelName]; ok {
		return val == node.Value
	} else {
		return false
	}
}

type And struct {
	Operands []Selector
}

func (node And) Evaluate(labels labels.Labels) bool {
	for _, operand := range node.Operands {
		if !operand.Evaluate(labels)  {
			return false
		}
	}
	return true
}
