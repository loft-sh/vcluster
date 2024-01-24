package devpod

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/gorilla/websocket"
	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v3/pkg/apis/storage/v1"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/devpod/list"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/client/naming"
	"github.com/loft-sh/loftctl/v3/pkg/kube"
	"github.com/loft-sh/loftctl/v3/pkg/parameters"
	"github.com/loft-sh/loftctl/v3/pkg/remotecommand"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// UpCmd holds the cmd flags:
type UpCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

// NewUpCmd creates a new command
func NewUpCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &UpCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:   "up",
		Short: "Runs up on a workspace",
		Long: `
#######################################################
#################### loft devpod up ###################
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), os.Stdin, os.Stdout, os.Stderr)
		},
	}

	return c
}

func (cmd *UpCmd) Run(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	workspace, err := findWorkspace(ctx, baseClient)
	if err != nil {
		return err
	}

	// create workspace if doesn't exist
	if workspace == nil {
		workspace, err = createWorkspace(ctx, baseClient, cmd.Log.ErrorStreamOnly())
		if err != nil {
			return fmt.Errorf("create workspace: %w", err)
		}
	}

	conn, err := dialWorkspace(baseClient, workspace, "up", optionsFromEnv(storagev1.DevPodFlagsUp))
	if err != nil {
		return err
	}

	_, err = remotecommand.ExecuteConn(ctx, conn, stdin, stdout, stderr, cmd.Log.ErrorStreamOnly())
	if err != nil {
		return fmt.Errorf("error executing: %w", err)
	}

	return nil
}

func createWorkspace(ctx context.Context, baseClient client.Client, log log.Logger) (*managementv1.DevPodWorkspaceInstance, error) {
	workspaceID, workspaceUID, projectName, err := getWorkspaceInfo()
	if err != nil {
		return nil, err
	}

	// get template
	template := os.Getenv(LOFT_TEMPLATE_OPTION)
	if template == "" {
		return nil, fmt.Errorf("%s is missing in environment", LOFT_TEMPLATE_OPTION)
	}

	// create client
	managementClient, err := baseClient.Management()
	if err != nil {
		return nil, err
	}

	// get template version
	templateVersion := os.Getenv(LOFT_TEMPLATE_VERSION_OPTION)
	if templateVersion == "latest" {
		templateVersion = ""
	}

	// find parameters
	resolvedParameters, err := getParametersFromEnvironment(ctx, managementClient, projectName, template, templateVersion)
	if err != nil {
		return nil, fmt.Errorf("resolve parameters: %w", err)
	}

	// get workspace picture
	workspacePicture := os.Getenv("WORKSPACE_PICTURE")
	// get workspace source
	workspaceSource := os.Getenv("WORKSPACE_SOURCE")

	workspace := &managementv1.DevPodWorkspaceInstance{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: naming.SafeConcatNameMax([]string{workspaceID}, 53) + "-",
			Namespace:    naming.ProjectNamespace(projectName),
			Labels: map[string]string{
				storagev1.DevPodWorkspaceIDLabel:  workspaceID,
				storagev1.DevPodWorkspaceUIDLabel: workspaceUID,
			},
			Annotations: map[string]string{
				storagev1.DevPodWorkspacePictureAnnotation: workspacePicture,
				storagev1.DevPodWorkspaceSourceAnnotation:  workspaceSource,
			},
		},
		Spec: managementv1.DevPodWorkspaceInstanceSpec{
			DevPodWorkspaceInstanceSpec: storagev1.DevPodWorkspaceInstanceSpec{
				DisplayName: workspaceID,
				Parameters:  resolvedParameters,
				TemplateRef: &storagev1.TemplateRef{
					Name:    template,
					Version: templateVersion,
				},
			},
		},
	}

	// check if runner is defined
	runnerName := os.Getenv("LOFT_RUNNER")
	if runnerName != "" {
		workspace.Spec.RunnerRef.Runner = runnerName
	}

	// create instance
	workspace, err = managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(naming.ProjectNamespace(projectName)).Create(ctx, workspace, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	log.Infof("Created workspace %s", workspace.Name)

	// we need to wait until instance is scheduled
	err = wait.PollUntilContextTimeout(ctx, time.Second, 30*time.Second, true, func(ctx context.Context) (done bool, err error) {
		workspace, err = managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(naming.ProjectNamespace(projectName)).Get(ctx, workspace.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if !isReady(workspace) {
			log.Debugf("Workspace %s is in phase %s, waiting until its ready", workspace.Name, workspace.Status.Phase)
			return false, nil
		}

		log.Debugf("Workspace %s is ready", workspace.Name)
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("wait for instance to get ready: %w", err)
	}

	return workspace, nil
}

func getParametersFromEnvironment(ctx context.Context, kubeClient kube.Interface, projectName, templateName, templateVersion string) (string, error) {
	// are there any parameters in environment?
	environmentVariables := os.Environ()
	envMap := map[string]string{}
	for _, v := range environmentVariables {
		splitted := strings.SplitN(v, "=", 2)
		if len(splitted) != 2 {
			continue
		} else if !strings.HasPrefix(splitted[0], "TEMPLATE_OPTION_") {
			continue
		}

		envMap[splitted[0]] = splitted[1]
	}
	if len(envMap) == 0 {
		return "", nil
	}

	// find these in the template
	template, err := list.FindTemplate(ctx, kubeClient, projectName, templateName)
	if err != nil {
		return "", fmt.Errorf("find template: %w", err)
	}

	// find version
	var templateParameters []storagev1.AppParameter
	if len(template.Spec.Versions) > 0 {
		templateParameters, err = list.GetTemplateParameters(template, templateVersion)
		if err != nil {
			return "", err
		}
	} else {
		templateParameters = template.Spec.Parameters
	}

	// parse versions
	outMap := map[string]interface{}{}
	for _, parameter := range templateParameters {
		// check if its in environment
		val := envMap[list.VariableToEnvironmentVariable(parameter.Variable)]
		outVal, err := parameters.VerifyValue(val, parameter)
		if err != nil {
			return "", fmt.Errorf("validate parameter %s: %w", parameter.Variable, err)
		}

		outMap[parameter.Variable] = outVal
	}

	// convert to string
	out, err := yaml.Marshal(outMap)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func isReady(workspace *managementv1.DevPodWorkspaceInstance) bool {
	return workspace.Status.Phase == storagev1.InstanceReady
}

func getWorkspaceInfo() (string, string, string, error) {
	// get workspace id
	workspaceID := os.Getenv(LOFT_WORKSPACE_ID)
	if workspaceID == "" {
		return "", "", "", fmt.Errorf("%s is missing in environment", LOFT_WORKSPACE_ID)
	}

	// get workspace uid
	workspaceUID := os.Getenv(LOFT_WORKSPACE_UID)
	if workspaceUID == "" {
		return "", "", "", fmt.Errorf("%s is missing in environment", LOFT_WORKSPACE_UID)
	}

	// get project
	projectName := os.Getenv(LOFT_PROJECT_OPTION)
	if projectName == "" {
		return "", "", "", fmt.Errorf("%s is missing in environment", LOFT_PROJECT_OPTION)
	}

	return workspaceID, workspaceUID, projectName, nil
}

func findWorkspace(ctx context.Context, baseClient client.Client) (*managementv1.DevPodWorkspaceInstance, error) {
	_, workspaceUID, projectName, err := getWorkspaceInfo()
	if err != nil {
		return nil, err
	}

	// create client
	managementClient, err := baseClient.Management()
	if err != nil {
		return nil, fmt.Errorf("create management client: %w", err)
	}

	// get workspace
	workspaceList, err := managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(naming.ProjectNamespace(projectName)).List(ctx, metav1.ListOptions{
		LabelSelector: storagev1.DevPodWorkspaceUIDLabel + "=" + workspaceUID,
	})
	if err != nil {
		return nil, err
	} else if len(workspaceList.Items) == 0 {
		return nil, nil
	}

	return &workspaceList.Items[0], nil
}

func optionsFromEnv(name string) url.Values {
	options := os.Getenv(name)
	if options != "" {
		return url.Values{
			"options": []string{options},
		}
	}

	return nil
}

func dialWorkspace(baseClient client.Client, workspace *managementv1.DevPodWorkspaceInstance, subResource string, values url.Values) (*websocket.Conn, error) {
	restConfig, err := baseClient.ManagementConfig()
	if err != nil {
		return nil, err
	}

	host := restConfig.Host
	if workspace.Annotations != nil && workspace.Annotations[storagev1.DevPodWorkspaceRunnerEndpointAnnotation] != "" {
		host = workspace.Annotations[storagev1.DevPodWorkspaceRunnerEndpointAnnotation]
	}

	parsedURL, _ := url.Parse(host)
	if parsedURL != nil && parsedURL.Host != "" {
		host = parsedURL.Host
	}

	loftURL := "wss://" + host + "/kubernetes/management/apis/management.loft.sh/v1/namespaces/" + workspace.Namespace + "/devpodworkspaceinstances/" + workspace.Name + "/" + subResource
	if len(values) > 0 {
		loftURL += "?" + values.Encode()
	}

	dialer := websocket.Dialer{
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: true},
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
	}

	conn, response, err := dialer.Dial(loftURL, map[string][]string{
		"Authorization": {"Bearer " + restConfig.BearerToken},
	})
	if err != nil {
		if response != nil {
			out, _ := io.ReadAll(response.Body)
			headers, _ := json.Marshal(response.Header)
			return nil, fmt.Errorf("error dialing websocket %s (code %d): headers - %s, response - %s, error - %w", loftURL, response.StatusCode, string(headers), string(out), err)
		}

		return nil, fmt.Errorf("error dialing websocket %s: %w", loftURL, err)
	}

	return conn, nil
}
