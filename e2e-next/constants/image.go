package constants

import (
	"strings"
)

const (
	DefaultVclusterImage = "ghcr.io/loft-sh/vcluster:0.30.0"
)

var (
	repository string
	tag        string
)

func GetImage() string {
	if repository == "" && tag == "" {
		return DefaultVclusterImage
	}
	if tag == "" {
		return repository
	}
	return repository + ":" + tag
}

func GetRepository() string {
	if repository == "" {
		return "ghcr.io/loft-sh/vcluster"
	}
	return repository
}

func GetTag() string {
	if tag == "" {
		return "dev-next"
	}
	return tag
}

func SetImage(image string) {
	if strings.Contains(image, "@") {
		// Handle digest format: repo@sha256:xxx
		parts := strings.SplitN(image, "@", 2)
		repository = parts[0]
		tag = "@" + parts[1]
	} else {
		// Handle tag format: repo:tag
		parts := strings.SplitN(image, ":", 2)
		repository = parts[0]
		if len(parts) == 2 {
			tag = parts[1]
		} else {
			tag = ""
		}
	}
}
