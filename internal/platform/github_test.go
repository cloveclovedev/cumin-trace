package platform

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// CTR-TRC-007: a transport error names its kind, and holds neither the URL
// nor the token.
func TestATransportErrorNamesItsKindWithoutTheURLOrTheToken(t *testing.T) {
	const token = "test-token-0123456789"
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"a timeout", fmt.Errorf("wrap: %w", context.DeadlineExceeded), "timeout"},
		{"a DNS failure", &net.DNSError{Err: "no such host", Name: "api.example.test"}, "DNS lookup failed"},
		{"another failure", errors.New("connection refused"), "connection failed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := failureKind(test.err)
			if got != test.want {
				t.Errorf("got %q; want %q", got, test.want)
			}
		})
	}

	// A closed server refuses the connection on any machine.
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	_, err := NewGitHubClient(server.URL, token).IssuesWithLabel("octocat", "Hello-World", "req")
	if err == nil || err.Error() != "GitHub: cannot connect (connection failed)" || strings.Contains(err.Error(), token) {
		t.Errorf("got %v; want GitHub: cannot connect (connection failed)", err)
	}
}
