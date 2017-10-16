package jsonpath

import (
	"context"
	"fmt"

	"github.com/PaesslerAG/gval"
)

// New returns an selector for given jsonpath
// If the JSON Path plainly the selector returns all Matchs
// A JSON Path is not plain if it contains one of
// [key1, key2 ...], .., *, [min:max], [min:max:step], (? expression)
func New(path string) (gval.Evaluable, error) {
	return lang.NewEvaluable(path)
}

//Get executes given jsonpath on given value
func Get(path string, value interface{}) (interface{}, error) {
	eval, err := lang.NewEvaluable(path)
	if err != nil {
		return nil, err
	}
	return eval(context.Background(), value)
}

type match func(key string, v interface{})

//Matchs of a jsonpath. The key is an Pointer to an Array of the Values used for the wildcards in the jsonpath
type Matchs = map[*Wildcards]interface{}

//Wildcards TODO find correct name
type Wildcards []string

var lang = gval.NewLanguage(
	gval.Base(),
	gval.PrefixExtension('$', parse(getRootEvaluable)),
	gval.PrefixExtension('@', parse(getCurrentEvaluable)),
)

//Language is the jsonpath Language
func Language() gval.Language {
	return lang
}

func (w Wildcards) String() string {
	return fmt.Sprint([]string(w))
}
