package jsonpath

import (
	"context"

	"github.com/PaesslerAG/gval"
)

type parser struct {
	gval.Parser
	single
	multis []multi
}

//multi evaluate wildcard
type multi func(c context.Context, r, v interface{}, m match)

//single evaluate exactly one result
type single func(c context.Context, r, v interface{}) (interface{}, error)

func (p *parser) newSingleStage(next single) {
	if p.single == nil {
		p.single = next
		return
	}
	last := p.single
	p.single = func(c context.Context, r, v interface{}) (interface{}, error) {
		v, err := last(c, r, v)
		if err != nil {
			return nil, err
		}
		return next(c, r, v)
	}
}

func (p *parser) newMultiStage(next multi) {
	if p.single != nil {
		s := p.single
		p.single = nil
		p.multis = append(p.multis, func(c context.Context, r, v interface{}, m match) {
			v, err := s(c, r, v)
			if err != nil {
				return
			}
			next(c, r, v, m)
		})
		return
	}
	p.multis = append(p.multis, next)
}

func (p *parser) evaluable() gval.Evaluable {
	if p.multis == nil {
		return p.single.evaluable
	}
	multis := p.multis
	if p.single != nil {
		s := p.single
		inner := multis[len(multis)-1]
		multis[len(multis)-1] = func(c context.Context, r, v interface{}, m match) {
			inner(c, r, v, func(key string, v interface{}) {
				v, err := s(c, r, v)
				if err != nil {
					return
				}
				m(key, v)
			})
		}

	}
	return func(c context.Context, v interface{}) (interface{}, error) {
		res := Matches{}
		evaluateMultis(c, v, v, nil, multis, &res)

		return res, nil
	}
}

func evaluateMultis(c context.Context, r, v interface{}, keys []string, ms []multi, matches *Matches) {
	if len(ms) == 0 {
		(*matches)[&keys] = v
		return
	}
	ms[0](c, r, v, func(key string, v interface{}) {
		evaluateMultis(c, r, v, append(keys, key), ms[1:], matches)
	})
}

func (s single) evaluable(c context.Context, v interface{}) (interface{}, error) {
	return s(c, v, v)
}
