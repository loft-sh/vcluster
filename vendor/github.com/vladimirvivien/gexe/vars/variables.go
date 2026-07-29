package vars

import (
	"os"
	"regexp"
	"sync"
)

var (
	// varsRegex matches variable pairs of the forms: a=b, c="${d}",  e='f' g="h i j ${k}"
	// Var pairs must not contain space between name and value.
	varsRegex = regexp.MustCompile(`([A-Za-z0-9_]+)=["']?([^"']*?\$\{[A-Za-z0-9_\.\s]+\}[^"']*?|[^"']*)["']?`)
)

// varmap is used to store parsed variables
type varmap struct {
	key   string
	value string
}

// Variables stores a variable map that used for variable expansion
// in parsed commands and other parsed strings.
type Variables struct {
	sync.RWMutex
	err        error
	vars       map[string]string
	escapeChar rune
}

// New construction function to create a new Variables
func New() *Variables {
	return &Variables{vars: make(map[string]string), escapeChar: '\\'}
}

// WithEscapeChar sets the espacape char for the variable
func (v *Variables) WithEscapeChar(r rune) *Variables {
	v.escapeChar = r
	return v
}

// Err surfaces Variables error
func (v *Variables) Err() error {
	return v.err
}

// Envs declares process environment variables with support for
// variable expansion. Each variable must use the form:
//
// <key-name>=<value>
//
// With no space between key name, equal sign, and value,
// i.e. Envs(`GOOS=linux`, `GOARCH=amd64`, `INFO="OS: ${GOOS}, ARC: ${GOARCH}"`)
func (v *Variables) Envs(variables ...string) *Variables {
	if len(variables) == 0 {
		return v
	}
	varmaps := v.parseVars(variables...)
	for _, parsedVar := range varmaps {
		v.SetEnv(parsedVar.key, parsedVar.value)
	}
	return v
}

// SetEnv sets an environment variable key with value.
func (v *Variables) SetEnv(key, value string) *Variables {
	if err := os.Setenv(key, v.ExpandVar(value, v.Val)); err != nil {
		v.err = err
		return v
	}
	return v
}

// Vars declares gexe session variables with support for
// variable expansion. Each variable must use the form:
//
// <key-name>=<value>
//
// With no space between key name, equal sign, and value,
// i.e. Vars(`foo=bar`, `fuzz=${foo}`, `dazz="this is ${fuzz}"`)
func (v *Variables) Vars(variables ...string) *Variables {
	if len(variables) == 0 {
		return v
	}

	varmaps := v.parseVars(variables...)

	// set variables
	for _, parsedVar := range varmaps {
		v.SetVar(parsedVar.key, parsedVar.value)
	}
	return v
}

// SetVar declares a gexe session variable.
func (v *Variables) SetVar(name, value string) *Variables {
	expVar := v.ExpandVar(value, v.Val)
	v.Lock()
	defer v.Unlock()
	v.vars[name] = expVar
	return v
}

// UnsetVar removes a previously set gexe session variable.
func (v *Variables) UnsetVar(name string) *Variables {
	v.Lock()
	defer v.Unlock()
	delete(v.vars, name)
	return v
}

// Val searches for a gexe session variable with provided key, if not found
// searches for an environment variable with that key.
func (v *Variables) Val(key string) string {
	v.RLock()
	defer v.RUnlock()
	if val, ok := v.vars[key]; ok {
		return val
	}
	return os.Getenv(key)
}

// Eval returns the string str with its content expanded
// with variable references i.e. Eval("I am $HOME") returns
// "I am </user/home/path>"
func (v *Variables) Eval(str string) string {
	return v.ExpandVar(str, v.Val)
}

// parseVars parses each var line and maps each key to value into []varmap result.
// This method does not do variable expansion.
func (v *Variables) parseVars(lines ...string) []varmap {
	var result []varmap
	if len(lines) == 0 {
		return []varmap{}
	}

	// each line should contain (<key>)=(<val>) pair
	// matched with expressino which returns match[1] (key) and match[2] (value)
	for _, line := range lines {
		matches := varsRegex.FindStringSubmatch(line)
		if len(matches) >= 3 {
			result = append(result, varmap{key: matches[1], value: matches[2]})
		}
	}
	return result
}
