package jsonpath

import "context"

type path interface {
	evaluate(c context.Context, parameter interface{}) (interface{}, error)
	visitMatchs(c context.Context, r interface{}, visit pathMatcher)
	withSelector(single) path
	withMultiSelector(multi) path
}

type simplePath []single

func (p simplePath) evaluate(ctx context.Context, root interface{}) (interface{}, error) {
	return p.evaluatePath(ctx, root, root)
}

func (p simplePath) evaluatePath(ctx context.Context, root, value interface{}) (interface{}, error) {
	var err error
	for _, sel := range p {
		value, err = sel(ctx, root, value)
		if err != nil {
			return nil, err
		}
	}
	return value, nil
}

func (p simplePath) matcher(ctx context.Context, r interface{}, m match) match {
	if len(p) == 0 {
		return m
	}
	return func(k, v interface{}) {
		res, err := p.evaluatePath(ctx, r, v)
		if err == nil {
			m(k, res)
		}
	}
}

func (p simplePath) visitMatchs(c context.Context, r interface{}, visit pathMatcher) {
	visit(nil, r)
	return
}

func (p simplePath) withSelector(selector single) path {
	return append(p, selector)
}
func (p simplePath) withMultiSelector(selector multi) path {
	return &ambiguousPath{
		parent: p,
		branch: selector,
	}
}

type ambiguousPath struct {
	parent path
	branch multi
	ending simplePath
}

func (p *ambiguousPath) evaluate(ctx context.Context, parameter interface{}) (interface{}, error) {
	matchs := []interface{}{}
	p.visitMatchs(ctx, parameter, func(keys []interface{}, match interface{}) {
		matchs = append(matchs, match)
	})
	return matchs, nil
}

func (p *ambiguousPath) visitMatchs(ctx context.Context, r interface{}, visit pathMatcher) {
	p.parent.visitMatchs(ctx, r, func(keys []interface{}, match interface{}) {
		// create a matcher and call p.branch
		p.branch(ctx, r, match, p.ending.matcher(ctx, r, visit.matcher(keys)))

	})
}

func (p *ambiguousPath) branchMatcher(ctx context.Context, r interface{}, m match) match {
	return func(k, v interface{}) {
		p.branch(ctx, r, v, m)
	}
}

func (p *ambiguousPath) withSelector(selector single) path {
	p.ending = append(p.ending, selector)
	return p
}
func (p *ambiguousPath) withMultiSelector(selector multi) path {
	return &ambiguousPath{
		parent: p,
		branch: selector,
	}
}

type ambiguousResults struct {
}

type pathMatcher func(keys []interface{}, match interface{})

func (m pathMatcher) matcher(keys []interface{}) match {
	return func(key, match interface{}) {
		m(append(keys, key), match)
	}
}
