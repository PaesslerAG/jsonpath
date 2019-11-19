package jsonpath

import (
	"context"

	"github.com/PaesslerAG/gval"
)

type PathValue struct {
	Path  []string
	Value interface{}
}

// values allows us to represent a single value or multiple values under a
// common interface.
type values interface {
	// get retrieves the value(s) as an interface.
	get() interface{}

	// concat appends this value to the given destination, creating it if
	// necessary.
	concat(dest *values)

	// flatMap iterates over all values encapsulated by this value and returns a new
	// values with the result.
	flatMap(fn func(value) (values, error)) (values, error)
}

type value struct {
	wildcards [][]string
	value     interface{}
}

var _ values = value{}

func (v value) get() interface{} { return v.value }

func (v value) concat(dest *values) {
	if *dest == nil {
		*dest = v
		return
	}

	switch r := (*dest).(type) {
	case value:
		*dest = valueSlice{r, v}
	case valueSlice:
		*dest = append(r, v)
	}
}

func (v value) flatMap(fn func(value) (values, error)) (values, error) {
	return fn(v)
}

func (v value) prefix(children values) values {
	if len(v.wildcards) == 0 {
		return children
	}

	vs, _ := children.flatMap(func(nv value) (values, error) {
		nv.wildcards = append(append([][]string{}, v.wildcards...), nv.wildcards...)
		return nv, nil
	})
	return vs
}

type valueSlice []value

var _ values = valueSlice{}

func (vs valueSlice) get() interface{} {
	r := make([]interface{}, len(vs))
	for i, v := range vs {
		r[i] = v.get()
	}
	return r
}

func (vs valueSlice) concat(dest *values) {
	if *dest == nil {
		// Force initialization to a slice.
		*dest = valueSlice{}
	}

	for _, v := range vs {
		v.concat(dest)
	}
}

func (vs valueSlice) flatMap(fn func(value) (values, error)) (values, error) {
	var nvs values = valueSlice{}
	for _, v := range vs {
		nv, err := fn(v)
		if err != nil {
			return nil, err
		} else if nv == nil {
			continue
		}

		nv.concat(&nvs)
	}

	return nvs, nil
}

func eachValue(vs values, fn func(value) error) error {
	_, err := vs.flatMap(func(v value) (values, error) {
		if err := fn(v); err != nil {
			return nil, err
		}
		return v, nil
	})
	return err
}

type selectorMode int

const (
	selectorKeepErrors selectorMode = 1 + iota
	selectorDropErrors
)

func (m selectorMode) coalesce(nm selectorMode) selectorMode {
	if m == selectorDropErrors {
		return m
	}
	return nm
}

type pathSelector struct {
	fn   selector
	mode selectorMode
}

type path struct {
	root      gval.Evaluable
	mode      selectorMode
	selectors []pathSelector
}

func (p *path) appendSelector(fn selector, mode selectorMode) {
	p.selectors = append(p.selectors, pathSelector{fn, mode})
}

func (p *path) reduce(c context.Context, parameter interface{}) (values, error) {
	rv, err := p.root(c, parameter)
	if err != nil {
		return nil, err
	}

	root := value{value: rv}

	var apply func(vs values, mode selectorMode, rest []pathSelector) (values, error)
	apply = func(vs values, mode selectorMode, rest []pathSelector) (values, error) {
		if len(rest) == 0 {
			return vs, nil
		}

		// Otherwise we have more traversal to do.
		sel := rest[0]

		return vs.flatMap(func(v value) (values, error) {
			nvs, err := sel.fn(c, v.value)
			if err != nil {
				if mode == selectorDropErrors {
					return nil, nil
				}

				return nil, err
			}

			return apply(v.prefix(nvs), mode.coalesce(sel.mode), rest[1:])
		})
	}

	return apply(root, p.mode, p.selectors)
}

func (p *path) evaluate(c context.Context, parameter interface{}) (interface{}, error) {
	pvs, err := p.reduce(c, parameter)
	if err != nil {
		return nil, err
	} else if pvs == nil {
		return nil, nil
	}
	return pvs.get(), nil
}
