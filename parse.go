package jsonpath

import (
	"fmt"
	"math"
	"text/scanner"

	"github.com/PaesslerAG/gval"
)

func parse(s single) func(p *gval.Parser) (r gval.Evaluable, err error) {
	return func(p *gval.Parser) (r gval.Evaluable, err error) {
		scn := &parser{Parser: *p, single: s}
		err = scn.parsePath()

		*p = scn.Parser
		if err != nil {
			return nil, err
		}
		return scn.evaluable(), nil
	}
}

func (scn *parser) parsePath() error {
	switch scn.Scan() {
	case '.':
		return scn.parseSelect()
	case '[':
		keys, seperator, err := scn.parseBracket()

		if err != nil {
			return err
		}

		switch seperator {
		case ':':
			if len(keys) > 3 {
				return fmt.Errorf("range query has at least the parameter [min:max:step]")
			}
			keys = append(keys, []gval.Evaluable{
				scn.Const(0), scn.Const(float64(math.MaxInt32)), scn.Const(1)}[len(keys):]...)
			scn.newMultiStage(getRangeEvaluable(keys[0], keys[1], keys[2]))
		case '?':
			if len(keys) != 1 {
				return fmt.Errorf("filter needs exactly one key")
			}
			scn.newMultiStage(filterEvaluable(keys[0]))
		default:
			if len(keys) == 1 {
				scn.newSingleStage(getSelectEvaluable(keys[0]))
			} else {
				scn.newMultiStage(getMultiSelectEvaluable(keys))
			}
		}
		return scn.parsePath()
	default:
		scn.Camouflage("jsonpath", '.', '[')
		return nil
	}
}

func (scn *parser) parseSelect() error {
	scan := scn.Scan()
	switch scan {
	case scanner.Ident:
		scn.newSingleStage(getSelectEvaluable(scn.Const(scn.TokenText())))
		return scn.parsePath()
	case '.':
		scn.newMultiStage(mapperEvaluable)
		return scn.parseMapper()
	case '*':
		scn.newMultiStage(starEvaluable)
		return scn.parsePath()
	default:
		return scn.Expected("JSON select", scanner.Ident, '.', '*')
	}
}

func (scn *parser) parseBracket() (keys []gval.Evaluable, seperator rune, err error) {
	for {
		scan := scn.Scan()
		skipScan := false
		switch scan {
		case '?':
			skipScan = true
		case ':':
			i := float64(0)
			if len(keys) == 1 {
				i = math.MaxInt32
			}
			keys = append(keys, scn.Const(i))
			skipScan = true
		case '*':
			if scn.Scan() != ']' {
				return nil, 0, scn.Expected("JSON bracket star", ']')
			}
			return []gval.Evaluable{}, 0, nil
		case ']':
			if seperator == ':' {
				skipScan = true
				break
			}
			fallthrough
		default:
			scn.Camouflage("jsonpath brackets")
			key, err := scn.ParseExpression()
			if err != nil {
				return nil, 0, err
			}
			keys = append(keys, key)
		}
		if !skipScan {
			scan = scn.Scan()
		}
		if seperator == 0 {
			seperator = scan
		}
		switch scan {
		case ':', ',':
		case ']':
			return
		case '?':
			if len(keys) != 0 {
				return nil, 0, scn.Expected("JSON filter", ']')
			}
		default:
			return nil, 0, scn.Expected("JSON bracket separator", ':', ',')
		}
		if seperator != scan {
			return nil, 0, fmt.Errorf("mixed %v and %v in JSON bracket", seperator, scan)
		}
	}
}

func (scn *parser) parseMapper() error {
	scan := scn.Scan()
	switch scan {
	case scanner.Ident:
		scn.newSingleStage(getSelectEvaluable(scn.Const(scn.TokenText())))
	case '[':
		keys, seperator, err := scn.parseBracket()

		if err != nil {
			return err
		}
		switch seperator {
		case ':':
			return fmt.Errorf("mapper can not be combined with range query")
		case '?':
			if len(keys) != 1 {
				return fmt.Errorf("filter needs exactly one key")
			}
			scn.newMultiStage(filterEvaluable(keys[0]))
		default:
			scn.newMultiStage(getMultiSelectEvaluable(keys))
		}
	case '*':
		scn.newMultiStage(starEvaluable)
	default:
		return scn.Expected("JSON mapper", '[', scanner.Ident, '*')
	}
	return scn.parsePath()
}
