package jsonpath

import (
	"bytes"
	"context"
)

// Exported for testing purposes
type CollectFullPathsContextKey struct{}

type path interface {
	evaluate(c context.Context, parameter interface{}) (interface{}, error)
	evaluateWithPaths(c context.Context, parameter interface{}) (interface{}, error)
	visitMatchs(c context.Context, r interface{}, visit pathMatcher)
	withPlainSelector(plainSelector) path
	withAmbiguousSelector(ambiguousSelector) path
}

type plainPath []plainSelector

type ambiguousMatcher func(key, v interface{})

func (p plainPath) evaluate(ctx context.Context, root interface{}) (interface{}, error) {
	_, value, err := p.evaluatePath(ctx, root, root)
	return value, err
}

func (p plainPath) evaluateWithPaths(ctx context.Context, root interface{}) (interface{}, error) {
	keys, value, err := p.evaluatePath(ctx, root, root)
	m := map[string]interface{}{}
	m[toJSONPath(keys)] = value
	return m, err
}

func (p plainPath) evaluatePath(ctx context.Context, root, value interface{}) ([]interface{}, interface{}, error) {
	keys := []interface{}{}
	for _, sel := range p {
		k, v, err := sel(ctx, root, value)
		if err != nil {
			return nil, nil, err
		}
		if k != nil {
			keys = append(keys, k)
		}
		value = v
	}
	return keys, value, nil
}

func (p plainPath) matcher(ctx context.Context, r interface{}, match ambiguousMatcher) ambiguousMatcher {
	if len(p) == 0 {
		return match
	}
	return func(k, v interface{}) {
		keys := k
		collectFullPaths := ctx.Value(CollectFullPathsContextKey{})
		ks, res, err := p.evaluatePath(ctx, r, v)
		if b, ok := collectFullPaths.(bool); ok && b {
			keys = append(ks, k)
		}
		if err == nil {
			match(keys, res)
		}
	}
}

func (p plainPath) visitMatchs(ctx context.Context, r interface{}, visit pathMatcher) {
	keys, res, err := p.evaluatePath(ctx, r, r)
	if err == nil {
		visit(keys, res)
	}
}

func (p plainPath) withPlainSelector(selector plainSelector) path {
	return append(p, selector)
}
func (p plainPath) withAmbiguousSelector(selector ambiguousSelector) path {
	return &ambiguousPath{
		parent: p,
		branch: selector,
	}
}

type ambiguousPath struct {
	parent path
	branch ambiguousSelector
	ending plainPath
}

func (p *ambiguousPath) evaluate(ctx context.Context, parameter interface{}) (interface{}, error) {
	matchs := []interface{}{}
	p.visitMatchs(ctx, parameter, func(keys []interface{}, match interface{}) {
		matchs = append(matchs, match)
	})
	return matchs, nil
}

func (p *ambiguousPath) evaluateWithPaths(ctx context.Context, parameter interface{}) (interface{}, error) {
	m := map[string]interface{}{}
	p.visitMatchs(ctx, parameter, func(keys []interface{}, match interface{}) {
		m[toJSONPath(convertPath(keys))] = match
	})
	return m, nil
}

func (p *ambiguousPath) visitMatchs(ctx context.Context, r interface{}, visit pathMatcher) {
	p.parent.visitMatchs(ctx, r, func(keys []interface{}, v interface{}) {
		p.branch(ctx, r, v, p.ending.matcher(ctx, r, visit.matcher(keys)))
	})
}

func (p *ambiguousPath) branchMatcher(ctx context.Context, r interface{}, m ambiguousMatcher) ambiguousMatcher {
	return func(k, v interface{}) {
		p.branch(ctx, r, v, m)
	}
}

func (p *ambiguousPath) withPlainSelector(selector plainSelector) path {
	p.ending = append(p.ending, selector)
	return p
}
func (p *ambiguousPath) withAmbiguousSelector(selector ambiguousSelector) path {
	return &ambiguousPath{
		parent: p,
		branch: selector,
	}
}

type pathMatcher func(keys []interface{}, match interface{})

func (m pathMatcher) matcher(keys []interface{}) ambiguousMatcher {
	return func(key, match interface{}) {
		m(append(keys, key), match)
	}
}

func convertPath(segments []interface{}) []interface{} {
	paths := []interface{}{}
	sCount := len(segments)

	for i, sRaw := range segments {
		if s, ok := sRaw.([]interface{}); ok {
			liIndex := len(s) - 1

			if (liIndex >= 0 && i == sCount-1) || (i > 0 && liIndex < sCount) {
				s = convertPath(append([]interface{}{s[liIndex]}, s[:liIndex]...))
			}

			paths = append(paths, convertPath(s)...)
		} else {
			paths = append(paths, sRaw)
		}
	}

	return paths
}

func toJSONPath(segments []interface{}) string {
	sb := bytes.Buffer{}
	sb.WriteString("$")
	quoteWildcardValues(&sb, convertPath(segments))

	return sb.String()
}
