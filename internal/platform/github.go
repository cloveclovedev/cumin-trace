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
// on one page, in the order of creation, with the first 100 sub-issues of
// each. It is a query and changes nothing (CTR-TRC-001, CTR-TRC-003,
// CTR-TRC-007).
const requirementIssuesQuery = `query requirementIssues($owner: String!, $name: String!, $label: String!, $after: String) {
  repository(owner: $owner, name: $name) {
    issues(labels: [$label], states: [OPEN, CLOSED], first: 100, after: $after,
           orderBy: {field: CREATED_AT, direction: ASC}) {
      pageInfo { hasNextPage endCursor }
      nodes {
        id number state stateReason closedAt body
        milestone { title }
        subIssues(first: 100) {
          pageInfo { hasNextPage endCursor }
          nodes {
            id number state
            repository { nameWithOwner }
            milestone { title }
          }
        }
      }
    }
  }
}`

// moreSubIssuesQuery reads the sub-issues of one issue from the 101st on, 100
// on one page (CTR-TRC-003).
const moreSubIssuesQuery = `query moreSubIssues($id: ID!, $after: String) {
  node(id: $id) {
    ... on Issue {
      subIssues(first: 100, after: $after) {
        pageInfo { hasNextPage endCursor }
        nodes { id number state repository { nameWithOwner } milestone { title } }
      }
    }
  }
}`

// closersQuery reads the last closed event of each issue and the pull request
// that closed the issue, if a pull request did (CTR-TRC-003).
const closersQuery = `query closers($ids: [ID!]!) {
  nodes(ids: $ids) {
    ... on Issue {
      id
      timelineItems(itemTypes: [CLOSED_EVENT], last: 1) {
        nodes {
          ... on ClosedEvent {
            closer { ... on PullRequest { number repository { nameWithOwner } } }
          }
        }
      }
    }
  }
}`

// closersPerQuery is the number of issues in one closers query.
const closersPerQuery = 100

// Issue is one issue as GitHub gives it. The values are those of GitHub, such
// as "OPEN" and "NOT_PLANNED".
type Issue struct {
	ID          string
	Number      int
	State       string
	StateReason string
	// ClosedAt is the time when the issue was closed. It is the zero time
	// for an open issue.
	ClosedAt time.Time
	Body     string
	// Milestone is the title of the milestone. It is empty for no milestone.
	Milestone string
	SubIssues []SubIssue
}

// SubIssue is one sub-issue of an issue as GitHub gives it.
type SubIssue struct {
	ID string
	// Repository is the repository of the sub-issue, as <owner>/<name>.
	Repository string
	Number     int
	State      string
	// Milestone is the title of the milestone. It is empty for no milestone.
	Milestone string
	// ClosedBy is the pull request that closed the sub-issue. It is nil for
	// an open sub-issue and for one that no pull request closed.
	ClosedBy *PullRequest
}

// PullRequest is one pull request.
type PullRequest struct {
	// Repository is the repository of the pull request, as <owner>/<name>.
	Repository string
	Number     int
}

// subIssuesPage is one page of the sub-issues of an issue in a response.
type subIssuesPage struct {
	PageInfo pageInfo `json:"pageInfo"`
	Nodes    []*struct {
		ID         string `json:"id"`
		Number     int    `json:"number"`
		State      string `json:"state"`
		Repository struct {
			NameWithOwner string `json:"nameWithOwner"`
		} `json:"repository"`
		Milestone *struct {
			Title string `json:"title"`
		} `json:"milestone"`
	} `json:"nodes"`
}

// subIssues returns the sub-issues of the page.
func (p subIssuesPage) subIssues() []SubIssue {
	subIssues := make([]SubIssue, 0, len(p.Nodes))
	for _, node := range p.Nodes {
		// A connection can hold a null node, such as an issue that the token
		// cannot read.
		if node == nil {
			continue
		}
		subIssue := SubIssue{ID: node.ID, Repository: node.Repository.NameWithOwner, Number: node.Number, State: node.State}
		if node.Milestone != nil {
			subIssue.Milestone = node.Milestone.Title
		}
		subIssues = append(subIssues, subIssue)
	}
	return subIssues
}

// pageInfo tells whether a connection has another page.
type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

