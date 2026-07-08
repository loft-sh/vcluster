package connect

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/sshconfig"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/loft-sh/vcluster/pkg/platform/slurm"
	"github.com/loft-sh/vcluster/pkg/platform/tailnet"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

// SlurmCmd holds the connect slurm command flags.
type SlurmCmd struct {
	*flags.GlobalFlags

	Project  string
	Alias    string
	Wait     bool
	Remove   bool
	Stdio    bool
	Host     string
	Debug    bool
	Insecure bool

	log log.Logger
}

// newSlurmCmd creates the `connect slurm` command.
func newSlurmCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &SlurmCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("connect slurm", `
Configures SSH access to a Slurm instance's login node over the platform tailnet.

After running this once you can reach the login node with a plain "ssh <alias>".
The command writes an SSH config drop-in (~/.ssh/vcluster/config) whose
ProxyCommand starts an ephemeral tailnet node and tunnels to the login node.

Example:
vcluster platform connect slurm my-slurm
vcluster platform connect slurm my-slurm --project my-project --alias my-slurm
vcluster platform connect slurm my-slurm --remove
########################################################
	`)
	useLine, validator := util.NamedPositionalArgsValidator(true, true, "SLURM_INSTANCE_NAME")
	c := &cobra.Command{
		Use:   "slurm" + useLine,
		Short: "Configures SSH access to a Slurm instance's login node",
		Long:  description,
		Args:  validator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// In stdio (ProxyCommand) mode stdout carries the raw SSH byte
			// stream, so all logging must go to stderr and no version warning
			// may be printed.
			if cmd.Stdio {
				cmd.log = log.GetInstance().ErrorStreamOnly()
				return cmd.runStdio(cobraCmd.Context(), args[0])
			}

			upgrade.PrintNewerVersionWarning()

			if cmd.Remove {
				return cmd.runRemove(cobraCmd.Context(), args[0])
			}
			return cmd.runConnect(cobraCmd.Context(), args[0])
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project the Slurm instance belongs to")
	c.Flags().StringVar(&cmd.Alias, "alias", "", "The SSH host alias to use (default: <instance>.<project>.slurm)")
	c.Flags().BoolVar(&cmd.Wait, "wait", true, "Wait until the Slurm instance is ready for SSH access before writing the config")
	c.Flags().BoolVar(&cmd.Remove, "remove", false, "Remove the SSH config entry for this Slurm instance")
	c.Flags().BoolVar(&cmd.Stdio, "stdio", false, "Internal: run as an SSH ProxyCommand, tunneling stdin/stdout to the login node")
	_ = c.Flags().MarkHidden("stdio")
	c.Flags().StringVar(&cmd.Host, "host", "", "Internal: platform host to pin for the tailnet connection in --stdio mode")
	_ = c.Flags().MarkHidden("host")
	c.Flags().BoolVar(&cmd.Debug, "debug", false, "Emit tailnet connection logs to stderr (also enabled by VCLUSTER_SLURM_DEBUG=true)")
	c.Flags().BoolVar(&cmd.Insecure, "insecure", false, "Tolerate the platform's self-signed certificate when connecting to the tailnet coordinator")

	return c
}

// runConnect writes (or replaces) the SSH config drop-in for the instance.
func (cmd *SlurmCmd) runConnect(ctx context.Context, instance string) error {
	cfg := cmd.LoadedConfig(cmd.log)
	platformClient, err := platform.InitClientFromConfig(ctx, cfg)
	if err != nil {
		return err
	}

	_, project, err := platform.SelectProjectOrCluster(ctx, platformClient, "", cmd.Project, false, cmd.log)
	if err != nil {
		return err
	}
	cmd.Project = project

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	// Early existence/RBAC check and wait for SSH readiness.
	if _, err := slurm.WaitForTailnetReady(ctx, managementClient, project, instance, cmd.Wait, cmd.log); err != nil {
		return err
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve vcluster executable path: %w", err)
	}

	manager, err := sshconfig.New()
	if err != nil {
		return err
	}

	block := sshconfig.Block{
		PlatformHost: cfg.Platform.Host,
		Project:      project,
		Instance:     instance,
		Alias:        cmd.Alias,
		Executable:   executable,
		Insecure:     cmd.Insecure || cfg.Platform.Insecure,
	}
	if err := manager.Add(block); err != nil {
		return err
	}

	alias := cmd.Alias
	if alias == "" {
		alias = sshconfig.DefaultAlias(instance, project)
	}
	cmd.log.Donef("Configured SSH access to Slurm instance %s. Connect with:", ansi.Color(instance, "white+b"))
	cmd.log.Infof("  ssh %s", ansi.Color(alias, "white+b"))
	return nil
}

