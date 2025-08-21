package jsonpath

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/PaesslerAG/gval"
)

// plainSelector evaluate exactly one result
type plainSelector func(c context.Context, r, v interface{}) (interface{}, interface{}, error)

// ambiguousSelector evaluate wildcard
type ambiguousSelector func(c context.Context, r, v interface{}, match ambiguousMatcher)

// @
func currentElementSelector() plainSelector {
	return func(c context.Context, r, v interface{}) (interface{}, interface{}, error) {
		return nil, c.Value(currentElement{}), nil
	}
}

type currentElement struct{}

func currentContext(c context.Context, v interface{}) context.Context {
	return context.WithValue(c, currentElement{}, v)
}

// .x, [x]
func directSelector(key gval.Evaluable) plainSelector {
	return func(c context.Context, r, v interface{}) (interface{}, interface{}, error) {

		e, k, err := selectValue(c, key, r, v)
		if err != nil {
			return nil, nil, err
		}

		return k, e, nil
	}
}

// * / [*]
func starSelector() ambiguousSelector {
	return func(c context.Context, r, v interface{}, match ambiguousMatcher) {
		visitAll(v, func(key string, val interface{}) { match(key, val) })
	}
}

// [x, ...]
func multiSelector(keys []gval.Evaluable) ambiguousSelector {
	if len(keys) == 0 {
		return starSelector()
	}
	return func(c context.Context, r, v interface{}, match ambiguousMatcher) {
		for _, k := range keys {
			e, wildcard, err := selectValue(c, k, r, v)
			if err != nil {
				continue
			}
			match(wildcard, e)
		}
	}
}

func selectValue(c context.Context, key gval.Evaluable, r, v interface{}) (value interface{}, jkey string, err error) {

	c = currentContext(c, v)

	switch o := v.(type) {

	case []interface{}:
		i, err := key.EvalInt(c, r)
		if err != nil {
			return nil, "", fmt.Errorf("could not select value, invalid key: %s", err)
		}
		p := i
		if i < 0 {
			p = len(o) + i
		}
		if p < 0 || p >= len(o) {
			return nil, strconv.Itoa(i), nil
		}
		return o[p], strconv.Itoa(i), nil

	case map[string]interface{}:
		k, err := key.EvalString(c, r)
		if err != nil {
			return nil, "", fmt.Errorf("could not select value, invalid key: %s", err)
		}

		if r, ok := o[k]; ok {
			return r, k, nil
		}
		return nil, "", fmt.Errorf("unknown key %s", k)

	case Array:
		i, err := key.EvalInt(c, r)
		if err != nil {
			return nil, "", fmt.Errorf("could not select value, invalid key: %s", err)
		}
		p := i
		if i < 0 {
			p = o.Len() + i
		}
		if p < 0 || p >= o.Len() {
			return nil, strconv.Itoa(i), nil
		}
		r, err := o.SelectGVal(c, strconv.Itoa(p))
		if err != nil {
			return nil, "", err
		}
		return r, strconv.Itoa(i), nil

	case Object:
		k, err := key.EvalString(c, r)
		if err != nil {
			return nil, "", fmt.Errorf("could not select value, invalid key: %s", err)
		}

		r, err := o.SelectGVal(c, k)
		if err != nil {
			return nil, "", err
		}
		return r, k, nil

	default:
		return selectValueByReflect(c, key, r, v)
	}

}

// ..
func mapperSelector() ambiguousSelector {
	return mapper
}

func mapper(c context.Context, r, v interface{}, match ambiguousMatcher) {
	match([]interface{}{}, v)
	visitAll(v, func(wildcard string, v interface{}) {
		mapper(c, r, v, func(key interface{}, v interface{}) {
			match(append([]interface{}{wildcard}, key.([]interface{})...), v)
		})
	})
}

func selectValueByReflect(c context.Context, key gval.Evaluable, r, v interface{}) (value interface{}, jkey string, err error) {

	c = currentContext(c, v)

	// Use reflect to determine type
	val := reflect.ValueOf(v)

	// Handle pointer types
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, "", fmt.Errorf("nil pointer")
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		// Handle slice and array types
		i, err := key.EvalInt(c, r)
		if err != nil {
			return nil, "", fmt.Errorf("could not select value, invalid key: %s", err)
		}

		length := val.Len()
		p := i
		if i < 0 {
			p = length + i
		}
		if p < 0 || p >= length {
			return nil, strconv.Itoa(i), nil
		}

		// Get element value
		elem := val.Index(p)
		if elem.CanInterface() {
			return elem.Interface(), strconv.Itoa(i), nil
		}
		return nil, strconv.Itoa(i), fmt.Errorf("cannot access element at index %d", i)

	case reflect.Map:
		// Handle map types
		k, err := key.EvalString(c, r)
		if err != nil {
			return nil, "", fmt.Errorf("could not select value, invalid key: %s", err)
		}

		// Create reflect.Value for key
		keyVal := reflect.ValueOf(k)
		if !keyVal.Type().AssignableTo(val.Type().Key()) {
			return nil, "", fmt.Errorf("key type %T is not assignable to map key type %v", k, val.Type().Key())
		}

		// Find value in map
		elem := val.MapIndex(keyVal)
		if !elem.IsValid() {
			return nil, "", fmt.Errorf("unknown key %s", k)
		}

		if elem.CanInterface() {
			return elem.Interface(), k, nil
		}
		return nil, "", fmt.Errorf("cannot access value for key %s", k)

	case reflect.Struct:
		// Handle struct types
		k, err := key.EvalString(c, r)
		if err != nil {
			return nil, "", fmt.Errorf("could not select value, invalid key: %s", err)
		}

		// Find field, supporting both field name and JSON tag
		field, fieldName := findStructField(val, k)
		if !field.IsValid() {
			return nil, "", fmt.Errorf("unknown field %s", k)
		}

		if field.CanInterface() {
			return field.Interface(), fieldName, nil
		}
		return nil, "", fmt.Errorf("cannot access field %s", k)

	default:
		return nil, "", fmt.Errorf("unsupported value type %T for select, expected slice, array, map, struct, map[string]interface{}, []interface{}", val.Kind())
	}
}

