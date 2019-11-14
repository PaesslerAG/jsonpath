package jsonpath

import (
	"context"
	"fmt"

	"github.com/PaesslerAG/gval"
)

func variableSelector(path gval.Evaluables) gval.Evaluable {
	return func(c context.Context, v interface{}) (interface{}, error) {
		for _, key := range path {
			switch o := v.(type) {
			case []interface{}:
				i, err := key.EvalInt(c, v)
				if err != nil {
					return nil, fmt.Errorf("could not select value, invalid key: %s", err)
				}
				if i < 0 || i >= len(o) {
					return nil, fmt.Errorf("index %d out of bounds", i)
				}

				v = o[i]
			case map[string]interface{}:
				k, err := key.EvalString(c, v)
				if err != nil {
					return nil, fmt.Errorf("could not select value, invalid key: %s", err)
				}

				r, ok := o[k]
				if !ok {
					return nil, fmt.Errorf("unknown key %s", k)
				}

				v = r
			default:
				return nil, fmt.Errorf("unsupported value type %T for select, expected map[string]interface{} or []interface{}", o)
			}
		}

		return v, nil
	}
}

func VariableSelector() func(path gval.Evaluables) gval.Evaluable {
	return variableSelector
}

type variableGetter struct {
	p *gval.Parser
}

func (vg variableGetter) ForEvaluables(c context.Context, keys gval.Evaluables, parameter interface{}) (interface{}, error) {
	return vg.p.Var(keys...)(c, parameter)
}

func (vg variableGetter) ForConst(c context.Context, keys []interface{}, parameter interface{}) (interface{}, error) {
	var eks gval.Evaluables
	switch len(keys) {
	case 1:
		eks = gval.Evaluables{vg.p.Const(keys[0])}
	default:
		eks = make(gval.Evaluables, len(keys))
		for i, k := range keys {
			eks[i] = vg.p.Const(k)
		}
	}

	return vg.ForEvaluables(c, eks, parameter)
}
