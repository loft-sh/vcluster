package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Group struct {
	File    string
	Name    string
	Content string
	Imports *[]string
}

func ProcessGroups(groups map[string]*Group) {
	for groupID, group := range groups {
		groupContent := group.Content
		groupFileContent := ""

		if strings.TrimSpace(group.Content) != "" {
			if group.Name != "" {
				groupContent = "\n" + `<div className="group-name">` + group.Name + `</div>` + "\n\n" + groupContent
			}

			groupImportContent := ""
			for _, partialFile := range *group.Imports {
				groupImportContent = groupImportContent + GetPartialImport(partialFile, group.File)
			}

			if groupImportContent != "" {
				groupImportContent = groupImportContent + "\n\n"
			}

			groupFileContent = fmt.Sprintf(`%s<div className="group" data-group="%s">%s`+"\n"+`</div>`, groupImportContent, groupID, groupContent)
		}

		err := os.MkdirAll(filepath.Dir(group.File), os.ModePerm)
		if err != nil {
			panic(err)
		}

		err = os.WriteFile(group.File, []byte(groupFileContent), os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
}
