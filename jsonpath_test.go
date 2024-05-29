package jsonpath_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/PaesslerAG/gval"
	"github.com/PaesslerAG/jsonpath"
)

type jsonpathTest struct {
	name          string
	path          string
	data          string
	lang          gval.Language
	reorder       bool
	want          interface{}
	wantWithPaths map[string]interface{}
	wantErr       bool
	wantParseErr  bool
}

type obj = map[string]interface{}
type arr = []interface{}

func TestJsonPath(t *testing.T) {

	tests := []jsonpathTest{
		{
			name: "root string",
			path: "$",
			data: `"hey"`,
			want: "hey",
			wantWithPaths: obj{
				"$": "hey",
			},
		},
		{
			name: "root object",
			path: "$",
			data: `{"a":"aa"}`,
			want: obj{"a": "aa"},
			wantWithPaths: obj{
				"$": obj{"a": "aa"},
			},
		},
		{
			name: "simple select array",
			path: "$[1]",
			data: `[7, "hey"]`,
			want: "hey",
			wantWithPaths: obj{
				`$["1"]`: "hey",
			},
		},
		{
			name: "negative select array",
			path: "$[-1]",
			data: `[7, "hey"]`,
			want: "hey",
			wantWithPaths: obj{
				`$["-1"]`: "hey",
			},
		},
		{
			name: "negative select on short array",
			path: "$[-2]",
			data: `[7]`,
			want: nil,
			wantWithPaths: obj{ // Not sure this makes sense, but not sure how to tell no match from a "null" match
				`$["-2"]`: nil,
			},
		},
		{
			name: "simple select object",
			path: "$[1]",
			data: `{"1":"aa"}`,
			want: "aa",
			wantWithPaths: obj{
				`$["1"]`: "aa",
			},
		},
		{
			name: "simple select out of bounds",
			path: "$[1]",
			data: `["hey"]`,
			want: nil,
			wantWithPaths: obj{ // Not sure this makes sense, but not sure how to tell no match from a "null" match
				`$["1"]`: nil,
			},
		},
		{
			name:    "simple select unknown key",
			path:    "$[1]",
			data:    `{"2":"aa"}`,
			wantErr: true,
		},
		{
			name: "select array",
			path: "$[3].a",
			data: `[55,41,70,{"a":"bb"}]`,
			want: "bb",
			wantWithPaths: obj{
				`$["3"]["a"]`: "bb",
			},
		},
		{
			name: "select object",
			path: "$[3].a",
			data: `{"3":{"a":"aa"}}`,
			want: "aa",
			wantWithPaths: obj{
				`$["3"]["a"]`: "aa",
			},
		},
		{
			name: "range array",
			path: "$[2:6].a",
			data: `[55,41,70,{"a":"bb"}]`,
			want: arr{"bb"},
			wantWithPaths: obj{
				`$["3"]["a"]`: "bb",
			},
		},
		{
			name:          "range object", //no range over objects
			path:          "$[2:6].a",
			data:          `{"3":{"a":"aa"}}`,
			want:          arr{},
			wantWithPaths: obj{},
		},
		{
			name: "range multi match",
			path: "$[2:6].a",
			data: `[{"a":"xx"},41,{"a":"b1"},{"a":"b2"},55,{"a":"b3"},{"a":"x2"} ]`,
			want: arr{
				"b1",
				"b2",
				"b3",
			},
			wantWithPaths: obj{
				`$["2"]["a"]`: "b1",
				`$["3"]["a"]`: "b2",
				`$["5"]["a"]`: "b3",
			},
		},
		{
			name: "range all",
			path: "$[:]",
			data: `[55,41,70,{"a":"bb"}]`,
			want: arr{
				55.,
				41.,
				70.,
				obj{"a": "bb"},
			},
			wantWithPaths: obj{
				`$["0"]`: 55.,
				`$["1"]`: 41.,
				`$["2"]`: 70.,
				`$["3"]`: obj{"a": "bb"},
			},
		},
		{
			name: "range all even",
			path: "$[::2]",
			data: `[55,41,70,{"a":"bb"}]`,
			want: arr{
				55.,
				70.,
			},
			wantWithPaths: obj{
				`$["0"]`: 55.,
				`$["2"]`: 70.,
			},
		},
		{
			name: "range all even reverse",
			path: "$[::-2]",
			data: `[55,41,70,{"a":"bb"}]`,
			want: arr{
				obj{"a": "bb"},
				41.,
			},
			wantWithPaths: obj{
				`$["3"]`: obj{"a": "bb"},
				`$["1"]`: 41.,
			},
		},
		{
			name: "range reverse",
			path: "$[2:6:-1].a",
			data: `[55,41,70,{"a":"bb"}]`,
			want: arr{
				"bb",
			},
			wantWithPaths: obj{
				`$["3"]["a"]`: "bb",
			},
		},
		{
			name: "range reverse multi match",
			path: "$[2:6:-1].a",
			data: `[{"a":"xx"},41,{"a":"b1"},{"a":"b2"},55,{"a":"b3"},{"a":"x2"} ]`,
			want: arr{
				"b3",
				"b2",
				"b1",
			},
			wantWithPaths: obj{
				`$["5"]["a"]`: "b3",
				`$["3"]["a"]`: "b2",
				`$["2"]["a"]`: "b1",
			},
		},
		{
			name:          "range even selection",
			path:          "$[2:6:2].a",
			data:          `[55,41,70,{"a":"bb"}]`,
			want:          arr{},
			wantWithPaths: obj{},
		},
		{
			name: "range even multi match selection",
			path: "$[2:6:2].a",
			data: `[{"a":"xx"},41,{"a":"b1"},{"a":"b2"},{"a":"b3"},{"a":"x2"} ]`,
			want: arr{
				"b1",
				"b3",
			},
			wantWithPaths: obj{
				`$["2"]["a"]`: "b1",
				`$["4"]["a"]`: "b3",
			},
		},
		{
			name: "current",
			path: "$.a[@.max]",
			data: `{"a":{"max":"3a", "3a":"aa"}, "1":{"a":"1a"}, "x":{"7":"bb"}}`,
			want: "aa",
			wantWithPaths: obj{
				`$["a"]["3a"]`: "aa",
			},
		},
		{
			name: "union array",
			path: "$[1, 3].a",
			data: `[55,{"a":"1a"},70,{"a":"bb"}]`,
			want: arr{
				"1a",
				"bb",
			},
			wantWithPaths: obj{
				`$["1"]["a"]`: "1a",
				`$["3"]["a"]`: "bb",
			},
		},
		{
			name: "negative union array",
			path: "$[1, -5, 3].a",
			data: `[55,{"a":"1a"},70,{"a":"bb"}]`,
			want: arr{
				"1a",
				"bb",
			},
			wantWithPaths: obj{
				`$["1"]["a"]`: "1a",
				`$["3"]["a"]`: "bb",
			},
		},
		{
			name: "union object",
			path: "$[1, 3].a",
			data: `{"3":{"a":"3a"}, "1":{"a":"1a"}, "x":{"7":"bb"}}`,
			want: arr{
				"1a",
				"3a",
			},
			wantWithPaths: obj{
				`$["1"]["a"]`: "1a",
				`$["3"]["a"]`: "3a",
			},
		},
		{
			name: "union array partilly matched",
			path: "$[1, 3].a",
			data: `[55,41,70,{"a":"bb"}]`,
			want: arr{
				"bb",
			},
			wantWithPaths: obj{
				`$["3"]["a"]`: "bb",
			},
		},
		{
			name: "union object partilly matched",
			path: "$[1, 3].a",
			data: `{"1":{"a":"aa"}, "3":{}, "x":{"7":"bb"}}`,
			want: arr{
				"aa",
			},
			wantWithPaths: obj{
				`$["1"]["a"]`: "aa",
			},
		},
		{
			name: "union wildcard array",
			path: "$[1, 3].*",
			data: `[55,{"a":"1a"},70,{"b":"bb", "c":"cc"}]`,
			want: arr{
				"1a",
				"bb",
				"cc",
			},
			wantWithPaths: obj{
				`$["1"]["a"]`: "1a",
				`$["3"]["b"]`: "bb",
				`$["3"]["c"]`: "cc",
			},
			reorder: true,
		},
		{
			name: "union wildcard object",
			path: "$[1, 3].*",
			data: `{"3":{"a":"3a"}, "1":{"7":"1a"}, "x":{"a":"bb"}}`,
			want: arr{
				"1a",
				"3a",
			},
			wantWithPaths: obj{
				`$["1"]["7"]`: "1a",
				`$["3"]["a"]`: "3a",
			},
		},
		{
			name: "union wildcard array partilly matched",
			path: "$[1, 3].*",
			data: `[55,41,70,{"a":"bb"}]`,
			want: arr{
				"bb",
			},
			wantWithPaths: obj{
				`$["3"]["a"]`: "bb",
			},
		},
		{
			name: "union wildcard object partilly matched",
			path: "$[1, 3].*",
			data: `{"1":{"a":"aa", "7":"cc"}, "3":{}, "x":{"7":"bb"}}`,
			want: arr{
				"aa",
				"cc",
			},
			wantWithPaths: obj{
				`$["1"]["a"]`: "aa",
				`$["1"]["7"]`: "cc",
			},
			reorder: true,
		},
		{
			name: "union bracket wildcard array",
			path: "$[1, 3][*]",
			data: `[55,{"a":"1a"},70,{"b":"bb", "c":"cc"}]`,
			want: arr{
				"1a",
				"bb",
				"cc",
			},
			wantWithPaths: obj{
				`$["1"]["a"]`: "1a",
				`$["3"]["b"]`: "bb",
				`$["3"]["c"]`: "cc",
			},
			reorder: true,
		},
		{
			name: "union bracket wildcard object",
			path: "$[1, 3][*]",
			data: `{"3":{"a":"3a"}, "1":{"7":"1a"}, "x":{"a":"bb"}}`,
			want: arr{
				"1a",
				"3a",
			},
			wantWithPaths: obj{
				`$["1"]["7"]`: "1a",
				`$["3"]["a"]`: "3a",
			},
		},
		{
			name:         "incomplete",
			path:         "$[3].",
			wantParseErr: true,
		},
		{
			name:         "mixed bracket",
			path:         "$[3,5:1].",
			wantParseErr: true,
		},
		{
			name: "mapper",
			path: "$..x",
			data: `{
					"a" : {"x" : 1},
					"b" : [{"x" : 2}, {"y" : 3}],
					"x" : 4
				}`,
			want: arr{
				1.,
				2.,
				4.,
			},
			wantWithPaths: obj{
				`$["a"]["x"]`:      1.,
				`$["b"]["0"]["x"]`: 2.,
				`$["x"]`:           4.,
			},
			reorder: true,
		},
		{
			name: "mapper union",
			path: `$..["x", "a"]`,
			data: `{
					"a" : {"x" : 1},
					"b" : [{"x" : 2}, {"y" : 3}],
					"x" : 4
				}`,
			want: arr{
				1.,
				2.,
				4.,
				obj{"x": 1.},
			},
			wantWithPaths: obj{
				`$["a"]["x"]`:      1.,
				`$["b"]["0"]["x"]`: 2.,
				`$["x"]`:           4.,
				`$["a"]`:           obj{"x": 1.},
			},
			reorder: true,
		},
		{
			name: "mapper wildcard",
			path: `$..*`,
			data: `{"1":{"a":"aa", "b":[1 ,2, 3]}, "3":{}, "x":{"7":"bb"}}`,
			want: arr{
				1.,
				2.,
				3.,
				"aa",
				"bb",
				arr{1., 2., 3.},
				obj{},
				obj{"7": "bb"},
				obj{"a": "aa", "b": arr{1., 2., 3.}},
			},
			wantWithPaths: obj{
				`$["1"]["b"]["0"]`: 1.,
				`$["1"]["b"]["1"]`: 2.,
				`$["1"]["b"]["2"]`: 3.,
				`$["1"]["a"]`:      "aa",
				`$["x"]["7"]`:      "bb",
				`$["1"]["b"]`:      arr{1., 2., 3.},
				`$["3"]`:           obj{},
				`$["x"]`:           obj{"7": "bb"},
				`$["1"]`:           obj{"a": "aa", "b": arr{1., 2., 3.}},
			},
			reorder: true,
		},
		{
			name: "mapper filter true",
			path: `$..[?true]`,
			data: `{"1":{"a":"aa", "b":[1 ,2, 3]}, "3":{}, "x":{"7":"bb"}}`,
			want: arr{
				1.,
				2.,
				3.,
				"aa",
				"bb",
				arr{1., 2., 3.},
				obj{},
				obj{"7": "bb"},
				obj{"a": "aa", "b": arr{1., 2., 3.}},
			},
			wantWithPaths: obj{
				`$["1"]["b"]["0"]`: 1.,
				`$["1"]["b"]["1"]`: 2.,
				`$["1"]["b"]["2"]`: 3.,
				`$["1"]["a"]`:      "aa",
				`$["x"]["7"]`:      "bb",
				`$["1"]["b"]`:      arr{1., 2., 3.},
				`$["3"]`:           obj{},
				`$["x"]`:           obj{"7": "bb"},
				`$["1"]`:           obj{"a": "aa", "b": arr{1., 2., 3.}},
			},
			reorder: true,
		},
		{
			name: "mapper filter a=aa",
			path: `$..[?@.a=="aa"]`,
			data: `{"1":{"a":"aa", "b":[1 ,2, 3]}, "3":{}, "x":{"7":"bb"}, "y":{"a":"bb"}}`,
			want: arr{
				obj{"a": "aa", "b": arr{1., 2., 3.}},
			},
			wantWithPaths: obj{
				`$["1"]`: obj{"a": "aa", "b": arr{1., 2., 3.}},
			},
		},
		{
			name: "mapper filter (a=aa)",
			path: `$..[?(@.a=="aa")]`,
			data: `{"1":{"a":"aa", "b":[1 ,2, 3]}, "3":{}, "x":{"7":"bb"}, "y":{"a":"bb"}}`,
			want: arr{
				obj{"a": "aa", "b": arr{1., 2., 3.}},
			},
			wantWithPaths: obj{
				`$["1"]`: obj{"a": "aa", "b": arr{1., 2., 3.}},
			},
		},
		{
			name: "key value",
			path: `$[?(@.key=="x")].value`,
			data: `[{"key": "x","value":"a"},{"key": "y","value":"b"}]`,
			want: arr{
				"a",
			},
			wantWithPaths: obj{
				`$["0"]["value"]`: "a",
			},
		},
		{
			name: "script",
			path: `$.*.value(@=="a")`,
			data: `[{"key": "x","value":"a"},{"key": "y","value":"b"}]`,
			want: arr{
				true,
				false,
			},
			wantWithPaths: obj{
				`$["0"]["value"]`: true,
				`$["1"]["value"]`: false,
			},
		},
		{
			name: "mapper script",
			path: `$..(@=="a")`,
			data: `[{"key": "x","value":"a"},{"key": "y","value":"b"}]`,
			want: arr{
				false,
				false,
				false,
				false,
				false,
				false,
				true,
			},
			wantWithPaths: obj{
				`$`:               false,
				`$["0"]`:          false,
				`$["1"]`:          false,
				`$["key"]["0"]`:   false,
				`$["key"]["1"]`:   false,
				`$["value"]["0"]`: true,
				`$["value"]["1"]`: false,
			},
			reorder: true,
		},
		{
			name: "mapper select script",
			path: `$.abc.f..["x"](@ == "1")`,
			data: `{
					"abc":{
						"d":[
							"1",
							"1"
						],
						"f":{
							"a":{
								"x":"1"
							},
							"b":{
								"x":"1"
							},
							"c":{
								"x":"xx"
							}
						}
					}
				}`,
			want: arr{
				false,
				true,
				true,
			},
			wantWithPaths: obj{
				`$["abc"]["f"]["a"]["x"]`: true,
				`$["abc"]["f"]["b"]["x"]`: true,
				`$["abc"]["f"]["c"]["x"]`: false,
			},
			reorder: true,
		},
		{
			name: "float equal",
			path: `$.a == 1.23`,
			data: `{"a":1.23, "b":2}`,
			want: true,
			// wantWithPaths: obj{}, Not sure how to handle this one, since it's not a match but a script evaluation
		},
		{
			name: "ending star",
			path: `$.welcome.message[*]`,
			data: `{"welcome":{"message":["Good Morning", "Hello World!"]}}`,
			want: arr{"Good Morning", "Hello World!"},
			wantWithPaths: obj{
				`$["welcome"]["message"]["0"]`: "Good Morning",
				`$["welcome"]["message"]["1"]`: "Hello World!",
			},
		},
	}
	for _, tt := range tests {
		tt.lang = jsonpath.Language()
		t.Run(tt.name, tt.test)
	}
}

