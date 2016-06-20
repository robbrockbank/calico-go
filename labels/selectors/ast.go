package selector

type Selector interface {
	Evaluate(labels map[string]string) bool
}

type LabelNode struct {
	LabelName string
}

type LabelEqValueNode struct {
	LabelName string
	Value string
}

func (node LabelEqValueNode) Evaluate(labels map[string]string) bool {
	if val, ok := labels[node.LabelName]; ok {
		return val == node.Value
	} else {
		return false
	}
}

type AndNode struct {
	Operands []Selector
}

func (node AndNode) Evaluate(labels map[string]string) bool {
	for _, operand := range node.Operands {
		if !operand.Evaluate(labels)  {
			return false
		}
	}
	return true
}
