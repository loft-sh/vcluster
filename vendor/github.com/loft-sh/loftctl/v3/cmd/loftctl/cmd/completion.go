package cmd

import (
	"bytes"
	"os"
	"text/template"

	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// CompletionCmd holds the cmd flags
type CompletionCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

func NewCompletionCmd(command *cobra.Command, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &CompletionCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("completion", `

To load completions:
Bash:
$ source <({{.Use}} completion bash)
# To load completions for each session, execute once:
Linux:
  $ {{.Use}} completion bash > /etc/bash_completion.d/{{.Use}}
MacOS:
  $ {{.Use}} completion bash > /usr/local/etc/bash_completion.d/{{.Use}}
Zsh:
# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:
$ echo "autoload -U compinit; compinit" >> ~/.zshrc
# To load completions for each session, execute once:
$ {{.Use}} completion zsh > "${fpath[1]}/_{{.Use}}"
# You will need to start a new shell for this setup to take effect.
Fish:
$ {{.Use}} completion fish | source
# To load completions for each session, execute once:
$ {{.Use}} completion fish > ~/.config/fish/completions/{{.Use}}.fish
	`)
	tmpl, err := template.New("completion").Parse(description)
	if err != nil {
		panic(err)
	}
	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, command)
	if err != nil {
		panic(err)
	}
	completionDescription := tpl.String()

	// completionCmd represents the completion command
	var completionCmd = &cobra.Command{
		Use:                   "completion [bash|zsh|fish|powershell]",
		Short:                 "Generate shell completion scripts",
		Long:                  completionDescription,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args)
		},
	}
	return completionCmd
}

func (cmd *CompletionCmd) Run(cobraCmd *cobra.Command, args []string) error {
	switch args[0] {
	case "bash":
		return cobraCmd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		return cobraCmd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		return cobraCmd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		return cobraCmd.Root().GenPowerShellCompletion(os.Stdout)
	}
	return nil
}