func (tt jsonpathTest) test(t *testing.T) {
	get, err := tt.lang.NewEvaluable(tt.path)
	if (err != nil) != tt.wantParseErr {
		t.Fatalf("[%s]: New() error = %v, wantErr %v", tt.name, err, tt.wantErr)
	}
	if tt.wantParseErr {
		return
	}
	var v interface{}
	err = json.Unmarshal([]byte(tt.data), &v)
	if err != nil {
		t.Fatalf("[%s]: could not parse json input: %v", tt.name, err)
	}
	got, err := get(context.Background(), v)

	if tt.wantErr {
		if err == nil {
			t.Errorf("[%s]: expected error %v but got %v", tt.name, tt.wantErr, got)
			return
		}
		return
	}

	if err != nil {
		t.Errorf("[%s]: JSONPath(%s) error = %v", tt.name, tt.path, err)
		return
	}

	if tt.reorder {
		reorder(got.(arr))
	}

	if !reflect.DeepEqual(got, tt.want) {
		t.Fatalf("[%s]: expected %v, but got %v", tt.name, tt.want, got)
	}

	if tt.wantWithPaths != nil {
		tName := fmt.Sprintf("%s-with-paths", tt.name)
		ctx := context.WithValue(context.Background(), jsonpath.CollectFullPathsContextKey{}, true)
		get, err = tt.lang.NewEvaluableWithContext(ctx, tt.path)
		if err != nil {
			t.Errorf("[%s]: expected nil, but got %v", tName, got)
		}

		got, err = get(ctx, v)
		if err != nil {
			t.Errorf("[%s]: expected nil, but got %v", tName, got)
		}

		if !reflect.DeepEqual(got, tt.wantWithPaths) {
			t.Fatalf("[%s]: expected %v, but got %v", tName, tt.wantWithPaths, got)
		}
	}
}

func reorder(sl []interface{}) {
	sort.Slice(sl, func(i, j int) bool {
		a := sl[i]
		b := sl[j]
		if reflect.TypeOf(a) != reflect.TypeOf(b) {
			return typeOrder(a) < typeOrder(b)
		}

		switch a := a.(type) {
		case string:
			return a < b.(string)
		case float64:
			return a < b.(float64)
		case bool:
			return !a || b.(bool)
		case arr:
			return len(a) < len(b.(arr))
		case obj:
			return len(a) < len(b.(obj))
		default:
			panic(fmt.Errorf("unknown type %T", a))
		}
	})
}

func typeOrder(o interface{}) int {
	switch o.(type) {
	case bool:
		return 0
	case float64:
		return 1
	case string:
		return 2
	case arr:
		return 3
	case obj:
		return 4

	default:
		panic(fmt.Errorf("unknown type %T", o))
	}
}
