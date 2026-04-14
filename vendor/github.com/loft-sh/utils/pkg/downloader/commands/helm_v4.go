package commands

func NewHelmV4Command() Command {
	return &helmCommand{
		version:       "v4.0.4",
		versionPrefix: `:"v4.`,
	}
}
