package command

import (
	"reflect"
	"testing"
)

func TestMergeArgs(t *testing.T) {
	tests := []struct {
		name      string
		baseArgs  []string
		extraArgs []string
		expected  []string
	}{
		{
			name:      "no extra args",
			baseArgs:  []string{"--foo=bar"},
			extraArgs: []string{},
			expected:  []string{"--foo=bar"},
		},
		{
			name:      "extra args",
			baseArgs:  []string{"--foo=bar"},
			extraArgs: []string{"--baz=qux"},
			expected:  []string{"--foo=bar", "--baz=qux"},
		},
		{
			name:      "extra args with same flag",
			baseArgs:  []string{"--foo=bar", "--baz=qux"},
			extraArgs: []string{"--baz=quux"},
			expected:  []string{"--foo=bar", "--baz=quux"},
		},
		{
			name:      "extra args with same flag",
			baseArgs:  []string{"--foo=bar", "--baz=qux"},
			extraArgs: []string{"--foo=baz"},
			expected:  []string{"--baz=qux", "--foo=baz"},
		},
		{
			name:      "mutli args with same flag",
			baseArgs:  []string{"--foo=bar", "--baz=qux"},
			extraArgs: []string{"--foo=bax", "--foo=bay"},
			expected:  []string{"--baz=qux", "--foo=bax", "--foo=bay"},
		},
		{
			name:      "nil extra args",
			baseArgs:  []string{"--foo=bar", "--baz=qux"},
			extraArgs: nil,
			expected:  []string{"--foo=bar", "--baz=qux"},
		},
		{
			name:      "no-flags",
			baseArgs:  []string{"/test/path", "arg1", "arg2", "--my-flag=true"},
			extraArgs: []string{},
			expected:  []string{"/test/path", "arg1", "arg2", "--my-flag=true"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := MergeArgs(test.baseArgs, test.extraArgs)
			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf("expected %v, got %v", test.expected, result)
			}
		})
	}
}