func visitAll(v interface{}, visit func(key string, v interface{})) {

	switch v := v.(type) {

	case []interface{}:
		for i, e := range v {
			k := strconv.Itoa(i)
			visit(k, e)
		}

	case map[string]interface{}:
		for k, e := range v {
			visit(k, e)
		}

	case Array:
		v.ForEach(visit)

	case Object:
		v.ForEach(visit)
	default:
		visitAllByReflect(v, visit)
	}

}

func visitAllByReflect(v interface{}, visit func(key string, v interface{})) {

	// Use reflect to determine type
	val := reflect.ValueOf(v)

	// Handle pointer types
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		// Handle slice and array types
		for i := 0; i < val.Len(); i++ {
			k := strconv.Itoa(i)
			elem := val.Index(i)
			if elem.CanInterface() {
				visit(k, elem.Interface())
			}
		}

	case reflect.Map:
		// Handle map types
		iter := val.MapRange()
		for iter.Next() {
			key := iter.Key()
			value := iter.Value()
			if key.CanInterface() && value.CanInterface() {
				// Convert key to string
				var keyStr string
				switch k := key.Interface().(type) {
				case string:
					keyStr = k
				default:
					keyStr = fmt.Sprintf("%v", k)
				}
				visit(keyStr, value.Interface())
			}
		}

	case reflect.Struct:
		// Handle struct types
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			fieldType := val.Type().Field(i)
			if field.CanInterface() {
				// Prioritize JSON tag, if not, use field name
				key := fieldType.Name
				jsonTag := fieldType.Tag.Get("json")
				if jsonTag != "" {
					// Handle JSON tag, which may include options like "omitempty"
					if commaIndex := strings.Index(jsonTag, ","); commaIndex != -1 {
						jsonTag = jsonTag[:commaIndex]
					}
					if jsonTag != "" && jsonTag != "-" {
						key = jsonTag
					}
				}
				visit(key, field.Interface())
			}
		}

	}
}

// [? ]
func filterSelector(filter gval.Evaluable) ambiguousSelector {
	return func(c context.Context, r, v interface{}, match ambiguousMatcher) {
		visitAll(v, func(wildcard string, v interface{}) {
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

// [::]
func rangeSelector(min, max, step gval.Evaluable) ambiguousSelector {

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

		if step == 0 {
			step = 1
		}

		// process v
		switch o := v.(type) {

		case []interface{}:
			n := len(o)
			min = negmax(min, n)
			max = negmax(max, n)

			if min > max {
				return
			}

			if step > 0 {
				for i := min; i < max; i += step {
					match(strconv.Itoa(i), o[i])
				}
			} else {
				for i := max - 1; i >= min; i += step {
					match(strconv.Itoa(i), o[i])
				}
			}

		case Array:
			n := o.Len()
			min = negmax(min, n)
			max = negmax(max, n)

			if min > max {
				return
			}

			if step > 0 {
				for i := min; i < max; i += step {
					k := strconv.Itoa(i)
					r, _ := o.SelectGVal(c, k)
					match(k, r)
				}
			} else {
				for i := max - 1; i >= min; i += step {
					k := strconv.Itoa(i)
					r, _ := o.SelectGVal(c, k)
					match(k, r)
				}
			}
		default:
			val := reflect.ValueOf(v)

			// Handle pointer types
			if val.Kind() == reflect.Ptr {
				if val.IsNil() {
					return
				}
				val = val.Elem()
			}
			if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
				return
			}

			// Handle slice and array types
			n := val.Len()
			min = negmax(min, n)
			max = negmax(max, n)

			if min > max {
				return
			}

			if step > 0 {
				for i := min; i < max; i += step {
					elem := val.Index(i)
					if elem.CanInterface() {
						match(strconv.Itoa(i), elem.Interface())
					}
				}
			} else {
				for i := max - 1; i >= min; i += step {
					elem := val.Index(i)
					if elem.CanInterface() {
						match(strconv.Itoa(i), elem.Interface())
					}
				}
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
	return func(c context.Context, r, v interface{}) (interface{}, interface{}, error) {
		value, err := script(currentContext(c, v), r)
		return nil, value, err
	}
}

// findStructField finds struct field, supporting both field name and JSON tag
func findStructField(val reflect.Value, key string) (reflect.Value, string) {
	typ := val.Type()

	// First try direct field name matching
	if field := val.FieldByName(key); field.IsValid() {
		return field, key
	}

	// Then try JSON tag matching
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Get JSON tag
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag != "" {
			// Handle JSON tag, which may include options like "omitempty"
			if commaIndex := strings.Index(jsonTag, ","); commaIndex != -1 {
				jsonTag = jsonTag[:commaIndex]
			}

			if jsonTag == key {
				return field, fieldType.Name
			}
		}
	}

	// If not found, return invalid value
	return reflect.Value{}, ""
}
