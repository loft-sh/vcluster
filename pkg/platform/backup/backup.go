package backup

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientpkg "sigs.k8s.io/controller-runtime/pkg/client"
)

type LogFn func(msg string)

func All(ctx context.Context, client clientpkg.Client, skip []string, infoFn LogFn) ([]runtime.Object, []error) {
	objects := []runtime.Object{}
	backupErrors := []error{}
	if !contains(skip, "clusterroletemplates") {
		infoFn("Backing up clusterroletemplates...")
		objs, err := clusterRoles(ctx, client)
		if err != nil {
			backupErrors = append(backupErrors, errors.Wrap(err, "backup clusterrole templates"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(skip, "clusteraccesses") {
		infoFn("Backing up clusteraccesses...")
		users, err := clusterAccess(ctx, client)
		if err != nil {
			backupErrors = append(backupErrors, errors.Wrap(err, "backup cluster accesses"))
		} else {
			objects = append(objects, users...)
		}
	}
	if !contains(skip, "users") {
		infoFn("Backing up users...")
		objs, err := users(ctx, client)
		if err != nil {
			backupErrors = append(backupErrors, errors.Wrap(err, "backup users"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(skip, "teams") {
		infoFn("Backing up teams...")
		objs, err := teams(ctx, client)
		if err != nil {
			backupErrors = append(backupErrors, errors.Wrap(err, "backup teams"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(skip, "sharedsecrets") {
		infoFn("Backing up shared secrets...")
		objs, err := sharedSecrets(ctx, client)
		if err != nil {
			backupErrors = append(backupErrors, errors.Wrap(err, "backup shared secrets"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(skip, "accesskeys") {
		infoFn("Backing up access keys...")
		objs, err := accessKeys(ctx, client)
		if err != nil {
			backupErrors = append(backupErrors, errors.Wrap(err, "backup access keys"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(skip, "apps") {
		infoFn("Backing up apps...")
		objs, err := apps(ctx, client)
		if err != nil {
			backupErrors = append(backupErrors, errors.Wrap(err, "backup apps"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(skip, "spacetemplates") {
		infoFn("Backing up space templates...")
		objs, err := spaceTemplates(ctx, client)
		if err != nil {
			backupErrors = append(backupErrors, errors.Wrap(err, "backup space templates"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(skip, "virtualclustertemplates") {
		infoFn("Backing up virtual cluster templates...")
		objs, err := virtualClusterTemplate(ctx, client)
		if err != nil {
			backupErrors = append(backupErrors, errors.Wrap(err, "backup virtual cluster templates"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(skip, "devpodworkspacetemplates") {
		infoFn("Backing up devpod workspace templates...")
		objs, err := devPodWorkspaceTemplate(ctx, client)
		if err != nil {
			backupErrors = append(backupErrors, errors.Wrap(err, "backup devpod workspace templates"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(skip, "clusters") {
		infoFn("Backing up clusters...")
		objs, err := clusters(ctx, client)
		if err != nil {
			backupErrors = append(backupErrors, errors.Wrap(err, "backup clusters"))
		} else {
			objects = append(objects, objs...)
		}
	}
	if !contains(skip, "runners") {
		infoFn("Backing up runners...")
		objs, err := runners(ctx, client)
		if err != nil {
			backupErrors = append(backupErrors, errors.Wrap(err, "backup runners"))
		} else {
			objects = append(objects, objs...)
		}
	}
	projects := []string{}
	if !contains(skip, "projects") {
		infoFn("Backing up projects...")
		objs, projectNames, err := allProjects(ctx, client)
		if err != nil {
			backupErrors = append(backupErrors, errors.Wrap(err, "backup projects"))
		} else {
			objects = append(objects, objs...)
		}

		projects = projectNames
	}
	if len(projects) > 0 {
		if !contains(skip, "virtualclusterinstances") {
			infoFn("Backing up virtual cluster instances...")
			objs, err := virtualClusterInstances(ctx, client, projects)
			if err != nil {
				backupErrors = append(backupErrors, errors.Wrap(err, "backup virtual cluster instances"))
			} else {
				objects = append(objects, objs...)
			}
		}
		if !contains(skip, "devpodworkspaceinstances") {
			infoFn("Backing up devpod workspace instances...")
			objs, err := devPodWorkspaceInstances(ctx, client, projects)
			if err != nil {
				backupErrors = append(backupErrors, errors.Wrap(err, "backup devpod workspace instances"))
			} else {
				objects = append(objects, objs...)
			}
		}
		if !contains(skip, "spaceinstances") {
			infoFn("Backing up space instances...")
			objs, err := spaceInstances(ctx, client, projects)
			if err != nil {
				backupErrors = append(backupErrors, errors.Wrap(err, "backup space instances"))
			} else {
				objects = append(objects, objs...)
			}
		}
		if !contains(skip, "projectsecrets") {
			infoFn("Backing up project secrets...")
			objs, err := projectSecrets(ctx, client, projects)
			if err != nil {
				backupErrors = append(backupErrors, errors.Wrap(err, "backup project secrets"))
			} else {
				objects = append(objects, objs...)
			}
		}
	}

	return objects, backupErrors
}

func allProjects(ctx context.Context, client clientpkg.Client) ([]runtime.Object, []string, error) {
	projectList := &storagev1.ProjectList{}
	err := client.List(ctx, projectList)
	if err != nil {
		return nil, nil, err
	}

	retList := []runtime.Object{}

	projectNames := []string{}
	for _, project := range projectList.Items {
		u := project

		err := resetMetadata(client.Scheme(), &u)
		if err != nil {
			return nil, nil, err
		}

		retList = append(retList, &u)
		projectNames = append(projectNames, u.Name)
	}

	return retList, projectNames, nil
}

func virtualClusterInstances(ctx context.Context, client clientpkg.Client, projects []string) ([]runtime.Object, error) {
	retList := []runtime.Object{}
	for _, projectName := range projects {
		virtualClusterInstanceList := &storagev1.VirtualClusterInstanceList{}
		err := client.List(ctx, virtualClusterInstanceList, clientpkg.InNamespace(projectutil.ProjectNamespace(projectName)))
		if err != nil {
			return nil, err
		}

		for _, o := range virtualClusterInstanceList.Items {
			u := o
			u.Status = storagev1.VirtualClusterInstanceStatus{}
			err := resetMetadata(client.Scheme(), &u)
			if err != nil {
				return nil, err
			}

			retList = append(retList, &u)
		}
	}

	return retList, nil
}

func devPodWorkspaceInstances(ctx context.Context, client clientpkg.Client, projects []string) ([]runtime.Object, error) {
	retList := []runtime.Object{}
	for _, projectName := range projects {
		devPodWorkspaceInstanceList := &storagev1.DevPodWorkspaceInstanceList{}
		err := client.List(ctx, devPodWorkspaceInstanceList, clientpkg.InNamespace(projectutil.ProjectNamespace(projectName)))
		if err != nil {
			return nil, err
		}

		for _, o := range devPodWorkspaceInstanceList.Items {
			u := o
			u.Status = storagev1.DevPodWorkspaceInstanceStatus{}
			err := resetMetadata(client.Scheme(), &u)
			if err != nil {
				return nil, err
			}

			retList = append(retList, &u)
		}
	}

	return retList, nil
}

func spaceInstances(ctx context.Context, client clientpkg.Client, projects []string) ([]runtime.Object, error) {
	retList := []runtime.Object{}
	for _, projectName := range projects {
		spaceInstanceList := &storagev1.SpaceInstanceList{}
		err := client.List(ctx, spaceInstanceList, clientpkg.InNamespace(projectutil.ProjectNamespace(projectName)))
		if err != nil {
			return nil, err
		}

		for _, o := range spaceInstanceList.Items {
			u := o
			u.Status = storagev1.SpaceInstanceStatus{}
			err := resetMetadata(client.Scheme(), &u)
			if err != nil {
				return nil, err
			}

			retList = append(retList, &u)
		}
	}

	return retList, nil
}

func projectSecrets(ctx context.Context, client clientpkg.Client, projects []string) ([]runtime.Object, error) {
	retList := []runtime.Object{}
	for _, projectName := range projects {
		secretList := &corev1.SecretList{}
		err := client.List(ctx, secretList, clientpkg.InNamespace(projectutil.ProjectNamespace(projectName)))
		if err != nil {
			return nil, err
		}

		for _, secret := range secretList.Items {
			u := secret

			if !isProjectSecret(u) {
				continue
			}

			err := resetMetadata(client.Scheme(), &u)
			if err != nil {
				return nil, err
			}

			retList = append(retList, &u)
		}
	}

	return retList, nil
}

func clusters(ctx context.Context, client clientpkg.Client) ([]runtime.Object, error) {
	clusterList := &storagev1.ClusterList{}
	err := client.List(ctx, clusterList)
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, o := range clusterList.Items {
		u := o
		u.Status = storagev1.ClusterStatus{}
		err := resetMetadata(client.Scheme(), &u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)

		// find user secrets
		if u.Spec.Config.SecretName != "" {
			secret, err := getSecret(ctx, client, u.Spec.Config.SecretNamespace, u.Spec.Config.SecretName)
			if err != nil {
				if !errors.Is(err, ErrMissingName) {
					return nil, fmt.Errorf("get cluster secret: %w", err)
				}

				continue
			}

			retList = append(retList, secret)
		}
	}

	return retList, nil
}

func runners(ctx context.Context, client clientpkg.Client) ([]runtime.Object, error) {
	runnerList := &storagev1.RunnerList{}
	err := client.List(ctx, runnerList)
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, o := range runnerList.Items {
		u := o
		u.Status = storagev1.RunnerStatus{}
		err := resetMetadata(client.Scheme(), &u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func clusterRoles(ctx context.Context, client clientpkg.Client) ([]runtime.Object, error) {
	objs := &storagev1.ClusterRoleTemplateList{}
	err := client.List(ctx, objs)
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, o := range objs.Items {
		u := o
		u.Status = storagev1.ClusterRoleTemplateStatus{}
		err := resetMetadata(client.Scheme(), &u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func clusterAccess(ctx context.Context, client clientpkg.Client) ([]runtime.Object, error) {
	objs := &storagev1.ClusterAccessList{}
	err := client.List(ctx, objs)
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, o := range objs.Items {
		u := o
		u.Status = storagev1.ClusterAccessStatus{}
		err := resetMetadata(client.Scheme(), &u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func virtualClusterTemplate(ctx context.Context, client clientpkg.Client) ([]runtime.Object, error) {
	virtualClusterTemplates := &storagev1.VirtualClusterTemplateList{}
	err := client.List(ctx, virtualClusterTemplates)
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, o := range virtualClusterTemplates.Items {
		u := o
		u.Status = storagev1.VirtualClusterTemplateStatus{}
		err := resetMetadata(client.Scheme(), &u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func spaceTemplates(ctx context.Context, client clientpkg.Client) ([]runtime.Object, error) {
	spaceTemplates := &storagev1.SpaceTemplateList{}
	err := client.List(ctx, spaceTemplates)
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, o := range spaceTemplates.Items {
		u := o
		u.Status = storagev1.SpaceTemplateStatus{}
		err := resetMetadata(client.Scheme(), &u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func devPodWorkspaceTemplate(ctx context.Context, client clientpkg.Client) ([]runtime.Object, error) {
	devPodWorkspaceTemplates := &storagev1.DevPodWorkspaceTemplateList{}
	err := client.List(ctx, devPodWorkspaceTemplates)
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, o := range devPodWorkspaceTemplates.Items {
		u := o
		u.Status = storagev1.DevPodWorkspaceTemplateStatus{}
		err := resetMetadata(client.Scheme(), &u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func apps(ctx context.Context, client clientpkg.Client) ([]runtime.Object, error) {
	apps := &storagev1.AppList{}
	err := client.List(ctx, apps)
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, o := range apps.Items {
		u := o
		u.Status = storagev1.AppStatus{}
		err := resetMetadata(client.Scheme(), &u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func accessKeys(ctx context.Context, client clientpkg.Client) ([]runtime.Object, error) {
	accessKeyList := &storagev1.AccessKeyList{}
	err := client.List(ctx, accessKeyList)
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, o := range accessKeyList.Items {
		u := o
		u.Status = storagev1.AccessKeyStatus{}
		err := resetMetadata(client.Scheme(), &u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func sharedSecrets(ctx context.Context, client clientpkg.Client) ([]runtime.Object, error) {
	sharedSecretList := &storagev1.SharedSecretList{}
	err := client.List(ctx, sharedSecretList, clientpkg.InNamespace(""))
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, sharedSecret := range sharedSecretList.Items {
		u := sharedSecret
		u.Status = storagev1.SharedSecretStatus{}
		err := resetMetadata(client.Scheme(), &u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func teams(ctx context.Context, client clientpkg.Client) ([]runtime.Object, error) {
	teamList := &storagev1.TeamList{}
	err := client.List(ctx, teamList)
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, team := range teamList.Items {
		u := team
		u.Status = storagev1.TeamStatus{}
		err := resetMetadata(client.Scheme(), &u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)
	}

	return retList, nil
}

func users(ctx context.Context, client clientpkg.Client) ([]runtime.Object, error) {
	userList := &storagev1.UserList{}
	err := client.List(ctx, userList)
	if err != nil {
		return nil, err
	}

	retList := []runtime.Object{}
	for _, user := range userList.Items {
		u := user
		u.Status = storagev1.UserStatus{}
		err := resetMetadata(client.Scheme(), &u)
		if err != nil {
			return nil, err
		}

		retList = append(retList, &u)

		// find user secrets
		if u.Spec.PasswordRef != nil {
			secret, err := getSecret(ctx, client, u.Spec.PasswordRef.SecretNamespace, u.Spec.PasswordRef.SecretName)
			if err != nil {
				if !errors.Is(err, ErrMissingName) {
					return nil, fmt.Errorf("get user secret: %w", err)
				}
			} else if secret != nil {
				retList = append(retList, secret)
			}
		}
		if u.Spec.CodesRef != nil {
			secret, err := getSecret(ctx, client, u.Spec.CodesRef.SecretNamespace, u.Spec.CodesRef.SecretName)
			if err != nil {
				if !errors.Is(err, ErrMissingName) {
					return nil, fmt.Errorf("get user secret: %w", err)
				}
			} else if secret != nil {
				retList = append(retList, secret)
			}
		}
	}

	return retList, nil
}

func isProjectSecret(secret corev1.Secret) bool {
	for k, v := range secret.Labels {
		if k == "loft.sh/project-secret" && v == "true" {
			return true
		}
	}

	return false
}

var ErrMissingName = errors.New("backup: missing namespace or name")

func getSecret(ctx context.Context, client clientpkg.Client, namespace, name string) (*corev1.Secret, error) {
	if namespace == "" || name == "" {
		return nil, ErrMissingName
	}

	secret := &corev1.Secret{}
	err := client.Get(ctx, clientpkg.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, secret)
	if err != nil && !kerrors.IsNotFound(err) {
		return nil, err
	}

	err = resetMetadata(client.Scheme(), secret)
	if err != nil {
		return nil, fmt.Errorf("reset metadata secret: %w", err)
	}

	return secret, nil
}

func resetMetadata(scheme *runtime.Scheme, obj runtime.Object) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	accessor.SetGenerateName("")
	accessor.SetSelfLink("")
	accessor.SetCreationTimestamp(metav1.Time{})
	accessor.SetFinalizers(nil)
	accessor.SetGeneration(0)
	accessor.SetManagedFields(nil)
	accessor.SetOwnerReferences(nil)
	accessor.SetResourceVersion("")
	accessor.SetUID("")
	accessor.SetDeletionTimestamp(nil)

	gvk, err := GVKFrom(scheme, obj)
	if err != nil {
		return err
	}

	typeAccessor, err := meta.TypeAccessor(obj)
	if err != nil {
		return err
	}

	typeAccessor.SetKind(gvk.Kind)
	typeAccessor.SetAPIVersion(gvk.GroupVersion().String())
	return nil
}

func GVKFrom(scheme *runtime.Scheme, obj runtime.Object) (schema.GroupVersionKind, error) {
	gvks, _, err := scheme.ObjectKinds(obj)
	if err != nil {
		return schema.GroupVersionKind{}, err
	} else if len(gvks) != 1 {
		return schema.GroupVersionKind{}, errors.Errorf("unexpected number of object kinds: %d", len(gvks))
	}

	return gvks[0], nil
}

func contains(arr []string, s string) bool {
	for _, t := range arr {
		if t == s {
			return true
		}
	}
	return false
}

func ToYAML(objects []runtime.Object) ([]byte, error) {
	retString := []string{}
	for _, o := range objects {
		out, err := yaml.Marshal(o)
		if err != nil {
			return nil, errors.Wrap(err, "marshal object")
		}

		retString = append(retString, string(out))
	}
	s := strings.Join(retString, "\n---\n")

	return []byte(s), nil
}
