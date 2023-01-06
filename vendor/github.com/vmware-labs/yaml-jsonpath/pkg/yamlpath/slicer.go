/*
 * Copyright 2020 VMware, Inc.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package yamlpath

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func slice(index string, length int) ([]int, error) {
	if union := strings.Split(index, ","); len(union) > 1 {
		combination := []int{}
		for i, idx := range union {
			sl, err := slice(idx, length)
			if err != nil {
				return nil, fmt.Errorf("error in union member %d: %s", i, err)
			}
			combination = append(combination, sl...)
		}
		return combination, nil
	}

	index = strings.TrimSpace(index)

	if index == "*" {
		return indices(0, length, 1, length), nil
	}

	subscr := strings.Split(index, ":")
	if len(subscr) > 3 {
		return nil, errors.New("malformed array index, too many colons")
	}
	type subscript struct {
		present bool
		value   int
	}
	var subscripts []subscript = []subscript{{false, 0}, {false, 0}, {false, 0}}
	const (
		sFrom = iota
		sTo
		sStep
	)
	for i, s := range subscr {
		s = strings.TrimSpace(s)
		if s != "" {
			n, err := strconv.Atoi(s)
			if err != nil {
				return nil, errors.New("non-integer array index")
			}
			subscripts[i] = subscript{
				present: true,
				value:   n,
			}
		}
	}

	// pick out the case of a single subscript first since the "to" value needs special-casing
	if len(subscr) == 1 {
		if !subscripts[sFrom].present {
			return nil, errors.New("array index missing")
		}
		from := subscripts[sFrom].value
		if from < 0 {
			from += length
		}
		return indices(from, from+1, 1, length), nil
	}

	var from, to, step int

	if subscripts[sStep].present {
		step = subscripts[sStep].value
		if step == 0 {
			return nil, errors.New("array index step value must be non-zero")
		}
	} else {
		step = 1
	}

	if subscripts[sFrom].present {
		from = subscripts[sFrom].value
		if from < 0 {
			from += length
		}
	} else {
		if step > 0 {
			from = 0
		} else {
			from = length - 1
		}
	}

	if subscripts[sTo].present {
		to = subscripts[sTo].value
		if to < 0 {
			to += length
		}
	} else {
		if step > 0 {
			to = length
		} else {
			to = -1
		}
	}

	return indices(from, to, step, length), nil
}

func indices(from, to, step, length int) []int {
	slice := []int{}
	if step > 0 {
		if from < 0 {
			from = 0 // avoid CPU attack
		}
		if to > length {
			to = length // avoid CPU attack
		}
		for i := from; i < to; i += step {
			if 0 <= i && i < length {
				slice = append(slice, i)
			}
		}
	} else if step < 0 {
		if from > length {
			from = length // avoid CPU attack
		}
		if to < -1 {
			to = -1 // avoid CPU attack
		}
		for i := from; i > to; i += step {
			if 0 <= i && i < length {
				slice = append(slice, i)
			}
		}
	}
	return slice
}