// next returns the cursor of the next page, or nil after the last page. A
// cursor that does not move would ask for the same page again without an
// end, so it is an error.
func (p pageInfo) next(after *string) (*string, error) {
	if !p.HasNextPage {
		return nil, nil
	}
	if p.EndCursor == "" || (after != nil && p.EndCursor == *after) {
		return nil, errors.New("GitHub: invalid page cursor")
	}
	cursor := p.EndCursor
	return &cursor, nil
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
// label, open and closed, across every page. Each issue holds all its
// sub-issues, and each closed sub-issue holds the pull request that closed it.
func (c *GitHubClient) IssuesWithLabel(owner, name, label string) ([]Issue, error) {
	var issues []Issue
	// more holds the cursor of the next page of sub-issues, by the index of
	// the issue.
	more := map[int]*string{}
	var after *string
	for {
		var data struct {
			Repository *struct {
				Issues struct {
					PageInfo pageInfo `json:"pageInfo"`
					Nodes    []struct {
						ID          string     `json:"id"`
						Number      int        `json:"number"`
						State       string     `json:"state"`
						StateReason *string    `json:"stateReason"`
						ClosedAt    *time.Time `json:"closedAt"`
						Body        string     `json:"body"`
						Milestone   *struct {
							Title string `json:"title"`
						} `json:"milestone"`
						SubIssues subIssuesPage `json:"subIssues"`
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
			issue := Issue{ID: node.ID, Number: node.Number, State: node.State, Body: node.Body, SubIssues: node.SubIssues.subIssues()}
			if node.StateReason != nil {
				issue.StateReason = *node.StateReason
			}
			if node.ClosedAt != nil {
				issue.ClosedAt = *node.ClosedAt
			}
			if node.Milestone != nil {
				issue.Milestone = node.Milestone.Title
			}
			next, err := node.SubIssues.PageInfo.next(nil)
			if err != nil {
				return nil, err
			}
			index := len(issues)
			issues = append(issues, issue)
			if next != nil {
				more[index] = next
			}
		}
		next, err := data.Repository.Issues.PageInfo.next(after)
		if err != nil {
			return nil, err
		}
		if next == nil {
			break
		}
		after = next
	}
	for i := range issues {
		if cursor, ok := more[i]; ok {
			rest, err := c.moreSubIssues(issues[i].ID, cursor)
			if err != nil {
				return nil, err
			}
			issues[i].SubIssues = append(issues[i].SubIssues, rest...)
		}
	}
	if err := c.setClosers(issues); err != nil {
		return nil, err
	}
	return issues, nil
}

// moreSubIssues returns the sub-issues of the issue with the id, from the
// page after the cursor to the last page.
func (c *GitHubClient) moreSubIssues(id string, after *string) ([]SubIssue, error) {
	var subIssues []SubIssue
	for {
		var data struct {
			Node *struct {
				SubIssues subIssuesPage `json:"subIssues"`
			} `json:"node"`
		}
		if err := c.query(moreSubIssuesQuery, map[string]any{"id": id, "after": after}, &data); err != nil {
			return nil, err
		}
		if data.Node == nil {
			return nil, fmt.Errorf("GitHub: no issue %s", id)
		}
		subIssues = append(subIssues, data.Node.SubIssues.subIssues()...)
		next, err := data.Node.SubIssues.PageInfo.next(after)
		if err != nil {
			return nil, err
		}
		if next == nil {
			return subIssues, nil
		}
		after = next
	}
}

// setClosers reads the pull request that closed each closed sub-issue, 100
// sub-issues in one query, and each sub-issue once. It sends no query when no
// sub-issue is closed.
func (c *GitHubClient) setClosers(issues []Issue) error {
	var ids []string
	seen := map[string]bool{}
	for _, issue := range issues {
		for _, subIssue := range issue.SubIssues {
			if subIssue.State == "CLOSED" && !seen[subIssue.ID] {
				seen[subIssue.ID] = true
				ids = append(ids, subIssue.ID)
			}
		}
	}
	closers := map[string]*PullRequest{}
	for start := 0; start < len(ids); start += closersPerQuery {
		var data struct {
			Nodes []*struct {
				ID            string `json:"id"`
				TimelineItems struct {
					Nodes []struct {
						Closer *struct {
							Number     *int `json:"number"`
							Repository struct {
								NameWithOwner string `json:"nameWithOwner"`
							} `json:"repository"`
						} `json:"closer"`
					} `json:"nodes"`
				} `json:"timelineItems"`
			} `json:"nodes"`
		}
		batch := ids[start:min(start+closersPerQuery, len(ids))]
		if err := c.query(closersQuery, map[string]any{"ids": batch}, &data); err != nil {
			return err
		}
		for _, node := range data.Nodes {
			if node == nil || len(node.TimelineItems.Nodes) == 0 {
				continue
			}
			// A closer that is no pull request, such as a commit, has no number.
			closer := node.TimelineItems.Nodes[len(node.TimelineItems.Nodes)-1].Closer
			if closer != nil && closer.Number != nil {
				closers[node.ID] = &PullRequest{Repository: closer.Repository.NameWithOwner, Number: *closer.Number}
			}
		}
	}
	for i := range issues {
		for j := range issues[i].SubIssues {
			subIssue := &issues[i].SubIssues[j]
			if subIssue.State == "CLOSED" {
				subIssue.ClosedBy = closers[subIssue.ID]
			}
		}
	}
	return nil
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
