package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"slices"
	"strings"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v2"

	"golang.org/x/exp/maps"
)

const header = `---
title: Compatibility Matrix
sidebar_label: Compatibility Matrix
---

`

//go:embed matrix-template.tmpl
var templateString string

type issueList map[string]string

type KnownIssues struct {
	K8s map[string]issueList
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
	renderedBytes.WriteString(header)
	versionMap := vclusterconfig.K8SVersionMap
	buff := updateTableWithDistro("k8s", versionMap, issues)
	renderedBytes.WriteString(fmt.Sprintf(templateString, "k8s", "k8s", buff.String()))
	renderedBytes.WriteString(createKnownIssue(issues.K8s))
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
			os.Stderr.WriteString("compatibility matrix is not up to date, please update it by running `just generate-compatibility`")
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
	table.SetHeader(append([]string{""}, vclusterAPIs...))

	var issues map[string]issueList
	switch distroName {
	case "k8s":
		issues = knownIssues.K8s
	}

	for hostVersion, issueList := range issues {
		for vclusterAPI, issueDesc := range issueList {
			if issues[hostVersion] == nil {
				issues[hostVersion] = make(map[string]string, 0)
			}
			issues[hostVersion][removeRegistry(vclusterAPI)] = issueDesc
			if removeRegistry(vclusterAPI) != vclusterAPI {
				// avoids removing valid entries
				delete(issues[hostVersion], vclusterAPI)
			}
		}
	}

	for i, v := range hostVersions {
		table.Append(createLine(v, issues[v], vclusterAPIs, i))
	}

	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoFormatHeaders(false)
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
			char = ":warning"
		} else if i == lineNumber {
			char = ":white_check_mark:"
		} else {
			char = ":ok:"
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
