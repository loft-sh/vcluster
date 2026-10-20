package token

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	bootstrapapi "k8s.io/cluster-bootstrap/token/api"
)

type ListCmd struct {
	*flags.GlobalFlags

	Output string

	Log log.Logger
}

func NewListCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ListCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := `########################################################
################# vcluster token list #################
########################################################
List all node bootstrap tokens for a vCluster with private nodes enabled.
#######################################################
	`

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all node bootstrap tokens for a vCluster with private nodes enabled.",
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	listCmd.Flags().StringVar(&cmd.Output, "output", "table", "Choose the format of the output. [table|json]")
	return listCmd
}

type Token struct {
	ID      string
	Created time.Time
	Expires string
}

func (cmd *ListCmd) Run(ctx context.Context) error {
	// get the client
	vClient, err := getClient(cmd.GlobalFlags)
	if err != nil {
		return err
	}

	// list the tokens
	secrets, err := vClient.CoreV1().Secrets(metav1.NamespaceSystem).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=true", constants.TokenLabelKey),
	})
	if err != nil {
		return err
	}

	// gather the tokens
	tokens := make([]Token, 0)
	for _, secret := range secrets.Items {
		expires, ok := secret.Data[bootstrapapi.BootstrapTokenExpirationKey]
		if !ok {
			expires = []byte("never")
		}

		tokens = append(tokens, Token{
			ID:      string(secret.Data[bootstrapapi.BootstrapTokenIDKey]),
			Created: secret.CreationTimestamp.Time,
			Expires: string(expires),
		})
	}

	// print the tokens
	if cmd.Output == "json" {
		bytes, err := json.MarshalIndent(tokens, "", "    ")
		if err != nil {
			return fmt.Errorf("json marshal vClusters: %w", err)
		}

		cmd.Log.WriteString(logrus.InfoLevel, string(bytes)+"\n")
	} else {
		header := []string{"ID", "EXPIRES", "CREATED"}
		var values [][]string
		for _, token := range tokens {
			values = append(values, []string{token.ID, token.Expires, token.Created.Format(time.RFC3339)})
		}
		table.PrintTable(cmd.Log, header, values)
	}

	return nil
}
