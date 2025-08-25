package jsonpath

import (
	"context"
	"reflect"
	"testing"
)

func Test_selectValueByReflect(t *testing.T) {
	type args struct {
		name        string
		description string
		data        any
		path        string
		want        any
	}
	tests := []args{
		{
			name:        "test_map",
			description: "test map",
			data:        map[string]interface{}{"a": 1},
			path:        "$.a",
			want:        1,
		},
		{
			name:        "test_slice",
			description: "test slice",
			data:        []interface{}{1, 2, 3},
			path:        "$[1]",
			want:        2,
		},
		{
			name:        "test_map_slice",
			description: "test map-> slice",
			data:        map[string]interface{}{"a": []interface{}{1, 2, 3}},
			path:        "$['a'][1]",
			want:        2,
		},
		{
			name:        "test_map_slice_map",
			description: "test map-> slice-> map",
			data:        map[string][]map[string]any{"a": {{"b": 1}}},
			path:        "$['a'][0]['b']",
			want:        1,
		},
		{
			name:        "test_struct",
			description: "test struct",
			data:        struct{ A int }{A: 1},
			path:        "$['A']",
			want:        1,
		},
		{
			name:        "test_struct_slice",
			description: "test struct-> slice",
			data:        struct{ A []int }{A: []int{1, 2, 3}},
			path:        "$['A'][1]",
			want:        2,
		},
		{
			name:        "test_struct_slice_map",
			description: "test struct-> slice-> map",
			data:        struct{ A []map[string]int }{A: []map[string]int{{"b": 1}}},
			path:        "$['A'][0]['b']",
			want:        1,
		},
		{
			name:        "test_struct_with_json_tag",
			description: "test struct with json tag",
			data: struct {
				A int `json:"a"`
			}{A: 1},
			path: "$['a']",
			want: 1,
		},
		{
			name:        "test_struct_with_json_tag2",
			description: "test struct with json tag",
			data: struct {
				A int `json:"a"`
			}{A: 1},
			path: "$['A']",
			want: 1,
		},
	}
	runCase := ""
	for _, tt := range tests {
		if runCase != "" && tt.name != runCase {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			lang := Language()
			get, err := lang.NewEvaluable(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			result, err := get(context.Background(), tt.data)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("selectValueByReflect() got = %v, want %v", result, tt.want)
			}
		})
	}
}
