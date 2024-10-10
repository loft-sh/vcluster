# Contributing

## How to contribute to the Email Verifier for Go

There are many ways that you can contribute to the Email Verifier project:

- Submit a bug
- Submit a code fix for a bug
- Submit additions or modifications to the documentation
- Submit a feature request

All code submissions will be reviewed and tested by the team, and those that meet a high bar for both quality and design/roadmap appropriateness will be merged into the source. Be sure to follow the existing structure when adding new files/folders.

You can find information in editor support for Go tools here: [https://github.com/golang/go/wiki/IDEsAndTextEditorPlugins](https://github.com/golang/go/wiki/IDEsAndTextEditorPlugins)

If you encounter any bugs with the library please file an issue in the [Issues](https://github.com/AfterShip/email-verifier/issues) section of the project.

## Things to keep in mind when contributing

Some guidance for when you make a contribution:

- Add/update unit tests and code as required by your change
- Make sure you run all the unit tests before you create a [Pull Request](https://help.github.com/en/github/collaborating-with-issues-and-pull-requests/about-pull-requests).
- Run end-to-end tests or simple sample code to make sure the lib works in an end-to-end scenario.

## Big contributions

If your contribution is significantly big it is better to first check with the project developers in order to make sure the change aligns with the long term plans. This can be done simply by submitting a question via the GitHub Issues section.

## Setting up your environment

Want to get started hacking on the code? Super! Follow these instructions to get up and running.

First, make sure you have the prerequisites installed and available on your `$PATH`:

- Git
- Go 1.13 or higher
- Install [pre-commit](https://pre-commit.com/)

Next, get the code:

1. Fork this repo
2. Clone your fork locally (`git clone https://github.com/<youruser>/email-verifier.git`)
3. Open a terminal and move into your local copy (`cd email-verifier`)
4. Run `pre-commit install`

### Building

Run `go build`

### Testing

Run `make test`
