package list

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
)

func TestPrintData(t *testing.T) {
	headers := []string{"Project"}
	items := []map[string]string{
		{"Name": "test-project1"},
		{"Name": "test-project2"},
	}

	tmpFile, err := os.CreateTemp("", "printdata_*.log")
	assert.NilError(t, err)
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	logger := log.NewStreamLogger(tmpFile, tmpFile, logrus.InfoLevel)

	t.Run("JSON Output", func(t *testing.T) {
		assert.NilError(t, tmpFile.Truncate(0))
		_, err := tmpFile.Seek(0, 0)
		assert.NilError(t, err)

		err = PrintData(logger, "json", headers, items, func(item map[string]string) []string {
			return []string{item["Name"]}
		})
		assert.NilError(t, err)

		content, err := os.ReadFile(tmpFile.Name())
		assert.NilError(t, err)

		var actual []map[string]string
		err = json.Unmarshal(content, &actual)
		assert.NilError(t, err)

		expected := []map[string]string{
			{"Project": "test-project1"},
			{"Project": "test-project2"},
		}

		assert.DeepEqual(t, actual, expected)
	})

	t.Run("Table Output", func(t *testing.T) {
		assert.NilError(t, tmpFile.Truncate(0))
		_, err := tmpFile.Seek(0, 0)
		assert.NilError(t, err)

		err = PrintData(logger, "table", headers, items, func(item map[string]string) []string {
			return []string{item["Name"]}
		})
		assert.NilError(t, err)

		lines := readLines(t, tmpFile.Name())
		assert.Assert(t, len(lines) >= 4, "expected at least 4 lines, got %d", len(lines))

		// Strip empty lines if any
		nonEmpty := filterNonEmpty(lines)

		header := strings.ToLower(strings.TrimSpace(nonEmpty[0]))
		assert.Assert(t, strings.Contains(header, "project"), "expected header to contain 'project', got: %q", nonEmpty[0])

		// Check the values exist in the expected rows
		assert.Assert(t, strings.Contains(nonEmpty[2], "test-project1"))
		assert.Assert(t, strings.Contains(nonEmpty[3], "test-project2"))
	})

}

func filterNonEmpty(lines []string) []string {
	var out []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}

// readLines reads all lines from a file and returns them as a slice
func readLines(t *testing.T, path string) []string {
	file, err := os.Open(path)
	assert.NilError(t, err)
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	assert.NilError(t, scanner.Err())
	return lines
}