// runRemove removes the SSH config drop-in for the instance.
func (cmd *SlurmCmd) runRemove(ctx context.Context, instance string) error {
	cfg := cmd.LoadedConfig(cmd.log)
	platformClient, err := platform.InitClientFromConfig(ctx, cfg)
	if err != nil {
		return err
	}

	_, project, err := platform.SelectProjectOrCluster(ctx, platformClient, "", cmd.Project, false, cmd.log)
	if err != nil {
		return err
	}

	manager, err := sshconfig.New()
	if err != nil {
		return err
	}

	removed, err := manager.Remove(cfg.Platform.Host, project, instance)
	if err != nil {
		return err
	}
	if !removed {
		cmd.log.Infof("No SSH config entry found for Slurm instance %s in project %s", instance, project)
		return nil
	}

	cmd.log.Donef("Removed SSH config entry for Slurm instance %s", ansi.Color(instance, "white+b"))
	return nil
}

// runStdio runs the hidden ProxyCommand mode: it starts an ephemeral tailnet
// node and tunnels stdin/stdout to the login node's ssh port.
func (cmd *SlurmCmd) runStdio(ctx context.Context, instance string) error {
	if cmd.Project == "" {
		return fmt.Errorf("--project is required in --stdio mode")
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := cmd.LoadedConfig(cmd.log)

	host := cmd.Host
	if host == "" {
		host = cfg.Platform.Host
	}
	// The CLI stores a single platform login. If the pinned host no longer
	// matches it, the stored access key belongs to a different platform, so
	// fail before touching that platform with the wrong credentials.
	if sshconfig.NormalizeHost(cfg.Platform.Host) != sshconfig.NormalizeHost(host) {
		return fmt.Errorf("this SSH connection targets platform %q, but the CLI is currently logged into %q; run `vcluster platform login %s` and then `vcluster platform connect slurm %s --project %s` again to refresh the SSH config", host, cfg.Platform.Host, host, instance, cmd.Project)
	}

	platformClient, err := platform.InitClientFromConfig(ctx, cfg)
	if err != nil {
		return err
	}

	hostname, err := tailnet.BuildClientHostname(selfUserName(platformClient))
	if err != nil {
		return err
	}

	debug := cmd.Debug || os.Getenv("VCLUSTER_SLURM_DEBUG") == "true"

	return tailnet.RunStdio(ctx, tailnet.Options{
		Host:          host,
		AccessKey:     cfg.Platform.AccessKey,
		Hostname:      hostname,
		LoginHostname: fmt.Sprintf("ssh.%s.%s.slurm", instance, cmd.Project),
		Insecure:      cmd.Insecure || cfg.Platform.Insecure,
		Debug:         debug,
	})
}

// selfUserName returns the logged-in platform user's name for use as the
// tailnet hostname user segment, falling back to the team name.
func selfUserName(platformClient platform.Client) string {
	self := platformClient.Self()
	if self == nil {
		return ""
	}
	if self.Status.User != nil && self.Status.User.Name != "" {
		return self.Status.User.Name
	}
	if self.Status.Team != nil && self.Status.Team.Name != "" {
		return self.Status.Team.Name
	}
	return ""
}
