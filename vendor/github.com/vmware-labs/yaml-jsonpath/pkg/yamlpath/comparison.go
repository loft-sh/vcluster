/*
 * Copyright 2020 VMware, Inc.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package yamlpath

import "strconv"

type comparison int

const (
	compareLessThan comparison = iota
	compareEqual
	compareGreaterThan
	compareIncomparable
)

type orderingOperator string

const (
	operatorLessThan           orderingOperator = "<"
	operatorLessThanOrEqual    orderingOperator = "<="
	operatorGreaterThan        orderingOperator = ">"
	operatorGreaterThanOrEqual orderingOperator = ">="
)

func (o orderingOperator) String() string {
	return string(o)
}

type comparator func(comparison) bool

func equal(c comparison) bool {
	return c == compareEqual
}

func notEqual(c comparison) bool {
	return c != compareEqual
}

func greaterThan(c comparison) bool {
	return c == compareGreaterThan
}

func greaterThanOrEqual(c comparison) bool {
	return c == compareGreaterThan || c == compareEqual
}

func lessThan(c comparison) bool {
	return c == compareLessThan
}

func lessThanOrEqual(c comparison) bool {
	return c == compareLessThan || c == compareEqual
}

func compareStrings(a, b string) comparison {
	if a == b {
		return compareEqual
	}
	return compareIncomparable
}

func compareFloat64(lhs, rhs float64) comparison {
	if lhs < rhs {
		return compareLessThan
	}
	if lhs > rhs {
		return compareGreaterThan
	}
	return compareEqual
}

// compareNodeValues compares two values each of which may be a string, integer, or float
func compareNodeValues(lhs, rhs typedValue) comparison {
	if lhs.typ.isNumeric() && rhs.typ.isNumeric() {
		return compareFloat64(mustParseFloat64(lhs.val), mustParseFloat64(rhs.val))
	}
	if (lhs.typ != stringValueType && !lhs.typ.isNumeric()) || (rhs.typ != stringValueType && !rhs.typ.isNumeric()) {
		panic("invalid type of value passed to compareNodeValues") // should never happen
	}
	return compareStrings(lhs.val, rhs.val)
}

func mustParseFloat64(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic("invalid numeric value " + s) // should never happen
	}
	return f
}
