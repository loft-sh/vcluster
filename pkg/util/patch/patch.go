package patch

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/samber/lo"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Patch helps with navigating generic map[string]interface{} which is used by unstructured objects.
// It's called patch, because we only operate on patches from virtual to host and back and these functions
// help to keep it as generic as possible.
type Patch map[string]interface{}

type PathValue struct {
	Parent *PathValue

	Value interface{}

	Index int
	Key   string

	Path string
}

func (p Patch) DeepCopy() Patch {
	if p == nil {
		return nil
	}

	out, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	outPatch := map[string]interface{}{}
	err = json.Unmarshal(out, &outPatch)
	if err != nil {
		panic(err)
	}

	return outPatch
}

func (p Patch) IsEmpty() bool {
	return len(p) == 0
}

func (p Patch) Clear() {
	for k := range p {
		delete(p, k)
	}
}

type translateFn func(path string, val interface{}, exists bool) (interface{}, error)

func (p Patch) MustTranslate(path string, translate translateFn) {
	err := p.Translate(path, translate)
	if err != nil {
		panic(err)
	}
}

// Translate changes values on the given path.
func (p Patch) Translate(path string, translate translateFn) error {
	parsedPath, err := parsePathWithIndexing(path, true)
	if err != nil {
		panic(err)
	}
	if len(parsedPath) == 0 {
		retVal, err := translate("", map[string]interface{}(p), true)
		if err != nil {
			return err
		}

		patchRaw, err := json.Marshal(retVal)
		if err != nil {
			return err
		}

		return json.Unmarshal(patchRaw, &p)
	}

	// get last map / array
	curs, ok := p.getValue(parsedPath, 1)

	if !ok {
		return nil
	}

	// get last element
	for _, cur := range curs {
		segment := parsedPath[len(parsedPath)-1]

		switch t := cur.Value.(type) {
		case []interface{}:
			segment = trimBracketsPair(segment)
			if segment == "*" {
				for k := range t {
					t[k], err = translate(addPathElement(cur.Path, strconv.Itoa(k)), t[k], true)
					if err != nil {
						return err
					}
				}
				continue
			}

			index, err := strconv.Atoi(segment)
			if err != nil {
				return nil
			}

			if len(t) <= index {
				return nil
			}

			ret, err := translate(addPathElement(cur.Path, segment), t[index], true)
			if err != nil {
				return err
			}

			t[index] = ret

		case map[string]interface{}:
			switch {
			case segment == "[*]":
				for k := range t {
					t[k], err = translate(addPathElement(cur.Path, k), t[k], true)
					if err != nil {
						return err
					}
				}
			case isBracketEnclosed(segment): // a.path.to.some["segment"] case
				key := trimBracketsPair(segment)
				if key == "" {
					return fmt.Errorf("empty key in bracket notation in path %q", segment)
				}
				v, ok := t[key]
				valueFromExpression, err := translate(cur.Path, v, ok)
				if err != nil {
					return fmt.Errorf("translate value for key %q in path %q: %w", key, cur.Path, err)
				}
				if valueFromExpression == nil {
					p.Delete(JoinPath(cur.Path, key))
					continue
				}
				t[key] = valueFromExpression

			default: // a.path.to.some.segment case
				if val, ok := t[segment]; ok {
					t[segment], err = translate(JoinPath(cur.Path, segment), val, ok)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (p Patch) DeleteAllExcept(path string, except ...string) {
	parsedPath, err := parsePath(path)
	if err != nil {
		panic(err)
	}

	// get last map / array
	curs, ok := p.getValue(parsedPath, 0)
	if !ok {
		return
	}

	// we only support maps here for now.
	for _, cur := range curs {
		if t, ok := cur.Value.(map[string]interface{}); ok {
			for k := range t {
				if lo.Contains(except, k) {
					continue
				}

				delete(t, k)
			}
		}
	}

	// TODO: add support for arrays
}

func (p Patch) Delete(path string) {
	parsedPath, err := parsePath(path)
	if err != nil {
		panic(err)
	} else if len(parsedPath) == 0 {
		return
	}

	// get last map / array
	curs, ok := p.getValue(parsedPath, 1)
	if !ok {
		return
	}

	// delete last element, we only support maps here for now.
	for _, cur := range curs {
		segment := parsedPath[len(parsedPath)-1]
		if segment == "*" {
			if t, ok := cur.Value.(map[string]interface{}); ok {
				for k := range t {
					delete(t, k)
				}
				return
			}
		}

		if t, ok := cur.Value.(map[string]interface{}); ok {
			delete(t, segment)
		}
	}

	// TODO: add support for arrays
}

func (p Patch) String(path string) (string, bool) {
	val, ok := p.Value(path)
	if !ok {
		return "", false
	}

	strVal, ok := val.(string)
	return strVal, ok
}

// Set sets a value and will create objects along the way. Wildcard paths are not supported here.
func (p Patch) Set(path string, value interface{}) {
	// parse the path
	parsedPath, err := parsePath(path)
	if err != nil {
		panic(err)
	} else if len(parsedPath) == 0 {
		return
	}

	// walk through the patch
	curs, ok := p.getValueAndCreate(parsedPath, 1)
	if !ok {
		return
	}

	// get last element
	for _, curPathValue := range curs {
		segment := parsedPath[len(parsedPath)-1]
		switch cur := curPathValue.Value.(type) {
		case map[string]interface{}:
			if segment == "*" {
				for k := range cur {
					cur[k] = value
				}

				continue
			}

			cur[segment] = value
		case []interface{}:
			if segment == "*" {
				for k := range cur {
					cur[k] = value
				}

				continue
			}

			index, err := strconv.Atoi(segment)
			if err != nil {
				return
			}

			if len(cur) <= index {
				for i := len(cur); i <= index; i++ {
					cur = append(cur, nil)
				}

				switch parent := curPathValue.Parent.Value.(type) {
				case []interface{}:
					parent[curPathValue.Index] = cur
				case map[string]interface{}:
					parent[curPathValue.Key] = cur
				}
			}

			cur[index] = value
		}
	}
}

func (p Patch) Has(path string) bool {
	_, ok := p.Value(path)
	return ok
}

func Value[T any](path string, patches ...Patch) (T, bool) {
	for _, p := range patches {
		val, ok := p.Value(path)
		if !ok {
			continue
		}

		ret, ok := val.(T)
		if !ok {
			continue
		}

		return ret, true
	}

	var ret T
	return ret, false
}

func (p Patch) Value(path string) (interface{}, bool) {
	parsedPath, err := parsePath(path)
	if err != nil {
		panic(err)
	}

	vals, ok := p.getValue(parsedPath, 0)
	if !ok || len(vals) == 0 {
		return nil, false
	}

	return vals[0].Value, true
}

func (p Patch) Apply(obj client.Object) error {
	patchBytes, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal patch bytes: %w", err)
	}

	unstructuredMap, err := ConvertObjectToPatch(obj)
	if err != nil {
		return fmt.Errorf("to unstructured: %w", err)
	}

	objBytes, err := json.Marshal(unstructuredMap)
	if err != nil {
		return fmt.Errorf("marshal object: %w", err)
	}

	afterObjBytes, err := jsonpatch.MergePatch(objBytes, patchBytes)
	if err != nil {
		return fmt.Errorf("apply merge patch: %w", err)
	}

	afterObjMap := map[string]interface{}{}
	err = json.Unmarshal(afterObjBytes, &afterObjMap)
	if err != nil {
		return fmt.Errorf("unmarshal applied object: %w", err)
	}

	err = ConvertPatchToObject(afterObjMap, obj)
	if err != nil {
		return err
	}

	return nil
}

func (p Patch) getValue(parsedPath []string, index int) ([]PathValue, bool) {
	return nextValue(parsedPath, index, &PathValue{
		Value: map[string]interface{}(p),
	}, false)
}

func (p Patch) getValueAndCreate(parsedPath []string, index int) ([]PathValue, bool) {
	return nextValue(parsedPath, index, &PathValue{
		Value: map[string]interface{}(p),
	}, true)
}

func nextValue(parsedPath []string, index int, cur *PathValue, create bool) ([]PathValue, bool) {
	if len(parsedPath) <= index {
		return []PathValue{*cur}, true
	}

	firstPath := trimBracketsPair(parsedPath[0])
	switch val := cur.Value.(type) {
	case map[string]interface{}:
		if firstPath == "*" {
			retVals := make([]PathValue, 0, len(val))
			for k := range val {
				retVal, ok := nextValue(parsedPath[1:], index, &PathValue{
					Parent: cur,
					Value:  val[k],
					Key:    k,
					Path:   addPathElement(cur.Path, k),
				}, create)
				if ok {
					retVals = append(retVals, retVal...)
				}
			}
			if len(retVals) == 0 {
				return nil, false
			}

			return retVals, true
		}

		mapValue, ok := val[firstPath]
		if !ok && !create {
			return nil, false
		} else if create && (!ok || mapValue == nil) {
			val[firstPath] = createValue(parsedPath[1:])
			mapValue = val[firstPath]
		}

		return nextValue(parsedPath[1:], index, &PathValue{
			Parent: cur,
			Value:  mapValue,
			Key:    firstPath,
			Path:   addPathElement(cur.Path, firstPath),
		}, create)
	case []interface{}:
		// try to match all
		if firstPath == "*" {
			retVals := make([]PathValue, 0, len(val))
			for i := range val {
				retVal, ok := nextValue(parsedPath[1:], index, &PathValue{
					Parent: cur,
					Value:  val[i],
					Index:  i,
					Path:   addPathElement(cur.Path, strconv.Itoa(i)),
				}, create)
				if ok {
					retVals = append(retVals, retVal...)
				}
			}
			if len(retVals) == 0 {
				return nil, false
			}

			return retVals, true
		}

		// try to get index
		indexSegment, err := strconv.Atoi(firstPath)
		if err != nil {
			return nil, false
		}

		if len(val) <= indexSegment {
			if !create {
				return nil, false
			}

			for i := len(val); i < indexSegment; i++ {
				val = append(val, nil)
			}
			val = append(val, createValue(parsedPath[1:]))
		}

		arrVal := val[indexSegment]
		if create && arrVal == nil {
			val[indexSegment] = createValue(parsedPath[1:])
			arrVal = val[indexSegment]
		}

		return nextValue(parsedPath[1:], index, &PathValue{
			Parent: cur,
			Value:  arrVal,
			Index:  indexSegment,
			Path:   addPathElement(cur.Path, firstPath),
		}, create)
	}

	return nil, false
}

func createValue(pathSegment []string) interface{} {
	if len(pathSegment) == 0 {
		return map[string]interface{}{}
	}

	segment := trimBracketsPair(pathSegment[0])
	intVal, err := strconv.Atoi(segment)
	if err == nil {
		newVal := make([]interface{}, 0, intVal+1)
		for i := 0; i <= intVal; i++ {
			newVal = append(newVal, nil)
		}
		return newVal
	}

	return map[string]interface{}{}
}

func addPathElement(root, next string) string {
	if strings.Contains(next, ".") || strings.Contains(next, "[") || strings.Contains(next, "]") {
		if !strings.HasPrefix(next, "\"") {
			next = "\"" + next
		}
		if !strings.HasSuffix(next, "\"") {
			next += "\""
		}
	}

	return JoinPath(root, next)
}

func JoinPath(root, next string) string {
	if root == "" {
		return next
	}
	return root + "." + next
}

func trimBracketsPair(segment string) string {
	if isBracketEnclosed(segment) {
		return segment[1 : len(segment)-1]
	}
	return segment
}

func isBracketEnclosed(segment string) bool {
	if len(segment) < 2 {
		return false
	}
	return segment[0] == '[' && segment[len(segment)-1] == ']'
}
