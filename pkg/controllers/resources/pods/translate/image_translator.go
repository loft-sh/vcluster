package translate

type ImageTranslator interface {
	Translate(image string) string
}

type imageTranslator struct {
	translateImages map[string]string
}

func NewImageTranslator(translateImages map[string]string) (ImageTranslator, error) {
	return &imageTranslator{
		translateImages: translateImages,
	}, nil
}

func (i *imageTranslator) Translate(image string) string {
	out, ok := i.translateImages[image]
	if ok {
		return out
	}

	return image
}
