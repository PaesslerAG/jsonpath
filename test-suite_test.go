package jsonpath

import (
	"context"
	"os"
	"testing"

	"github.com/PaesslerAG/gval"
	//"github.com/PaesslerAG/jsonpath"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

var knownParsingErrors = map[string]string{
	//`bracket_notation`:                                                   `parsing error: $['key']	:1:3 - 1:8 could not parse string: invalid syntax`,
	//`bracket_notation_on_object_without_key`:                             `parsing error: $['missing']	:1:3 - 1:12 could not parse string: invalid syntax`,
	//`bracket_notation_with_dot`:                                          `parsing error: $['two.some']	:1:3 - 1:13 could not parse string: invalid syntax`,
	//`bracket_notation_with_quoted_dot_wildcard`:                          `parsing error: $['.*']	:1:3 - 1:7 could not parse string: invalid syntax`,
	//`bracket_notation_with_quoted_special_characters_combined`:           `parsing error: $[':@."$,*\'\\']	:1:3 - 1:16 could not parse string: invalid syntax`,
	//`bracket_notation_with_string_including_dot_wildcard`:                `parsing error: $['ni.*']	:1:3 - 1:9 could not parse string: invalid syntax`,
	//`dot_bracket_notation`:                                               `parsing error: $.['key']	:1:3 - 1:4 unexpected "[" while scanning JSON select expected Ident, "." or "*"`,
	//`dot_bracket_notation_with_double_quotes`:                            `parsing error: $.["key"]	:1:3 - 1:4 unexpected "[" while scanning JSON select expected Ident, "." or "*"`,
	//`dot_notation_after_recursive_descent_with_extra_dot`:                `parsing error: $...key	:1:4 - 1:5 unexpected "." while scanning JSON mapper expected "[", Ident or "*"`,
	//`dot_notation_after_union_with_keys`:                                 `parsing error: $['one','three'].key	:1:3 - 1:8 could not parse string: invalid syntax`,
	//`dot_notation_with_double_quotes`:                                    `parsing error: $."key"	:1:3 - 1:8 unexpected String while scanning JSON select expected Ident, "." or "*"`,
	//`dot_notation_with_double_quotes_after_recursive_descent`:            `parsing error: $.."key"	:1:4 - 1:9 unexpected String while scanning JSON mapper expected "[", Ident or "*"`,
	//`dot_notation_with_key_root_literal`:                                 `parsing error: $.$	:1:3 - 1:4 unexpected "$" while scanning JSON select expected Ident, "." or "*"`,
	//`dot_notation_with_number`:                                           `parsing error: $.2	:1:2 - 1:4 unexpected Float while scanning operator`,
	`dot_notation_with_number_-1`:        `parsing error: $.-1	:1:3 - 1:4 unexpected "-" while scanning JSON select expected Ident, "." or "*"`,
	`dot_notation_with_number_on_object`: `parsing error: $.2	:1:2 - 1:4 unexpected Float while scanning operator`,
	//`dot_notation_without_root`:                                          `parsing error: .key	:1:1 - 1:2 unexpected "." while scanning extensions`,
	//`empty`: `parsing error: 	 - 1:1 unexpected EOF while scanning extensions`,
	`filter_expression_with_boolean_and_operator`: `parsing error: $[?(@.key>42 && @.key<44)]	 - 1:16 unknown operator &&`,
	//`filter_expression_with_boolean_and_operator_and_value_false`:        `parsing error: $[?(@.key>0 && false)]	 - 1:15 unknown operator &&`,
	//`filter_expression_with_boolean_and_operator_and_value_true`:         `parsing error: $[?(@.key>0 && true)]	 - 1:15 unknown operator &&`,
	`filter_expression_with_boolean_or_operator`: `parsing error: $[?(@.key>43 || @.key<43)]	 - 1:16 unknown operator ||`,
	//`filter_expression_with_boolean_or_operator_and_value_false`:         `parsing error: $[?(@.key>0 || false)]	 - 1:15 unknown operator ||`,
	//`filter_expression_with_boolean_or_operator_and_value_true`:          `parsing error: $[?(@.key>0 || true)]	 - 1:15 unknown operator ||`,
	//`filter_expression_with_bracket_notation`:                            `parsing error: $[?(@['key']==42)]	:1:7 - 1:12 could not parse string: invalid syntax`,
	//`filter_expression_with_bracket_notation_and_current_object_literal`: `parsing error: $[?(@['@key']==42)]	:1:7 - 1:13 could not parse string: invalid syntax`,
	//`filter_expression_with_different_grouped_operators`:               `parsing error: $[?(@.a && (@.b || @.c))]	 - 1:11 unknown operator &&`,
	//`filter_expression_with_different_ungrouped_operators`:             `parsing error: $[?(@.a && @.b || @.c)]	 - 1:11 unknown operator &&`,
	//`filter_expression_with_dot_notation_with_dash`:                    `parsing error: $[?(@.key-dash == 'value')]	:1:19 - 1:26 could not parse string: invalid syntax`,
	//`filter_expression_with_dot_notation_with_number`:                  `parsing error: $[?(@.2 == 'second')]	:1:6 - 1:8 unexpected Float while scanning parentheses expected ")"`,
	//`filter_expression_with_dot_notation_with_number_on_array`:         `parsing error: $[?(@.2 == 'third')]	:1:6 - 1:8 unexpected Float while scanning parentheses expected ")"`,
	//`filter_expression_with_equals_array`:                              `parsing error: $[?(@.d==["v1","v2"])]	:1:10 - 1:11 unexpected "[" while scanning extensions`,
	//`filter_expression_with_equals_array_for_array_slice_with_range_1`: `parsing error: $[?(@[0:1]==[1])]	:1:13 - 1:14 unexpected "[" while scanning extensions`,
	//`filter_expression_with_equals_array_for_dot_notation_with_star`:  `parsing error: $[?(@.*==[1,2])]	:1:10 - 1:11 unexpected "[" while scanning extensions`,
	//`filter_expression_with_equals_array_or_equals_true`:              `parsing error: $[?(@.d==["v1","v2"] || (@.d == true))]	:1:10 - 1:11 unexpected "[" while scanning extensions`,
	//`filter_expression_with_equals_object`:                            `parsing error: $[?(@.d=={"k":"v"})]	:1:10 - 1:11 unexpected "{" while scanning extensions`,
	//`filter_expression_with_equals_string_with_single_quotes`:         `parsing error: $[?(@.key=='value')]	:1:12 - 1:19 could not parse string: invalid syntax`,
	//`filter_expression_with_in_array_of_values`:                       `parsing error: $[?(@.d in [2, 3])]	:1:9 - 1:11 unexpected Ident while scanning parentheses expected ")"`,
	//`filter_expression_with_in_current_object`:                        `parsing error: $[?(2 in @.d)]	:1:7 - 1:9 unexpected Ident while scanning parentheses expected ")"`,
	//`filter_expression_with_length_function`:                          `parsing error: $[?(@.length() == 4)]	:1:14 - 1:15 unexpected ")" while scanning extensions`,
	//`filter_expression_with_negation_and_equals`:                      `parsing error: $[?(!(@.key==42))]	:1:5 - 1:6 unexpected "!" while scanning extensions`,
	//`filter_expression_with_negation_and_equals_array_or_equals_true`: `parsing error: $[?(!(@.d==["v1","v2"]) || (@.d == true))]	:1:5 - 1:6 unexpected "!" while scanning extensions`,
	//`filter_expression_with_negation_and_less_than`:                `parsing error: $[?(!(@.key<42))]	:1:5 - 1:6 unexpected "!" while scanning extensions`,
	//`filter_expression_with_negation_and_without_value`:            `parsing error: $[?(!@.key)]	:1:5 - 1:6 unexpected "!" while scanning extensions`,
	//`filter_expression_with_not_equals_array_or_equals_true`:       `parsing error: $[?((@.d!=["v1","v2"]) || (@.d == true))]	:1:11 - 1:12 unexpected "[" while scanning extensions`,
	//`filter_expression_with_regular_expression`:                    `parsing error: $[?(@.name=~/hello.*/)]	 - 1:13 unknown operator =~`,
	//`filter_expression_with_regular_expression_from_member`:        `parsing error: $[?(@.name=~/@.pattern/)]	 - 1:13 unknown operator =~`,
	//`filter_expression_with_subpaths`:                              `parsing error: $[?(@.address.city=='Berlin')]	:1:21 - 1:29 could not parse string: invalid syntax`,
	//`filter_expression_with_triple_equal`:   `parsing error: $[?(@.key===42)]	:1:12 - 1:13 unexpected "=" while scanning extensions`,
	//`function_sum`:                          `parsing error: $.data.sum()	:1:12 - 1:13 unexpected ")" while scanning extensions`,
	//`recursive_descent`:                     `parsing error: $..	:1:4 - 1:4 unexpected EOF while scanning JSON mapper expected "[", Ident or "*"`,
	//`recursive_descent_after_dot_notation`:                         `parsing error: $.key..	:1:8 - 1:8 unexpected EOF while scanning JSON mapper expected "[", Ident or "*"`,
	//`union_with_filter`:                     `parsing error: $[?(@.key<3),?(@.key>6)]	:1:13 - 1:14 mixed 63 and 44 in JSON bracket`,
	//`union_with_keys`:                                              `parsing error: $['key','another']	:1:3 - 1:8 could not parse string: invalid syntax`,
	//`union_with_keys_on_object_without_key`:                        `parsing error: $['missing','key']	:1:3 - 1:12 could not parse string: invalid syntax`,
	//`union_with_repeated_matches_after_dot_notation_with_wildcard`: `parsing error: $.*[0,:5]	:1:7 - 1:8 mixed 44 and 58 in JSON bracket`,
	//`union_with_slice_and_number`: `parsing error: $[1:3,4]	:1:6 - 1:7 mixed 58 and 44 in JSON bracket`,
}

var knownEvaluationErrors = map[string]string{
	`bracket_notation_on_object_without_key`:                              `failed to evaluate selector: $['missing'] -> unknown key missing`,
	`bracket_notation_with_NFC_path_on_NFD_key`:                           `unknown key ü`,
	`bracket_notation_with_number_on_string`:                              `unsupported value type string for select, expected map[string]interface{}, []interface{} or Array`,
	`bracket_notation_with_quoted_wildcard_literal_on_object_without_key`: `unknown key *`,
	`current_with_dot_notation`:                                           `unsupported value type <nil> for select, expected map[string]interface{}, []interface{} or Array`,
	`dot_notation_on_array`:                                               `could not select value, invalid key: expected number but got key (string)`,
	`dot_notation_on_array_with_containing_object_matching_key`:           `could not select value, invalid key: expected number but got id (string)`,
	`dot_notation_on_object_without_key`:                                  `unknown key missing`,
	`dot_notation_with_key_named_length_on_array`:                         `could not select value, invalid key: expected number but got length (string)`,
}

var knownDifferences = map[string]struct{}{
	"bracket_notation_after_recursive_descent":                                                             {},
	"bracket_notation_with_number_after_dot_notation_with_wildcard_on_nested_arrays_with_different_length": {},
	"bracket_notation_with_number_on_object":                                                               {},
	"bracket_notation_with_wildcard_after_recursive_descent":                                               {},
	"bracket_notation_with_wildcard_on_object":                                                             {},
	"dot_notation_after_bracket_notation_after_recursive_descent":                                          {},
	"dot_notation_after_recursive_descent":                                                                 {},
	"dot_notation_after_recursive_descent_after_dot_notation":                                              {},
	"dot_notation_with_dash":                                                                               {},
	"dot_notation_with_wildcard_after_recursive_descent":                                                   {},
	"dot_notation_with_wildcard_after_recursive_descent_on_null_value_array":                               {},
	"dot_notation_with_wildcard_on_object":                                                                 {},
	"union_with_keys_after_recursive_descent":                                                              {},
}

type TestSuite struct {
	Queries []*TestCase `yaml:"queries"`
}

type TestCase struct {
	ID              string      `yaml:"id,omitempty"`
	Selector        string      `yaml:"selector,omitempty"`
	Document        interface{} `yaml:"document,omitempty"`
	Consensus       interface{} `yaml:"consensus,omitempty"`
	ScalarConsensus interface{} `yaml:"scalar-consensus,omitempty"`
}

func loadTestSuite() (*TestSuite, error) {
	// read file content
	content, err := os.ReadFile("testdata/regression_suite.yaml")
	if err != nil {
		return nil, err
	}
	// test suite
	var suite TestSuite
	// read yaml file
	err = yaml.Unmarshal(content, &suite)
	if err != nil {
		return nil, err
	}
	return &suite, nil
}

func TestRegressionDocument(t *testing.T) {
	// load test suite
	testSuite, err := loadTestSuite()
	if err != nil {
		t.Errorf("Error loading test suite: %v", err)
	}
	// gval language
	language := gval.NewLanguage(
		Language(),
		gval.Arithmetic(),
	)

	// focused tests
	focused := map[string]struct{}{
		//"array_slice_with_positive_start_and_negative_end_and_range_of_1": {},
	}
	// loop test cases
	for _, testCase := range testSuite.Queries {
		// check focused tests
		if _, ok := focused[testCase.ID]; ok {
			// execute
			executeTestCase(testCase, language, t, focused)
			// next
			continue
		}
		// others
		if len(focused) == 0 {
			// skip NOT_SUPPORTED
			if testCase.Consensus == "NOT_SUPPORTED" || (testCase.Consensus == nil && testCase.ScalarConsensus == nil) {
				continue
			}
			// skip tests known to have parsing errors
			if _, ok := knownParsingErrors[testCase.ID]; ok {
				continue
			}
			// execute
			executeTestCase(testCase, language, t, focused)
		}
	}
}

func executeTestCase(testCase *TestCase, language gval.Language, t *testing.T, focused map[string]struct{}) {
	// execute test case
	t.Run(testCase.ID, func(t *testing.T) {
		// parse selector
		eval, err := language.NewEvaluable(testCase.Selector)
		if err != nil {
			// error
			t.Errorf("failed to parse selector: %s -> %v", testCase.Selector, err)
			// exit
			return
		}
		// skip tests known to have evaluation errors
		if _, ok := knownEvaluationErrors[testCase.ID]; ok && len(focused) == 0 {
			// exit
			return
		}
		// evaluate
		result, err := eval(context.Background(), testCase.Document)
		if err != nil {
			// error
			t.Errorf("failed to evaluate selector: %s -> %v", testCase.Selector, err)
			// exit
			return
		}
		// skip tests known to have differences
		if _, ok := knownDifferences[testCase.ID]; ok && len(focused) == 0 {
			// exit
			return
		}
		// scalar consensus
		scalarDiff := cmp.Diff(testCase.ScalarConsensus, result)
		if scalarDiff == "" {
			// exit
			return
		}
		// consensus
		diff := cmp.Diff(testCase.Consensus, result)
		if diff == "" {
			// exit
			return
		}
		// error
		t.Errorf("invalid result:\n\npath: %s\n\nscalar-consensus: %s\n\nconsensus: %s", testCase.Selector, scalarDiff, diff)
	})
}
