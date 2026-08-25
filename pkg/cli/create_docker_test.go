package cli

import "testing"

func TestExtractContainerID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean 64-char ID",
			input:    "23e586d93b4ca2d51a8d21e27ad14807b1fefb4445adc275adcb03093a76b216\n",
			expected: "23e586d93b4ca2d51a8d21e27ad14807b1fefb4445adc275adcb03093a76b216",
		},
		{
			name:     "warning prepended on stderr/stdout",
			input:    "WARNING: The requested image's platform (linux/amd64) does not match the detected host platform (linux/arm64/v8)\n23e586d93b4ca2d51a8d21e27ad14807b1fefb4445adc275adcb03093a76b216\n",
			expected: "23e586d93b4ca2d51a8d21e27ad14807b1fefb4445adc275adcb03093a76b216",
		},
		{
			name:     "unable to find image warning prepended",
			input:    "Unable to find image 'alpine:latest' locally\nlatest: Pulling from library/alpine\nWARNING: The requested image's platform (linux/amd64) does not match\n23e586d93b4ca2d51a8d21e27ad14807b1fefb4445adc275adcb03093a76b216\n",
			expected: "23e586d93b4ca2d51a8d21e27ad14807b1fefb4445adc275adcb03093a76b216",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractContainerID(tt.input)
			if got != tt.expected {
				t.Errorf("extractContainerID() = %q, want %q", got, tt.expected)
			}
		})
	}
}
