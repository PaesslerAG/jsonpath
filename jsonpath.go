package jsonpath

import (
	"bytes"
	"fmt"

	"github.com/PaesslerAG/gval"
)

// NewJsonpath returns an selector for given jsonpath
// If the JSON Path plainly the selector returns all matchts as []Match
// A JSON Path is not plain if it contains one of
// [key1, key2 ...], .., *, [min:max], [min:max:step], (? expression)
func NewJsonpath(path string) (gval.Evaluable, error) {
	return lang.NewEvaluable(path)
}

type match func(key string, v interface{})

//Matches of a jsonpath. The key is an Array of the Values used for the wildcards in the jsonpath
type Matches map[*[]string]interface{}

var lang = gval.NewLanguage(
	gval.Base(),
	gval.PrefixExtension('$', parse(getRootEvaluable)),
	gval.PrefixExtension('@', parse(getCurrentEvaluable)),
)

//Language is the jsonpath Language
func Language() gval.Language {
	return lang
}

func (m Matches) String() string {
	sb := bytes.Buffer{}
	sb.WriteString("{")
	sep := ""
	for k, e := range m {
		sb.WriteString(sep)
		sep = " "
		sb.WriteString(fmt.Sprintf("%v", *k))
		sb.WriteString(":")
		sb.WriteString(fmt.Sprintf("%v", e))
	}
	sb.WriteString("}")
	return sb.String()
}
