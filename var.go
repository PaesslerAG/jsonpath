package jsonpath

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/PaesslerAG/gval"
)

type variableWildcard struct{}

type variableRecursiveDescent struct{}

type variableRange struct {
	Min, Max, Step int
}

type variableChild struct {
	Key interface{}
}

type VariableVisitor interface {
	VisitWildcard(ctx context.Context, parameter interface{}, next func(context.Context, []PathValue) error) error
	VisitRecursiveDescent(ctx context.Context, parameter interface{}, next func(context.Context, []PathValue) error) error
	VisitRange(ctx context.Context, parameter interface{}, min, max, step int, next func(context.Context, []PathValue) error) error
	VisitChild(ctx context.Context, parameter interface{}, key interface{}, next func(context.Context, PathValue) error) error
}

type VariableVisitorFuncs struct {
	VisitWildcardFunc         func(ctx context.Context, parameter interface{}, next func(context.Context, []PathValue) error) error
	VisitRecursiveDescentFunc func(ctx context.Context, parameter interface{}, next func(context.Context, []PathValue) error) error
	VisitRangeFunc            func(ctx context.Context, parameter interface{}, min, max, step int, next func(context.Context, []PathValue) error) error
	VisitChildFunc            func(ctx context.Context, parameter interface{}, key interface{}, next func(context.Context, PathValue) error) error
}

var _ VariableVisitor = VariableVisitorFuncs{}

func (vf VariableVisitorFuncs) VisitWildcard(c context.Context, v interface{}, next func(context.Context, []PathValue) error) error {
	if vf.VisitWildcardFunc != nil {
		return vf.VisitWildcardFunc(c, v, next)
	}

	var items []PathValue

	appender := func(c context.Context, kv PathValue) error {
		items = append(items, kv)
		return nil
	}

	switch vt := v.(type) {
	case []interface{}:
		for i := range vt {
			if err := vf.VisitChild(c, v, i, appender); err != nil {
				return fmt.Errorf("could not parse index %d: %v", i, err)
			}
		}
	case map[string]interface{}:
		for k := range vt {
			if err := vf.VisitChild(c, v, k, appender); err != nil {
				return fmt.Errorf("could not parse key %q: %v", k, err)
			}
		}
	}

	return next(c, items)
}

func (vf VariableVisitorFuncs) VisitRecursiveDescent(c context.Context, v interface{}, next func(context.Context, []PathValue) error) error {
	if vf.VisitRecursiveDescentFunc != nil {
		return vf.VisitRecursiveDescentFunc(c, v, next)
	}

	items := []PathValue{
		{Value: v},
	}

	var appender func(prefix ...string) func(context.Context, []PathValue) error
	appender = func(prefix ...string) func(context.Context, []PathValue) error {
		return func(c context.Context, vs []PathValue) error {
			for _, v := range vs {
				item := PathValue{
					Path:  append(append([]string{}, prefix...), v.Path...),
					Value: v.Value,
				}

				items = append(items, item)
				if err := vf.VisitWildcard(c, v.Value, appender(item.Path...)); err != nil {
					return fmt.Errorf("error resolving path %q: %v", strings.Join(v.Path, "."), err)
				}
			}

			return nil
		}
	}
	vf.VisitWildcard(c, v, appender())

	return next(c, items)
}

