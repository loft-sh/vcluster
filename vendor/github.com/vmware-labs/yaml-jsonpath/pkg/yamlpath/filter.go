/*
 * Copyright 2020 VMware, Inc.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package yamlpath

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type filter func(node, root *yaml.Node) bool

func newFilter(n *filterNode) filter {
	if n == nil {
		return never
	}

	switch n.lexeme.typ {
	case lexemeFilterAt, lexemeRoot:
		path := pathFilterScanner(n)
		return func(node, root *yaml.Node) bool {
			return len(path(node, root)) > 0
		}

	case lexemeFilterEquality, lexemeFilterInequality,
		lexemeFilterGreaterThan, lexemeFilterGreaterThanOrEqual,
		lexemeFilterLessThan, lexemeFilterLessThanOrEqual:
		return comparisonFilter(n)

	case lexemeFilterMatchesRegularExpression:
		return matchRegularExpression(n)

	case lexemeFilterNot:
		f := newFilter(n.children[0])
		return func(node, root *yaml.Node) bool {
			return !f(node, root)
		}

	case lexemeFilterOr:
		f1 := newFilter(n.children[0])
		f2 := newFilter(n.children[1])
		return func(node, root *yaml.Node) bool {
			return f1(node, root) || f2(node, root)
		}

	case lexemeFilterAnd:
		f1 := newFilter(n.children[0])
		f2 := newFilter(n.children[1])
		return func(node, root *yaml.Node) bool {
			return f1(node, root) && f2(node, root)
		}

	case lexemeFilterBooleanLiteral:
		b, err := strconv.ParseBool(n.lexeme.val)
		if err != nil {
			panic(err) // should not happen
		}
		return func(node, root *yaml.Node) bool {
			return b
		}

	default:
		return never
	}
}

func never(node, root *yaml.Node) bool {
	return false
}

func comparisonFilter(n *filterNode) filter {
	compare := func(b bool) bool {
		var c comparison
		if b {
			c = compareEqual
		} else {
			c = compareIncomparable
		}
		return n.lexeme.comparator()(c)
	}
	return nodeToFilter(n, func(l, r typedValue) bool {
		if !l.typ.compatibleWith(r.typ) {
			return compare(false)
		}
		switch l.typ {
		case booleanValueType:
			return compare(equalBooleans(l.val, r.val))

		case nullValueType:
			return compare(equalNulls(l.val, r.val))

		default:
			return n.lexeme.comparator()(compareNodeValues(l, r))
		}
	})
}

var x, y typedValue

func init() {
	x = typedValue{stringValueType, "x"}
	y = typedValue{stringValueType, "y"}
}

func nodeToFilter(n *filterNode, accept func(typedValue, typedValue) bool) filter {
	lhsPath := newFilterScanner(n.children[0])
	rhsPath := newFilterScanner(n.children[1])
	return func(node, root *yaml.Node) (result bool) {
		// perform a set-wise comparison of the values in each path
		match := false
		for _, l := range lhsPath(node, root) {
			for _, r := range rhsPath(node, root) {
				if !accept(l, r) {
					return false
				}
				match = true
			}
		}
		return match
	}
}

func equalBooleans(l, r string) bool {
	// Note: the YAML parser and our JSONPath lexer both rule out invalid boolean literals such as tRue.
	return strings.EqualFold(l, r)
}

func equalNulls(l, r string) bool {
	// Note: the YAML parser and our JSONPath lexer both rule out invalid null literals such as nUll.
	return true
}

// filterScanner is a function that returns a slice of typed values from either a filter literal or a path expression
// which refers to either the current node or the root node. It is used in filter comparisons.
type filterScanner func(node, root *yaml.Node) []typedValue

func emptyScanner(*yaml.Node, *yaml.Node) []typedValue {
	return []typedValue{}
}

func newFilterScanner(n *filterNode) filterScanner {
	switch {
	case n == nil:
		return emptyScanner

	case n.isItemFilter():
		return pathFilterScanner(n)

	case n.isLiteral():
		return literalFilterScanner(n)

	default:
		return emptyScanner
	}
}

func pathFilterScanner(n *filterNode) filterScanner {
	var at bool
	switch n.lexeme.typ {
	case lexemeFilterAt:
		at = true
	case lexemeRoot:
		at = false
	default:
		panic("false precondition")
	}
	subpath := ""
	for _, lexeme := range n.subpath {
		subpath += lexeme.val
	}
	path, err := NewPath(subpath)
	if err != nil {
		return emptyScanner
	}
	return func(node, root *yaml.Node) []typedValue {
		if at {
			return values(path.Find(node))
		}
		return values(path.Find(root))
	}
}

type valueType int

const (
	unknownValueType valueType = iota
	stringValueType
	intValueType
	floatValueType
	booleanValueType
	nullValueType
	regularExpressionValueType
)

func (vt valueType) isNumeric() bool {
	return vt == intValueType || vt == floatValueType
}

func (vt valueType) compatibleWith(vt2 valueType) bool {
	return vt.isNumeric() && vt2.isNumeric() || vt == vt2 || vt == stringValueType && vt2 == regularExpressionValueType
}

type typedValue struct {
	typ valueType
	val string
}

const (
	nullTag  = "!!null"
	boolTag  = "!!bool"
	strTag   = "!!str"
	intTag   = "!!int"
	floatTag = "!!float"
)

func typedValueOfNode(node *yaml.Node) typedValue {
	var t valueType = unknownValueType
	if node.Kind == yaml.ScalarNode {
		switch node.ShortTag() {
		case nullTag:
			t = nullValueType

		case boolTag:
			t = booleanValueType

		case strTag:
			t = stringValueType

		case intTag:
			t = intValueType

		case floatTag:
			t = floatValueType
		}
	}

	return typedValue{
		typ: t,
		val: node.Value,
	}
}

func newTypedValue(t valueType, v string) typedValue {
	return typedValue{
		typ: t,
		val: v,
	}
}

func typedValueOfString(s string) typedValue {
	return newTypedValue(stringValueType, s)
}

func typedValueOfInt(i string) typedValue {
	return newTypedValue(intValueType, i)
}

func typedValueOfFloat(f string) typedValue {
	return newTypedValue(floatValueType, f)
}

func values(nodes []*yaml.Node, err error) []typedValue {
	if err != nil {
		panic(fmt.Errorf("unexpected error: %v", err)) // should never happen
	}
	v := []typedValue{}
	for _, n := range nodes {
		v = append(v, typedValueOfNode(n))
	}
	return v
}

func literalFilterScanner(n *filterNode) filterScanner {
	v := n.lexeme.literalValue()
	return func(node, root *yaml.Node) []typedValue {
		return []typedValue{v}
	}
}

func matchRegularExpression(parseTree *filterNode) filter {
	return nodeToFilter(parseTree, stringMatchesRegularExpression)
}

func stringMatchesRegularExpression(s, expr typedValue) bool {
	if s.typ != stringValueType || expr.typ != regularExpressionValueType {
		return false // can't compare types so return false
	}
	re, _ := regexp.Compile(expr.val) // regex already compiled during lexing
	return re.Match([]byte(s.val))
}
