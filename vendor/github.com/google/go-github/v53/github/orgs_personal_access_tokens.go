// Copyright 2023 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"context"
	"fmt"
	"net/http"
)

// ReviewPersonalAccessTokenRequestOptions specifies the parameters to the ReviewPersonalAccessTokenRequest method.
type ReviewPersonalAccessTokenRequestOptions struct {
	Action string  `json:"action"`
	Reason *string `json:"reason,omitempty"`
}

// ReviewPersonalAccessTokenRequest approves or denies a pending request to access organization resources via a fine-grained personal access token.
// Only GitHub Apps can call this API, using the `organization_personal_access_token_requests: write` permission.
// `action` can be one of `approve` or `deny`.
//
// GitHub API docs: https://docs.github.com/en/rest/orgs/personal-access-tokens?apiVersion=2022-11-28#review-a-request-to-access-organization-resources-with-a-fine-grained-personal-access-token
func (s *OrganizationsService) ReviewPersonalAccessTokenRequest(ctx context.Context, org string, requestID int64, opts ReviewPersonalAccessTokenRequestOptions) (*Response, error) {
	u := fmt.Sprintf("orgs/%v/personal-access-token-requests/%v", org, requestID)

	req, err := s.client.NewRequest(http.MethodPost, u, &opts)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
