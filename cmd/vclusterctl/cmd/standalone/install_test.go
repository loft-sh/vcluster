package standalone

import (
	"reflect"
	"testing"
)

func TestParseExtraEnv(t *testing.T) {
	tests := []struct {
		name      string
		input     []string
		expected  map[string]string
		expectErr bool
	}{
		{
			name: "valid multiple env variables",
			input: []string{
				"KEY1=value1",
				"KEY2=value2",
			},
			expected: map[string]string{"KEY1": "value1", "KEY2": "value2"},
		},
		{
			name:      "invalid format missing '='",
			input:     []string{"KEYvalue"},
			expectErr: true,
		},
		{
			name:     "empty input slice",
			input:    []string{},
			expected: map[string]string{},
		},
		{
			name:     "invalid format key only",
			input:    []string{"KEY="},
			expected: map[string]string{"KEY": ""},
		},
		{
			name:      "invalid format value only",
			input:     []string{"=value"},
			expectErr: true,
		},
		{
			name:     "multiple spaces",
			input:    []string{"KEY=values=with=separator"},
			expected: map[string]string{"KEY": "values=with=separator"},
		},
		{
			name: "valid env variable with spaces",
			input: []string{
				"KEY1=value with spaces",
			},
			expected: map[string]string{"KEY1": "value with spaces"},
		},
		{
			name: "valid env variable with special characters",
			input: []string{
				"KEY1=value_with_special!@#$%^&*()",
			},
			expected: map[string]string{"KEY1": "value_with_special!@#$%^&*()"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseExtraEnv(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("parseExtraEnv() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseExtraEnv() got = %v, want = %v", result, tt.expected)
			}
		})
	}
}
