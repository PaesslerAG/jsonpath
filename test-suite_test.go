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
	`dot_notation_with_number_-1`:                 `parsing error: $.-1	:1:3 - 1:4 unexpected "-" while scanning JSON select expected Ident, "." or "*"`,
	`dot_notation_with_number_on_object`:          `parsing error: $.2	:1:2 - 1:4 unexpected Float while scanning operator`,
	`filter_expression_with_boolean_and_operator`: `parsing error: $[?(@.key>42 && @.key<44)]	 - 1:16 unknown operator &&`,
	`filter_expression_with_boolean_or_operator`:  `parsing error: $[?(@.key>43 || @.key<43)]	 - 1:16 unknown operator ||`,
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
