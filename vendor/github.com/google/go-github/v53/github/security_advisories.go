// Copyright 2023 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"context"
	"fmt"
)

type SecurityAdvisoriesService service

// RequestCVE requests a Common Vulnerabilities and Exposures (CVE) for a repository security advisory.
// The ghsaID is the GitHub Security Advisory identifier of the advisory.
//
// GitHub API docs: https://docs.github.com/en/rest/security-advisories/repository-advisories#request-a-cve-for-a-repository-security-advisory
func (s *SecurityAdvisoriesService) RequestCVE(ctx context.Context, owner, repo, ghsaID string) (*Response, error) {
	url := fmt.Sprintf("repos/%v/%v/security-advisories/%v/cve", owner, repo, ghsaID)

	req, err := s.client.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req, nil)
	if err != nil {
		if _, ok := err.(*AcceptedError); ok {
			return resp, nil
		}

		return resp, err
	}

	return resp, nil
}
