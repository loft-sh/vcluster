package commands

func NewHelmV3Command() Command {
	return &helmCommand{
		version:       "v3.12.3",
		versionPrefix: `:"v3.`,
	}
}
