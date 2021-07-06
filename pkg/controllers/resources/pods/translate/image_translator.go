package translate

import (
	"fmt"
	"strings"
)

type ImageTranslator interface {
	Translate(image string) string
}

type imageTranslator struct {
	translateImages map[string]string
}

func NewImageTranslator(translateImages []string) (ImageTranslator, error) {
	translateImagesMap := map[string]string{}
	for _, t := range translateImages {
		i := strings.Split(strings.TrimSpace(t), "=")
		if len(i) != 2 {
			return nil, fmt.Errorf("error parsing translate image '%s': bad format expected image1=image2", t)
		}

		translateImagesMap[i[0]] = i[1]
	}

	return &imageTranslator{
		translateImages: translateImagesMap,
	}, nil
}

func (i *imageTranslator) Translate(image string) string {
	out, ok := i.translateImages[image]
	if ok {
		return out
	}

	return image
}
