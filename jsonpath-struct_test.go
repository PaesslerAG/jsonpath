package jsonpath_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/PaesslerAG/jsonpath"
	"github.com/google/go-cmp/cmp"
)

type ArrayLike []int

func (array ArrayLike) SelectGVal(c context.Context, key string) (interface{}, error) {
	// index
	i, err := strconv.Atoi(key)
	if err != nil {
		return nil, err
	}
	// check bounds
	if i >= 0 || i < len(array) {
		return array[i], nil
	}
	return nil, nil
}

func (array ArrayLike) Len() int {
	return len(array)
}

func (array ArrayLike) ForEach(callback func(key string, v interface{})) {
	// loop
	for i, v := range array {
		callback(strconv.Itoa(i), v)
	}
}

type ObjectLike1 map[string]int

func (object ObjectLike1) SelectGVal(c context.Context, key string) (interface{}, error) {
	// check field exists
	if v, ok := object[key]; ok {
		return v, nil
	}
	return nil, nil
}

func (object ObjectLike1) ForEach(callback func(key string, v interface{})) {
	// loop
	for k, v := range object {
		callback(k, v)
	}
}

type MyStruct struct {
	a string
	b string
}

func (s *MyStruct) SelectGVal(c context.Context, key string) (interface{}, error) {
	// check key
	switch key {
	case "a":
		return s.a, nil
	case "b":
		return s.b, nil
	default:
		return nil, nil
	}
}

func (s *MyStruct) ForEach(callback func(key string, v interface{})) {
	callback("a", s.a)
	callback("b", s.b)
}

func TestArraySelectors(t *testing.T) {
	// init
	array := &ArrayLike{1, 2, 3, 4, 5}
	// assert
	assert(t, "$.*", array, []interface{}{1, 2, 3, 4, 5})
	assert(t, "$[0]", array, 1)
	assert(t, "$[-1]", array, 5)
	assert(t, "$[2, 3]", array, []interface{}{3, 4})
}

func TestObjectSelectors(t *testing.T) {
	// init
	s := &MyStruct{
		a: "v1",
		b: "v2",
	}
	// assert
	assert(t, "$.a", s, "v1")
	assert(t, "$.b", s, "v2")
}

func TestStructSelectors(t *testing.T) {
	// init
	object := &ObjectLike1{
		"a": 1,
		"b": -1,
	}
	// assert
	assert(t, "$.a", object, 1)
	assert(t, "$.b", object, -1)
}

func assert(t *testing.T, path string, value interface{}, expected interface{}) {
	// evaluate path
	result, err := jsonpath.Get(path, value)
	if err != nil {
		t.Errorf("failed to evaluate path expression: %s -> %v", path, err)
	}
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("invalid execution result: %s", diff)
	}
}
