package jsonpath

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/PaesslerAG/gval"
)

func TestNew(t *testing.T) {
	type matchKey = []string
	type match struct {
		key   Wildcards
		value interface{}
	}
	type obj = map[string]interface{}
	type arr = []interface{}
	type subcase struct {
		name    string
		data    string
		want    interface{}
		wantErr bool
	}
	tests := []struct {
		name      string
		path      string
		sub       []subcase
		extension gval.Language
		wantErr   bool
	}{
		{
			name: "root",
			path: "$",
			sub: []subcase{
				{
					name: "string",
					data: `"hey"`,
					want: "hey",
				},
				{
					name: "object",
					data: `{"a":"aa"}`,
					want: obj{"a": "aa"},
				},
			},
		},
		{
			name: "simple_select",
			path: "$[1]",
			sub: []subcase{
				{
					name: "array",
					data: `[7, "hey"]`,
					want: "hey",
				},
				{
					name: "object",
					data: `{"1":"aa"}`,
					want: "aa",
				},
				{
					name:    "array out of bounds",
					data:    `["hey"]`,
					wantErr: true,
				},
				{
					name:    "object out of bounds",
					data:    `{"2":"aa"}`,
					wantErr: true,
				},
			},
		},
		{
			name: "select",
			path: "$[3].a",
			sub: []subcase{
				{
					name: "array",
					data: `[55,41,70,{"a":"bb"}]`,
					want: "bb",
				},
				{
					name: "object",
					data: `{"3":{"a":"aa"}}`,
					want: "aa",
				},
			},
		},
		{
			name: "range",
			path: "$[2:6].a",
			sub: []subcase{
				{
					name: "array",
					data: `[55,41,70,{"a":"bb"}]`,
					want: []match{
						{matchKey{"3"}, "bb"},
					},
				},
				{
					name: "object", //no range over objects
					data: `{"3":{"a":"aa"}}`,
					want: []match{},
				},
				{
					name: "array multi match",
					data: `[{"a":"xx"},41,{"a":"b1"},{"a":"b2"},55,{"a":"b3"},{"a":"x2"} ]`,
					want: []match{
						{matchKey{"2"}, "b1"},
						{matchKey{"3"}, "b2"},
						{matchKey{"5"}, "b3"},
					},
				},
			},
		},
		{
			name: "range all",
			path: "$[:]",
			sub: []subcase{
				{
					name: "array",
					data: `[55,41,70,{"a":"bb"}]`,
					want: []match{
						{matchKey{"0"}, 55.},
						{matchKey{"1"}, 41.},
						{matchKey{"2"}, 70.},
						{matchKey{"3"}, obj{"a": "bb"}},
					},
				},
			},
		},
		{
			name: "range all even",
			path: "$[::2]",
			sub: []subcase{
				{
					name: "array",
					data: `[55,41,70,{"a":"bb"}]`,
					want: []match{
						{matchKey{"0"}, 55.},
						{matchKey{"2"}, 70.},
					},
				},
			},
		},
		{
			name: "range all even reverse",
			path: "$[::-2]",
			sub: []subcase{
				{
					name: "array",
					data: `[55,41,70,{"a":"bb"}]`,
					want: []match{
						{matchKey{"3"}, obj{"a": "bb"}},
						{matchKey{"1"}, 41.},
					},
				},
			},
		},
		{
			name: "range reverse",
			path: "$[2:6:-1].a",
			sub: []subcase{
				{
					name: "array",
					data: `[55,41,70,{"a":"bb"}]`,
					want: []match{
						{matchKey{"3"}, "bb"},
					},
				},
				{
					name: "object",
					data: `{"3":{"a":"aa"}}`,
					want: []match{},
				},
				{
					name: "array multi match",
					data: `[{"a":"xx"},41,{"a":"b1"},{"a":"b2"},55,{"a":"b3"},{"a":"x2"} ]`,
					want: []match{
						{matchKey{"5"}, "b3"},
						{matchKey{"3"}, "b2"},
						{matchKey{"2"}, "b1"},
					},
				},
			},
		},
		{
			name: "range even selection",
			path: "$[2:6:2].a",
			sub: []subcase{
				{
					name: "array",
					data: `[55,41,70,{"a":"bb"}]`,
					want: []match{},
				},
				{
					name: "object",
					data: `{"3":{"a":"aa"}}`,
					want: []match{},
				},
				{
					name: "array multi match",
					data: `[{"a":"xx"},41,{"a":"b1"},{"a":"b2"},{"a":"b3"},{"a":"x2"} ]`,
					want: []match{
						{matchKey{"2"}, "b1"},
						{matchKey{"4"}, "b3"},
					},
				},
			},
		},
		{
			name: "current",
			path: "$.a[@.max]",
			sub: []subcase{
				{
					name: "object",
					data: `{"a":{"max":"3a", "3a":"aa"}, "1":{"a":"1a"}, "x":{"7":"bb"}}`,
					want: "aa",
				},
			},
		},
		{
			name: "multi pick",
			path: "$[1, 3].a",
			sub: []subcase{
				{
					name: "array",
					data: `[55,{"a":"1a"},70,{"a":"bb"}]`,
					want: []match{
						{matchKey{"1"}, "1a"},
						{matchKey{"3"}, "bb"},
					},
				},
				{
					name: "object",
					data: `{"3":{"a":"3a"}, "1":{"a":"1a"}, "x":{"7":"bb"}}`,
					want: []match{
						{matchKey{"1"}, "1a"},
						{matchKey{"3"}, "3a"},
					},
				},
				{
					name: "array partilly matched",
					data: `[55,41,70,{"a":"bb"}]`,
					want: []match{
						{matchKey{"3"}, "bb"},
					},
				},
				{
					name: "object partilly matched",
					data: `{"1":{"a":"aa"}, "3":{}, "x":{"7":"bb"}}`,
					want: []match{
						{matchKey{"1"}, "aa"},
					},
				},
			},
		},
		{
			name: "multi pick star",
			path: "$[1, 3].*",
			sub: []subcase{
				{
					name: "array",
					data: `[55,{"a":"1a"},70,{"b":"bb", "c":"cc"}]`,
					want: []match{
						{matchKey{"1", "a"}, "1a"},
						{matchKey{"3", "b"}, "bb"},
						{matchKey{"3", "c"}, "cc"},
					},
				},
				{
					name: "object",
					data: `{"3":{"a":"3a"}, "1":{"7":"1a"}, "x":{"a":"bb"}}`,
					want: []match{
						{matchKey{"1", "7"}, "1a"},
						{matchKey{"3", "a"}, "3a"},
					},
				},
				{
					name: "array partilly matched",
					data: `[55,41,70,{"a":"bb"}]`,
					want: []match{
						{matchKey{"3", "a"}, "bb"},
					},
				},
				{
					name: "object partilly matched",
					data: `{"1":{"a":"aa", "7":"cc"}, "3":{}, "x":{"7":"bb"}}`,
					want: []match{
						{matchKey{"1", "7"}, "cc"},
						{matchKey{"1", "a"}, "aa"},
					},
				},
			},
		},
		{
			name: "multi pick bracket star",
			path: "$[1, 3][*]",
			sub: []subcase{
				{
					name: "array",
					data: `[55,{"a":"1a"},70,{"b":"bb", "c":"cc"}]`,
					want: []match{
						{matchKey{"1", "a"}, "1a"},
						{matchKey{"3", "b"}, "bb"},
						{matchKey{"3", "c"}, "cc"},
					},
				},
				{
					name: "object",
					data: `{"3":{"a":"3a"}, "1":{"7":"1a"}, "x":{"a":"bb"}}`,
					want: []match{
						{matchKey{"1", "7"}, "1a"},
						{matchKey{"3", "a"}, "3a"},
					},
				},
				{
					name: "array partilly matched",
					data: `[55,41,70,{"b":"bb"}]`,
					want: []match{
						{matchKey{"3", "b"}, "bb"},
					},
				},
				{
					name: "object partilly matched",
					data: `{"1":{"a":"aa", "7":"cc"}, "3":{}, "x":{"7":"bb"}}`,
					want: []match{
						{matchKey{"1", "7"}, "cc"},
						{matchKey{"1", "a"}, "aa"},
					},
				},
			},
		},
		{
			name:    "incomplete",
			path:    "$[3].",
			wantErr: true,
		},
		{
			name:    "mixed bracket",
			path:    "$[3,5:1].",
			wantErr: true,
		},
		{
			name: "mapper",
			path: "$..x",
			sub: []subcase{
				{
					name: "data x",
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
			},
		},
		{
			name: "mapper multipick",
			path: `$..["x", "a"]`,
			sub: []subcase{
				{
					name: "data x",
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
			},
		},
		{
			name: "mapper star",
			path: `$..*`,
			sub: []subcase{
				{
					name: "object",
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
			},
		},
		{
			name: "mapper filter true",
			path: `$..[?true]`,
			sub: []subcase{
				{
					name: "object",
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
			},
		},
		{
			name: "mapper filter a=aa",
			path: `$..[?@.a=="aa"]`,
			sub: []subcase{
				{
					name: "object",
					data: `{"1":{"a":"aa", "b":[1 ,2, 3]}, "3":{}, "x":{"7":"bb"}, "y":{"a":"bb"}}`,
					want: []match{
						{matchKey{``, "1"}, obj{"a": "aa", "b": arr{1., 2., 3.}}},
					},
				},
			},
			extension: gval.Full(),
		},
		{
			name: "mapper filter (a=aa)",
			path: `$..[?(@.a=="aa")]`,
			sub: []subcase{
				{
					name: "object",
					data: `{"1":{"a":"aa", "b":[1 ,2, 3]}, "3":{}, "x":{"7":"bb"}, "y":{"a":"bb"}}`,
					want: []match{
						{matchKey{``, "1"}, obj{"a": "aa", "b": arr{1., 2., 3.}}},
					},
				},
			},
		},
		{
			name: "key value",
			path: `$[?(@.key=="x")].value`,
			sub: []subcase{
				{
					name: "object",
					data: `[{"key": "x","value":"a"},{"key": "y","value":"b"}]`,
					want: []match{
						{matchKey{`0`}, "a"},
					},
				},
			},
		},
		{
			name: "script",
			path: `$.*.value(@=="a")`,
			sub: []subcase{
				{
					name: "object",
					data: `[{"key": "x","value":"a"},{"key": "y","value":"b"}]`,
					want: []match{
						{matchKey{`0`}, true},
						{matchKey{`1`}, false},
					},
				},
			},
		},
		{
			name: "mapper script",
			path: `$..(@=="a")`,
			sub: []subcase{
				{
					name: "object",
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
			},
		},
		{
			name: "mapper select script",
			path: `$.abc.f..["x"](@ == "1")`,
			sub: []subcase{
				{
					name: "object",
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
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			get, err := New(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			for _, sub := range tt.sub {
				t.Run(sub.name, func(t *testing.T) {
					var v interface{}
					err := json.Unmarshal([]byte(sub.data), &v)
					if err != nil {
						t.Fatalf("could not parse json input: %v", err)
					}
					got, err := get(context.Background(), v)
					if (err != nil) != sub.wantErr {
						t.Errorf("New()(*) error = %v, wantErr %v", err, sub.wantErr)
						return
					}

					if sub.wantErr {
						return
					}

					if matchs, ok := sub.want.([]match); ok {
						gotMatchs, ok := got.(Matchs)
						if !ok {
							t.Fatalf("expected multiple results but got %v (%T)", got, got)
						}
						for _, match := range matchs {
							var key *Wildcards
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

					if !reflect.DeepEqual(got, sub.want) {
						t.Fatalf("expected %v, but got %v", sub.want, got)
					}

				})
			}
		})
	}
}
