package selector

import (
	"github.com/op/go-logging"
	"strings"
)

var log = logging.MustGetLogger("labels.selectors.tests")

func Parse(selector string) (Selector, error) {
	log.Debugf("Parsing %#v", selector)
	ands := strings.Split(selector, "&&")
	operands := make([]Selector, 0, len(ands))
	for _, chunk := range ands {
		eqParts := strings.Split(chunk, "==")
		label := strings.Trim(eqParts[0], " ")
		literal := strings.Trim(eqParts[1], " \"'")
		eqNode := LabelEqValue{
			LabelName: label,
			Value:     literal,
		}
		operands = append(operands, eqNode)
	}
	if len(operands) == 1 {
		return operands[0], nil
	}
	return And{Operands: operands}, nil
}
