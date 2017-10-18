package jsonpath_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/PaesslerAG/gval"
	"github.com/PaesslerAG/jsonpath"
)

type jsonpathTest struct {
	name         string
	path         string
	data         string
	extension    gval.Language
	want         interface{}
	wantErr      bool
	wantParseErr bool
}

type match struct {
	key   jsonpath.Wildcards
	value interface{}
}

type matchKey = []string
type obj = map[string]interface{}
type arr = []interface{}

func TestNew(t *testing.T) {

	tests := []jsonpathTest{
		{
			name: "root string",
			path: "$",
			data: `"hey"`,
			want: "hey",
		},
		{
			name: "root object",
			path: "$",
			data: `{"a":"aa"}`,
			want: obj{"a": "aa"},
		},
		{
			name: "simple select array",
			path: "$[1]",
			data: `[7, "hey"]`,
			want: "hey",
		},
		{
			name: "simple select object",
			path: "$[1]",
			data: `{"1":"aa"}`,
			want: "aa",
		},
		{
			name:    "simple select out of bounds",
			path:    "$[1]",
			data:    `["hey"]`,
			wantErr: true,
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
		},
		{
			name: "select object",
			path: "$[3].a",
			data: `{"3":{"a":"aa"}}`,
			want: "aa",
		},
		{
			name: "range array",
			path: "$[2:6].a",
			data: `[55,41,70,{"a":"bb"}]`,
			want: []match{
				{matchKey{"3"}, "bb"},
			},
		},
		{
			name: "range object", //no range over objects
			path: "$[2:6].a",
			data: `{"3":{"a":"aa"}}`,
			want: []match{},
		},
		{
			name: "range multi match",
			path: "$[2:6].a",
			data: `[{"a":"xx"},41,{"a":"b1"},{"a":"b2"},55,{"a":"b3"},{"a":"x2"} ]`,
			want: []match{
				{matchKey{"2"}, "b1"},
				{matchKey{"3"}, "b2"},
				{matchKey{"5"}, "b3"},
			},
		},
		{
			name: "range all",
			path: "$[:]",
			data: `[55,41,70,{"a":"bb"}]`,
			want: []match{
				{matchKey{"0"}, 55.},
				{matchKey{"1"}, 41.},
				{matchKey{"2"}, 70.},
				{matchKey{"3"}, obj{"a": "bb"}},
			},
		},
		{
			name: "range all even",
			path: "$[::2]",
			data: `[55,41,70,{"a":"bb"}]`,
			want: []match{
				{matchKey{"0"}, 55.},
				{matchKey{"2"}, 70.},
			},
		},
		{
			name: "range all even reverse",
			path: "$[::-2]",
			data: `[55,41,70,{"a":"bb"}]`,
			want: []match{
				{matchKey{"3"}, obj{"a": "bb"}},
				{matchKey{"1"}, 41.},
			},
		},
		{
			name: "range reverse",
			path: "$[2:6:-1].a",
			data: `[55,41,70,{"a":"bb"}]`,
			want: []match{
				{matchKey{"3"}, "bb"},
			},
		},
		{
			name: "range reverse multi match",
			path: "$[2:6:-1].a",
			data: `[{"a":"xx"},41,{"a":"b1"},{"a":"b2"},55,{"a":"b3"},{"a":"x2"} ]`,
			want: []match{
				{matchKey{"5"}, "b3"},
				{matchKey{"3"}, "b2"},
				{matchKey{"2"}, "b1"},
			},
		},
		{
			name: "range even selection",
			path: "$[2:6:2].a",
			data: `[55,41,70,{"a":"bb"}]`,
			want: []match{},
		},
		{
			name: "range even multi match selection",
			path: "$[2:6:2].a",
			data: `[{"a":"xx"},41,{"a":"b1"},{"a":"b2"},{"a":"b3"},{"a":"x2"} ]`,
			want: []match{
				{matchKey{"2"}, "b1"},
				{matchKey{"4"}, "b3"},
			},
		},
		{
			name: "current",
			path: "$.a[@.max]",
			data: `{"a":{"max":"3a", "3a":"aa"}, "1":{"a":"1a"}, "x":{"7":"bb"}}`,
			want: "aa",
		},
		{
			name: "multi pick array",
			path: "$[1, 3].a",
			data: `[55,{"a":"1a"},70,{"a":"bb"}]`,
			want: []match{
				{matchKey{"1"}, "1a"},
				{matchKey{"3"}, "bb"},
			},
		},
		{
			name: "multi pick object",
			path: "$[1, 3].a",
			data: `{"3":{"a":"3a"}, "1":{"a":"1a"}, "x":{"7":"bb"}}`,
			want: []match{
				{matchKey{"1"}, "1a"},
				{matchKey{"3"}, "3a"},
			},
		},
		{
			name: "multi pick array partilly matched",
			path: "$[1, 3].a",
			data: `[55,41,70,{"a":"bb"}]`,
			want: []match{
				{matchKey{"3"}, "bb"},
			},
		},
		{
			name: "multi pick object partilly matched",
			path: "$[1, 3].a",
			data: `{"1":{"a":"aa"}, "3":{}, "x":{"7":"bb"}}`,
			want: []match{
				{matchKey{"1"}, "aa"},
			},
		},
		{
			name: "multi pick star array",
			path: "$[1, 3].*",
			data: `[55,{"a":"1a"},70,{"b":"bb", "c":"cc"}]`,
			want: []match{
				{matchKey{"1", "a"}, "1a"},
				{matchKey{"3", "b"}, "bb"},
				{matchKey{"3", "c"}, "cc"},
			},
		},
		{
			name: "multi pick star object",
			path: "$[1, 3].*",
			data: `{"3":{"a":"3a"}, "1":{"7":"1a"}, "x":{"a":"bb"}}`,
			want: []match{
				{matchKey{"1", "7"}, "1a"},
				{matchKey{"3", "a"}, "3a"},
			},
		},
		{
			name: "multi pick star array partilly matched",
			path: "$[1, 3].*",
			data: `[55,41,70,{"a":"bb"}]`,
			want: []match{
				{matchKey{"3", "a"}, "bb"},
			},
		},
		{
			name: "multi pick star object partilly matched",
			path: "$[1, 3].*",
			data: `{"1":{"a":"aa", "7":"cc"}, "3":{}, "x":{"7":"bb"}}`,
			want: []match{
				{matchKey{"1", "7"}, "cc"},
				{matchKey{"1", "a"}, "aa"},
			},
		},
		{
			name: "multi pick bracket star array",
			path: "$[1, 3][*]",
			data: `[55,{"a":"1a"},70,{"b":"bb", "c":"cc"}]`,
			want: []match{
				{matchKey{"1", "a"}, "1a"},
				{matchKey{"3", "b"}, "bb"},
				{matchKey{"3", "c"}, "cc"},
			},
		},
		{
			name: "multi pick bracket star object",
			path: "$[1, 3][*]",
			data: `{"3":{"a":"3a"}, "1":{"7":"1a"}, "x":{"a":"bb"}}`,
			want: []match{
				{matchKey{"1", "7"}, "1a"},
				{matchKey{"3", "a"}, "3a"},
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
			want: []match{
				{matchKey{`["a"]`}, 1.},
				{matchKey{`["b"]["0"]`}, 2.},
				{matchKey{``}, 4.},
			},
		},
		{
			name: "mapper multipick",
			path: `$..["x", "a"]`,
			data: `{
						"a" : {"x" : 1},
						"b" : [{"x" : 2}, {"y" : 3}],
						"x" : 4
					}`,
			want: []match{
				{matchKey{``, `a`}, obj{"x": 1.}},
				{matchKey{`["a"]`, `x`}, 1.},
				{matchKey{`["b"]["0"]`, `x`}, 2.},
				{matchKey{``, `x`}, 4.},
			},
		},
		{
			name: "mapper star",
			path: `$..*`,
			data: `{"1":{"a":"aa", "b":[1 ,2, 3]}, "3":{}, "x":{"7":"bb"}}`,
			want: []match{
				{matchKey{"", `1`}, obj{"a": "aa", "b": arr{1., 2., 3.}}},
				{matchKey{`["1"]`, "a"}, "aa"},
				{matchKey{`["1"]`, "b"}, arr{1., 2., 3.}},
				{matchKey{`["1"]["b"]`, "0"}, 1.},
				{matchKey{`["1"]["b"]`, "1"}, 2.},
				{matchKey{`["1"]["b"]`, "2"}, 3.},
				{matchKey{``, "3"}, obj{}},
				{matchKey{``, "x"}, obj{"7": "bb"}},
				{matchKey{`["x"]`, "7"}, "bb"},
			},
		},
		{
			name: "mapper filter true",
			path: `$..[?true]`,
			data: `{"1":{"a":"aa", "b":[1 ,2, 3]}, "3":{}, "x":{"7":"bb"}}`,
			want: []match{
				{matchKey{"", `1`}, obj{"a": "aa", "b": arr{1., 2., 3.}}},
				{matchKey{`["1"]`, "a"}, "aa"},
				{matchKey{`["1"]`, "b"}, arr{1., 2., 3.}},
				{matchKey{`["1"]["b"]`, "0"}, 1.},
				{matchKey{`["1"]["b"]`, "1"}, 2.},
				{matchKey{`["1"]["b"]`, "2"}, 3.},
				{matchKey{``, "3"}, obj{}},
				{matchKey{``, "x"}, obj{"7": "bb"}},
				{matchKey{`["x"]`, "7"}, "bb"},
			},
		},
		{
			name: "mapper filter a=aa",
			path: `$..[?@.a=="aa"]`,
			data: `{"1":{"a":"aa", "b":[1 ,2, 3]}, "3":{}, "x":{"7":"bb"}, "y":{"a":"bb"}}`,
			want: []match{
				{matchKey{``, "1"}, obj{"a": "aa", "b": arr{1., 2., 3.}}},
			},
			extension: gval.Full(),
		},
		{
			name: "mapper filter (a=aa)",
			path: `$..[?(@.a=="aa")]`,
			data: `{"1":{"a":"aa", "b":[1 ,2, 3]}, "3":{}, "x":{"7":"bb"}, "y":{"a":"bb"}}`,
			want: []match{
				{matchKey{``, "1"}, obj{"a": "aa", "b": arr{1., 2., 3.}}},
			},
		},
		{
			name: "key value",
			path: `$[?(@.key=="x")].value`,
			data: `[{"key": "x","value":"a"},{"key": "y","value":"b"}]`,
			want: []match{
				{matchKey{`0`}, "a"},
			},
		},
		{
			name: "script",
			path: `$.*.value(@=="a")`,
			data: `[{"key": "x","value":"a"},{"key": "y","value":"b"}]`,
			want: []match{
				{matchKey{`0`}, true},
				{matchKey{`1`}, false},
			},
		},
		{
			name: "mapper script",
			path: `$..(@=="a")`,
			data: `[{"key": "x","value":"a"},{"key": "y","value":"b"}]`,
			want: []match{
				{matchKey{``}, false},
				{matchKey{`["0"]`}, false},
				{matchKey{`["1"]`}, false},
				{matchKey{`["0"]["key"]`}, false},
				{matchKey{`["0"]["value"]`}, true},
				{matchKey{`["1"]["key"]`}, false},
				{matchKey{`["1"]["value"]`}, false},
			},
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
			want: []match{
				{matchKey{`["a"]`, `x`}, true},
				{matchKey{`["b"]`, `x`}, true},
				{matchKey{`["c"]`, `x`}, false},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func (tt jsonpathTest) test(t *testing.T) {
	get, err := jsonpath.New(tt.path)
	if (err != nil) != tt.wantParseErr {
		t.Fatalf("New() error = %v, wantErr %v", err, tt.wantErr)
	}
	if tt.wantParseErr {
		return
	}
	var v interface{}
	err = json.Unmarshal([]byte(tt.data), &v)
	if err != nil {
		t.Fatalf("could not parse json input: %v", err)
	}
	got, err := get(context.Background(), v)
	if (err != nil) != tt.wantErr {
		t.Errorf("New()(*) error = %v, wantErr %v", err, tt.wantErr)
		return
	}

	if tt.wantErr {
		return
	}

	if matchs, ok := tt.want.([]match); ok {
		gotMatchs, ok := got.(jsonpath.Matchs)
		if !ok {
			t.Fatalf("expected multiple results but got %v (%T)", got, got)
		}
		for _, match := range matchs {
			var key *jsonpath.Wildcards
			for k := range gotMatchs {
				if reflect.DeepEqual(*k, match.key) {
					key = k
					break
				}
			}
			if key == nil {
				t.Fatalf("missing %v in %+v", match, gotMatchs)
			}
			if !reflect.DeepEqual(match.value, gotMatchs[key]) {
				t.Fatalf("expected %v, but got %v for key %v", match.value, gotMatchs[key], *key)
			}
			delete(gotMatchs, key)
		}
		if len(gotMatchs) > 0 {
			t.Fatalf("unexpected matchs %v", gotMatchs)
		}
		return
	}

	if !reflect.DeepEqual(got, tt.want) {
		t.Fatalf("expected %v, but got %v", tt.want, got)
	}
}
