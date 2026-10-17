// Package platform reads the providers that cumin-trace depends on. The types
// of a provider stop here: the functions return the Go types of this package.
package platform

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// DefaultGraphQLURL is the URL of the GraphQL API of GitHub.
const DefaultGraphQLURL = "https://api.github.com/graphql"

// requirementIssuesQuery reads the issues with a label, open and closed, 100
// on one page, in the order of creation. It is a query and changes nothing
// (CTR-TRC-001, CTR-TRC-007).
const requirementIssuesQuery = `query requirementIssues($owner: String!, $name: String!, $label: String!, $after: String) {
  repository(owner: $owner, name: $name) {
    issues(labels: [$label], states: [OPEN, CLOSED], first: 100, after: $after,
           orderBy: {field: CREATED_AT, direction: ASC}) {
      pageInfo { hasNextPage endCursor }
      nodes {
        id number state stateReason body
        milestone { title }
      }
    }
  }
}`

// Issue is one issue as GitHub gives it. The values are those of GitHub, such
// as "OPEN" and "NOT_PLANNED".
type Issue struct {
	ID          string
	Number      int
	State       string
	StateReason string
	Body        string
	// Milestone is the title of the milestone. It is empty for no milestone.
	Milestone string
}

// GitHubClient reads GitHub through the GraphQL API. It only sends queries.
type GitHubClient struct {
	url   string
	token string
	http  *http.Client
}

// NewGitHubClient returns a client of the GraphQL API at url. The token goes
// only into the Authorization header.
func NewGitHubClient(url, token string) *GitHubClient {
	return &GitHubClient{url: url, token: token, http: &http.Client{Timeout: 60 * time.Second}}
}

// IssuesWithLabel returns the issues of the repository owner/name with the
// label, open and closed, across every page.
func (c *GitHubClient) IssuesWithLabel(owner, name, label string) ([]Issue, error) {
	var issues []Issue
	var after *string
	for {
		var data struct {
			Repository *struct {
				Issues struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []struct {
						ID          string  `json:"id"`
						Number      int     `json:"number"`
						State       string  `json:"state"`
						StateReason *string `json:"stateReason"`
						Body        string  `json:"body"`
						Milestone   *struct {
							Title string `json:"title"`
						} `json:"milestone"`
					} `json:"nodes"`
				} `json:"issues"`
			} `json:"repository"`
		}
		variables := map[string]any{"owner": owner, "name": name, "label": label, "after": after}
		if err := c.query(requirementIssuesQuery, variables, &data); err != nil {
			return nil, err
		}
		if data.Repository == nil {
			return nil, fmt.Errorf("GitHub: no repository %s/%s", owner, name)
		}
		for _, node := range data.Repository.Issues.Nodes {
			issue := Issue{ID: node.ID, Number: node.Number, State: node.State, Body: node.Body}
			if node.StateReason != nil {
				issue.StateReason = *node.StateReason
			}
			if node.Milestone != nil {
				issue.Milestone = node.Milestone.Title
			}
			issues = append(issues, issue)
		}
		page := data.Repository.Issues.PageInfo
		if !page.HasNextPage {
			return issues, nil
		}
		// A cursor that does not move would ask for the same page again
		// without an end.
		if page.EndCursor == "" || (after != nil && page.EndCursor == *after) {
			return nil, errors.New("GitHub: invalid page cursor")
		}
		cursor := page.EndCursor
		after = &cursor
	}
}

// query sends one query and decodes its data into data. An error holds the
// HTTP status or the first message of the GraphQL errors, and never the token.
func (c *GitHubClient) query(query string, variables map[string]any, data any) error {
	payload, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, c.url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("GitHub: invalid URL")
	}
	request.Header.Set("Authorization", "bearer "+c.token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "cumin-trace")
	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("GitHub: cannot connect (%s)", failureKind(err))
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("GitHub: cannot read the response")
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub: HTTP %d", response.StatusCode)
	}
	var result struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("GitHub: invalid response")
	}
	if len(result.Errors) > 0 {
		return fmt.Errorf("GitHub: %s", result.Errors[0].Message)
	}
	if err := json.Unmarshal(result.Data, data); err != nil {
		return fmt.Errorf("GitHub: invalid response")
	}
	return nil
}

// failureKind names the kind of a failure of the transport. The name holds
// neither the URL nor the token.
func failureKind(err error) string {
	var dnsErr *net.DNSError
	var certErr *tls.CertificateVerificationError
	var unknownAuthority x509.UnknownAuthorityError
	var hostnameErr x509.HostnameError
	var invalidCert x509.CertificateInvalidError
	var recordErr tls.RecordHeaderError
	var netErr net.Error
	switch {
	case errors.As(err, &netErr) && netErr.Timeout():
		return "timeout"
	case errors.As(err, &dnsErr):
		return "DNS lookup failed"
	case errors.As(err, &certErr), errors.As(err, &unknownAuthority), errors.As(err, &hostnameErr),
		errors.As(err, &invalidCert), errors.As(err, &recordErr):
		return "TLS failed"
	default:
		return "connection failed"
	}
}
