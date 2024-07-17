package generic

type MapperOptions struct {
	SkipIndex bool
}

type MapperOption func(options *MapperOptions)

func SkipIndex() MapperOption {
	return func(options *MapperOptions) {
		options.SkipIndex = true
	}
}

func getOptions(options ...MapperOption) *MapperOptions {
	newOptions := &MapperOptions{}
	for _, option := range options {
		option(newOptions)
	}
	return newOptions
}
