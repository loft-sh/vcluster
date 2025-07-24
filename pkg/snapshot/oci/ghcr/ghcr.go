package ghcr

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-github/v53/github"
	"golang.org/x/oauth2"
)

const containerPackageType = "container"

func IsGHCRContainerRegistry(ref name.Reference) bool {
	return strings.HasSuffix(ref.Context().RegistryStr(), "ghcr.io")
}

type ImageDeleter interface {
	Delete(ctx context.Context, ref name.Reference) error
}

func NewImageDeleter(token string) ImageDeleter {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.TODO(), ts)
	return &imageDeleter{
		client: github.NewClient(tc),
	}
}

type packageClient interface {
	PackageDeleteVersion(ctx context.Context, org, packageType, packageName string, packageVersionID int64) (*github.Response, error)
	PackageGetAllVersions(ctx context.Context, user, packageType, packageName string, opts *github.PackageListOptions) ([]*github.PackageVersion, *github.Response, error)
}

type imageDeleter struct {
	client *github.Client
}

func (i *imageDeleter) Delete(ctx context.Context, ref name.Reference) error {
	orgOrUser, pkg, tag, err := parsePackageInfo(ref)
	if err != nil {
		return err
	}

	pkgs, versions, err := i.getPackageVersions(ctx, orgOrUser, pkg)
	if err != nil {
		return err
	}

	version := findPackageVersionForTag(versions, tag)
	if version != nil {
		_, err = pkgs.PackageDeleteVersion(ctx, orgOrUser, containerPackageType, pkg, version.GetID())
		return err
	}
	return fmt.Errorf("image not found: %s", ref.String())
}

func (i *imageDeleter) getPackageVersions(ctx context.Context, orgOrUser, pkg string) (packageClient, []*github.PackageVersion, error) {
	var all []*github.PackageVersion

	options := &github.PackageListOptions{
		PackageType: ptr.String("container"),
		State:       ptr.String("active"),
		ListOptions: github.ListOptions{PerPage: 50},
	}
	for {
		versions, resp, err := i.client.Organizations.PackageGetAllVersions(ctx, orgOrUser, containerPackageType, pkg, options)
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			break
		} else if err != nil {
			return nil, nil, err
		} else if resp == nil {
			return nil, nil, fmt.Errorf("no response listing package %s versions", pkg)
		}

		all = append(all, versions...)
		if resp.NextPage == 0 {
			break
		}
		options.Page = resp.NextPage
	}
	if len(all) > 0 {
		// Organization packages
		return i.client.Organizations, all, nil
	}

	options = &github.PackageListOptions{
		PackageType: ptr.String("container"),
		State:       ptr.String("active"),
		ListOptions: github.ListOptions{PerPage: 50},
	}
	for {
		versions, resp, err := i.client.Users.PackageGetAllVersions(ctx, orgOrUser, containerPackageType, pkg, options)
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			break
		} else if err != nil {
			return nil, nil, err
		} else if resp == nil {
			return nil, nil, fmt.Errorf("no response listing package %s versions", pkg)
		}

		all = append(all, versions...)
		if resp.NextPage == 0 {
			break
		}
		options.Page = resp.NextPage
	}
	if len(all) > 0 {
		// User packages
		return i.client.Users, all, nil
	}

	// none found
	return nil, nil, nil
}

func findPackageVersionForTag(versions []*github.PackageVersion, tag string) *github.PackageVersion {
	for _, version := range versions {
		if version.GetMetadata() != nil && version.GetMetadata().GetContainer() != nil && slices.Contains(version.GetMetadata().GetContainer().Tags, tag) {
			return version
		}
	}
	return nil
}

func parsePackageInfo(ref name.Reference) (string, string, string, error) {
	parts := strings.SplitN(ref.Name(), "/", 3)
	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("could not determine user or organization")
	}

	pkgParts := strings.SplitN(parts[2], ":", 2)
	if len(pkgParts) < 2 {
		return "", "", "", fmt.Errorf("could not determine package")
	}

	return parts[1], pkgParts[0], pkgParts[1], nil
}