func (vf VariableVisitorFuncs) VisitRange(c context.Context, v interface{}, min, max, step int, next func(context.Context, []PathValue) error) error {
	if vf.VisitRangeFunc != nil {
		return vf.VisitRangeFunc(c, v, min, max, step, next)
	}

	if min > max {
		return nil
	}

	var items []PathValue

	switch vt := v.(type) {
	case []interface{}:
		n := len(vt)
		min = negmax(min, n)
		max = negmax(max, n)
	case map[string]interface{}:
		// Ranging over a map is explicitly not supported.
		return next(c, items)
	default:
		// Otherwise we hope the variable selector can do its job.
	}

	if step == 0 {
		step = 1
	}

	appender := func(c context.Context, kv PathValue) error {
		items = append(items, kv)
		return nil
	}

	if step > 0 {
		for i := min; i < max; i += step {
			if err := vf.VisitChild(c, v, i, appender); err != nil {
				return fmt.Errorf("could not parse index %d: %v", i, err)
			}
		}
	} else {
		for i := max - 1; i >= min; i += step {
			if err := vf.VisitChild(c, v, i, appender); err != nil {
				return fmt.Errorf("could not parse index %d: %v", i, err)
			}
		}
	}

	return next(c, items)
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

func (vf VariableVisitorFuncs) VisitChild(c context.Context, v interface{}, key interface{}, next func(context.Context, PathValue) error) error {
	if vf.VisitChildFunc != nil {
		return vf.VisitChildFunc(c, v, key, next)
	}

	var kv PathValue

	switch vt := v.(type) {
	case []interface{}:
		var i int
		switch kt := key.(type) {
		case string:
			ki, err := strconv.ParseInt(kt, 10, 32)
			if err != nil {
				return fmt.Errorf("unexpected string index %q for slice, must be convertible to int: %v", kt, err)
			}

			i = int(ki)
		case int, int8, int16, int32, int64:
			i = int(reflect.ValueOf(kt).Int())
		case uint, uint8, uint16, uint32, uint64:
			i = int(reflect.ValueOf(kt).Uint())
		case float32, float64:
			i = int(reflect.ValueOf(kt).Float())
		default:
			return fmt.Errorf("unexpected index type %T for slice", kt)
		}

		if i < 0 || i >= len(vt) {
			return fmt.Errorf("index %d out of bounds", i)
		}

		kv.Path = []string{strconv.Itoa(i)}
		kv.Value = vt[i]
	case map[string]interface{}:
		var k string
		switch kt := key.(type) {
		case string:
			k = kt
		case int, int8, int16, int32, int64:
			k = strconv.FormatInt(reflect.ValueOf(kt).Int(), 64)
		case uint, uint8, uint16, uint32, uint64:
			k = strconv.FormatUint(reflect.ValueOf(kt).Uint(), 64)
		case float32, float64:
			k = strconv.FormatFloat(reflect.ValueOf(kt).Float(), 'f', -1, 64)
		default:
			return fmt.Errorf("unexpected key type %T for map", kt)
		}

		r, ok := vt[k]
		if !ok {
			return fmt.Errorf("unknown key %s", k)
		}

		kv.Path = []string{k}
		kv.Value = r
	default:
		return fmt.Errorf("unsupported value type %T for select, expected map[string]interface{} or []interface{}", v)
	}

	return next(c, kv)
}

func VariableSelector(visitor VariableVisitor) func(path gval.Evaluables) gval.Evaluable {
	return func(path gval.Evaluables) gval.Evaluable {
		return func(c context.Context, v interface{}) (r interface{}, err error) {
			var next func(c context.Context, rest gval.Evaluables, v interface{}) (values, bool, error)
			next = func(c context.Context, rest gval.Evaluables, v interface{}) (values, bool, error) {
				if len(rest) == 0 {
					return nil, false, nil
				}

				t, err := rest[0](c, v)
				if err != nil {
					return nil, false, err
				}

				var r values
				combine := func(c context.Context, vs values) error {
					vs, err := vs.flatMap(func(v value) (values, error) {
						r, ok, err := next(c, rest[1:], v.value)
						if err != nil {
							return nil, err
						} else if !ok {
							return v, nil
						}

						return v.prefix(r), nil
					})
					if err != nil {
						return err
					}

					vs.concat(&r)
					return nil
				}

				switch tt := t.(type) {
				case variableWildcard:
					err = visitor.VisitWildcard(c, v, func(c context.Context, pvs []PathValue) error {
						return combine(c, pathValueSlice(pvs))
					})
				case variableRecursiveDescent:
					err = visitor.VisitRecursiveDescent(c, v, func(c context.Context, pvs []PathValue) error {
						return combine(c, pathValueSlice(pvs))
					})
				case variableRange:
					err = visitor.VisitRange(c, v, tt.Min, tt.Max, tt.Step, func(c context.Context, pvs []PathValue) error {
						return combine(c, pathValueSlice(pvs))
					})
				case variableChild:
					err = visitor.VisitChild(c, v, tt.Key, func(c context.Context, pv PathValue) error {
						return combine(c, value{wildcards: [][]string{pv.Path}, value: pv.Value})
					})
				default:
					err = fmt.Errorf("unknown variable type %T", t)
				}
				return r, err == nil, err
			}
			r, _, err = next(c, path, v)
			return
		}
	}
}

func pathValueSlice(pvs []PathValue) valueSlice {
	vs := make(valueSlice, len(pvs))
	for i, pv := range pvs {
		vs[i] = value{wildcards: [][]string{pv.Path}, value: pv.Value}
	}
	return vs
}

func ChildVariableSelector(fn func(ctx context.Context, parameter interface{}, key interface{}, next func(context.Context, PathValue) error) error) func(path gval.Evaluables) gval.Evaluable {
	return VariableSelector(VariableVisitorFuncs{
		VisitChildFunc: fn,
	})
}

func DefaultVariableVisitor() VariableVisitor {
	return VariableVisitorFuncs{}
}

func DefaultVariableSelector() func(path gval.Evaluables) gval.Evaluable {
	return VariableSelector(DefaultVariableVisitor())
}
