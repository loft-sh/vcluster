package patches

import (
	"testing"
)

func TestIsRootChild(t *testing.T) {
	testCases := map[OpPath]bool{
		"":      false,
		".baz":  true,
		"$baz":  true,
		"$":     false,
		"*.baz": false,
	}

	for input, expected := range testCases {
		actual := input.isRootChild()
		if expected != actual {
			t.Errorf("TestCase %s\nactual:%t\nexpected:%t", input, actual, expected)
		}
	}
}

func TestGetChildName(t *testing.T) {
	testCases := map[OpPath]string{
		``:                   ``,
		`.baz`:               `baz`,
		`$baz`:               `baz`,
		`$`:                  ``,
		`*.baz`:              `baz`,
		`*..baz`:             `baz`,
		"deployments['baz']": `baz`,
		`deployments["baz"]`: `baz`,
	}

	for input, expected := range testCases {
		actual := input.getChildName()
		if expected != actual {
			t.Errorf("TestCase %s\nactual:%s\nexpected:%s", input, actual, expected)
		}
	}
}

func TestGetParentPath(t *testing.T) {
	testCases := map[OpPath]string{
		"":                                "",
		"parent1.child1":                  "parent1",
		"parent1['child1']":               "parent1",
		"parent1.parent2.child1":          "parent1.parent2",
		"parent1['parent2']['child1']":    "parent1['parent2']",
		"$.parent1.child1":                "$.parent1",
		"$.*.child1":                      "$.*",
		"$.deployments[*].parent1.child1": "$.deployments[*].parent1",
		"$.deployments[?(@.name=='backend')].parent1.child1":  "$.deployments[?(@.name=='backend')].parent1",
		"$.deployments[?(@.name=~/^backend/)].parent1.child1": "$.deployments[?(@.name=~/^backend/)].parent1",
		"$.deployments[?(@.name=~/^backend/)].child1":         "$.deployments[?(@.name=~/^backend/)]",
		"$.deployments[?(@.name=~/^Backend/)].child1":         "$.deployments[?(@.name=~/^Backend/)]",
		"$.deployments[?(@.name=~/^backend/)]":                "$.deployments",
		"$.deployments[?(@.name=~/^backend\\//)]":             "$.deployments",
		"$.deployments[*]": "$.deployments",
	}

	for input, expected := range testCases {
		actual := input.getParentPath()
		if expected != actual {
			t.Errorf("TestCase %s\nactual:%s\nexpected:%s", input, actual, expected)
		}
	}
}
