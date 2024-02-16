package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v2"

	"github.com/loft-sh/vcluster-values/values"
	"golang.org/x/exp/maps"
)

//go:embed matrix-template.tmpl
var templateString string

type issueList map[string]string

type KnownIssues struct {
	K3s map[string]issueList
	K0s map[string]issueList
	K8s map[string]issueList
	Eks map[string]issueList
}

func main() {
	if len(os.Args) != 3 {
		os.Stderr.WriteString("usage: compat-matrix generate/validate outputfile")
		os.Exit(1)
	}
	knowIssuesBytes, err := os.ReadFile("known_issues.yaml")
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
	issues := KnownIssues{}
	err = yaml.UnmarshalStrict(knowIssuesBytes, &issues)
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}

	renderedBytes := &bytes.Buffer{}
	buff := updateTableWithDistro("k3s", values.K3SVersionMap, issues)
	renderedBytes.WriteString(fmt.Sprintf(templateString, "k3s", buff.String()))
	renderedBytes.WriteString(createKnownIssue(issues.K3s))
	buff.Reset()
	buff = updateTableWithDistro("k8s", values.K8SAPIVersionMap, issues)
	renderedBytes.WriteString(fmt.Sprintf(templateString, "k8s", buff.String()))
	renderedBytes.WriteString(createKnownIssue(issues.K8s))
	buff.Reset()
	buff = updateTableWithDistro("k0s", values.K0SVersionMap, issues)
	renderedBytes.WriteString(fmt.Sprintf(templateString, "k0s", buff.String()))
	renderedBytes.WriteString(createKnownIssue(issues.K0s))
	buff.Reset()
	buff = updateTableWithDistro("eks", values.EKSAPIVersionMap, issues)
	renderedBytes.WriteString(fmt.Sprintf(templateString, "eks", buff.String()))
	renderedBytes.WriteString(createKnownIssue(issues.Eks))
	buff.Reset()

	switch os.Args[1] {
	case "generate":
		err = os.WriteFile(os.Args[2], renderedBytes.Bytes(), 0644)
		if err != nil {
			os.Stderr.WriteString(err.Error())
			os.Exit(1)
		}
	case "validate":
		currentFile, err := os.ReadFile(os.Args[2])
		if err != nil {
			os.Stderr.WriteString(err.Error())
			os.Exit(1)
		}
		if !slices.Equal(currentFile, renderedBytes.Bytes()) {
			os.Stderr.WriteString("compatibility matrix is not up to date, please update it by running `just validate-compat-matrix`")
			os.Exit(1)
		}
	}
}

func updateTableWithDistro(distroName string, versionMap map[string]string, knownIssues KnownIssues) *bytes.Buffer {
	hostVersions := maps.Keys(versionMap)
	vclusterAPIs := maps.Values(versionMap)
	slices.Sort(hostVersions)
	slices.Reverse(hostVersions)
	slices.Sort(vclusterAPIs)
	slices.Reverse(vclusterAPIs)

	buff := &bytes.Buffer{}
	table := tablewriter.NewWriter(buff)
	for i, v := range vclusterAPIs {
		vclusterAPIs[i] = removeRegistry(v)
	}
	table.SetHeader(append([]string{"distro version\nhost version"}, vclusterAPIs...))

	var issues map[string]issueList
	switch distroName {
	case "k3s":
		issues = knownIssues.K3s
	case "k0s":
		issues = knownIssues.K0s
	case "k8s":
		issues = knownIssues.K8s
	case "eks":
		issues = knownIssues.Eks
	}

	for hostVersion, issueList := range issues {
		for vclusterApi, issueDesc := range issueList {
			issues[hostVersion][removeRegistry(vclusterApi)] = issueDesc
			if removeRegistry(vclusterApi) != vclusterApi {
				// avoids removing valid entries
				delete(issues[hostVersion], vclusterApi)
			}
		}
	}

	for i, v := range hostVersions {
		table.Append(createLine(v, issues[v], vclusterAPIs, i))
	}

	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.Render()

	return buff
}

func createKnownIssue(issues map[string]issueList) string {
	if len(issues) == 0 {
		return ""
	}
	keys := maps.Keys(issues)
	slices.Sort(keys)
	buff := &bytes.Buffer{}
	table := tablewriter.NewWriter(buff)
	table.SetHeader([]string{"vCluster Distro Version", "Host K8s Version", "Known Issues"})

	for _, hostVersion := range keys {
		for vclusterVersion, issue := range issues[hostVersion] {
			table.Append([]string{vclusterVersion, hostVersion, issue})
		}
	}
	table.Render()
	if buff.Len() > 0 {
		buff.WriteString("\n")
	}
	return buff.String()
}

func createLine(version string, list issueList, vclusterAPIVersion []string, lineNumber int) []string {
	line := make([]string, 1, len(vclusterAPIVersion)+1)
	line[0] = version
	for i, v := range vclusterAPIVersion {
		char := ""
		if list[v] != "" {
			char = "!"
		} else if i == lineNumber {
			char = "recommended"
		} else {
			char = "+"
		}
		line = append(line, char)
	}
	return line
}

func removeRegistry(vclusterAPIVersion string) string {
	lastColon := strings.LastIndex(vclusterAPIVersion, ":")
	if lastColon == -1 {
		return vclusterAPIVersion
	}
	return vclusterAPIVersion[lastColon+1:]
}
