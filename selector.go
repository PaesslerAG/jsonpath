package jsonpath

import (
	"context"
	"fmt"
	"strconv"

	"github.com/PaesslerAG/gval"
)

//plainSelector evaluate exactly one result
type plainSelector func(c context.Context, r, v interface{}) (interface{}, error)

//ambiguousSelector evaluate wildcard
type ambiguousSelector func(c context.Context, r, v interface{}, match ambiguousMatcher)

//@
func currentElementSelector() plainSelector {
	return func(c context.Context, r, v interface{}) (interface{}, error) {
		return c.Value(currentElement{}), nil
	}
}

type currentElement struct{}

func currentContext(c context.Context, v interface{}) context.Context {
	return context.WithValue(c, currentElement{}, v)
}

//.x, [x]
func directSelector(gv variableGetter, key gval.Evaluable) plainSelector {
	return func(c context.Context, r, v interface{}) (interface{}, error) {
		e, _, err := selectValue(c, gv, key, r, v)
		if err != nil {
			return nil, err
		}

		return e, nil
	}
}

// * / [*]
func starSelector(gv variableGetter) ambiguousSelector {
	return func(c context.Context, r, v interface{}, match ambiguousMatcher) {
		visitAll(c, gv, v, func(key string, val interface{}) { match(key, val) })
	}
}

// [x, ...]
func multiSelector(gv variableGetter, keys []gval.Evaluable) ambiguousSelector {
	if len(keys) == 0 {
		return starSelector(gv)
	}
	return func(c context.Context, r, v interface{}, match ambiguousMatcher) {
		for _, k := range keys {
			e, wildcard, err := selectValue(c, gv, k, r, v)
			if err != nil {
				continue
			}
			match(wildcard, e)
		}
	}
}

func selectValue(c context.Context, gv variableGetter, key gval.Evaluable, r, v interface{}) (value interface{}, jkey string, err error) {
	c = currentContext(c, v)

	ki, err := key.EvalString(c, r)
	if err != nil {
		return nil, "", fmt.Errorf("could not select value, invalid key: %s", err)
	}

	vi, err := gv.ForEvaluables(c, gval.Evaluables{key}, v)
	if err != nil {
		return nil, "", err
	}

	return vi, ki, nil
}

//..
func mapperSelector(gv variableGetter) (mapper ambiguousSelector) {
	mapper = func(c context.Context, r, v interface{}, match ambiguousMatcher) {
		match([]interface{}{}, v)
		visitAll(c, gv, v, func(wildcard string, v interface{}) {
			mapper(c, r, v, func(key interface{}, v interface{}) {
				match(append([]interface{}{wildcard}, key.([]interface{})...), v)
			})
		})
	}
	return
}

func visitAll(c context.Context, gv variableGetter, v interface{}, visit func(key string, v interface{})) {
	switch v := v.(type) {
	case []interface{}:
		for i := range v {
			e, err := gv.ForConst(c, []interface{}{i}, v)
			if err != nil {
				continue
			}

			k := strconv.Itoa(i)
			visit(k, e)
		}
	case map[string]interface{}:
		for k := range v {
			e, err := gv.ForConst(c, []interface{}{k}, v)
			if err != nil {
				continue
			}

			visit(k, e)
		}
	}
}

//[? ]
func filterSelector(gv variableGetter, filter gval.Evaluable) ambiguousSelector {
	return func(c context.Context, r, v interface{}, match ambiguousMatcher) {
		visitAll(c, gv, v, func(wildcard string, v interface{}) {
			condition, err := filter.EvalBool(currentContext(c, v), r)
			if err != nil {
				return
			}
			if condition {
				match(wildcard, v)
			}
		})
	}
}

//[::]
func rangeSelector(gv variableGetter, min, max, step gval.Evaluable) ambiguousSelector {
	return func(c context.Context, r, v interface{}, match ambiguousMatcher) {
		c = currentContext(c, v)

		min, err := min.EvalInt(c, r)
		if err != nil {
			return
		}
		max, err := max.EvalInt(c, r)
		if err != nil {
			return
		}
		step, err := step.EvalInt(c, r)
		if err != nil {
			return
		}

		if min > max {
			return
		}

		switch vt := v.(type) {
		case []interface{}:
			n := len(vt)
			min = negmax(min, n)
			max = negmax(max, n)
		case map[string]interface{}:
			// Ranging over a map is explicitly not supported.
			return
		default:
			// Otherwise we hope the variable selector can do its job.
		}

		if step == 0 {
			step = 1
		}

		if step > 0 {
			for i := min; i < max; i += step {
				e, err := gv.ForConst(c, []interface{}{i}, v)
				if err != nil {
					continue
				}

				match(strconv.Itoa(i), e)
			}
		} else {
			for i := max - 1; i >= min; i += step {
				e, err := gv.ForConst(c, []interface{}{i}, v)
				if err != nil {
					continue
				}

				match(strconv.Itoa(i), e)
			}
		}

	}
}

func negmax(n, max int) int {
	if n < 0 {
		n = max + n
		if n < 0 {
			n = 0
		}
	} else if n > max {
		return max
	}
	return n
}

// ()
func newScript(script gval.Evaluable) plainSelector {
	return func(c context.Context, r, v interface{}) (interface{}, error) {
		return script(currentContext(c, v), r)
	}
}
