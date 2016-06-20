package selector

type Selector interface {
	Evaluate(labels map[string]string) bool
}

type Label struct {
	LabelName string
}

type LabelEqValue struct {
	LabelName string
	Value string
}

func (node LabelEqValue) Evaluate(labels map[string]string) bool {
	if val, ok := labels[node.LabelName]; ok {
		return val == node.Value
	} else {
		return false
	}
}

type And struct {
	Operands []Selector
}

func (node And) Evaluate(labels map[string]string) bool {
	for _, operand := range node.Operands {
		if !operand.Evaluate(labels)  {
			return false
		}
	}
	return true
}
