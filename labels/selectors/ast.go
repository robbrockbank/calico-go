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

type LabelNeValueNode struct {
	LabelName string
	Value string
}

func (node LabelNeValueNode) Evaluate(labels map[string]string) bool {
	if val, ok := labels[node.LabelName]; ok {
		return val != node.Value
	} else {
		return true
	}
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

type NotNode struct {
	Operand Selector
}

func (node NotNode) Evaluate(labels map[string]string) bool {
	return !node.Operand.Evaluate(labels)
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

type OrNode struct {
	Operands []Selector
}

func (node OrNode) Evaluate(labels map[string]string) bool {
	for _, operand := range node.Operands {
		if operand.Evaluate(labels)  {
			return true
		}
	}
	return false
}

type AllNode struct {
}

func (node AllNode) Evaluate(labels map[string]string) bool {
	return true
}
