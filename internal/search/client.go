package search

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/oscar3170/nix-search/internal/version"
)

const (
	// DefaultBaseURL is the public backend used by search.nixos.org.
	DefaultBaseURL = "https://search.nixos.org/backend"

	// DefaultSchemaVersion is the Elasticsearch mapping version baked into the
	// site. The site bumps it whenever the mapping changes and only keeps
	// aliases for the versions it still serves, so this value goes stale; a
	// stale value costs one 404 round trip because Search then re-resolves the
	// alias from the catalog (see discover).
	DefaultSchemaVersion = 52

	// schemaVersionEnv overrides DefaultSchemaVersion. It also exists so the
	// integration tests can force the alias-discovery path.
	schemaVersionEnv = "NIX_SEARCH_SCHEMA_VERSION"

	// backendUser and backendPassword are the public credentials embedded in
	// the search.nixos.org web app.
	backendUser     = "aWVSALXpZv"
	backendPassword = "X8gPHnzL52wFEekuxsfQ9cSh"

	// maxResponseBytes caps how much of a response is read; a full page of
	// long package descriptions is a few hundred kilobytes.
	maxResponseBytes = 32 << 20

	// maxErrorBody is how much of an error response is quoted back.
	maxErrorBody = 500
)

// HTTPError is a non-200 response from the backend.
type HTTPError struct {
	StatusCode int
	URL        string
	Body       string
}

func (e *HTTPError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("HTTP %d from %s", e.StatusCode, e.URL)
	}
	return fmt.Sprintf("HTTP %d from %s\n%s", e.StatusCode, e.URL, e.Body)
}

// Client talks to the Elasticsearch backend behind search.nixos.org.
type Client struct {
	// BaseURL is the backend root, without a trailing slash.
	BaseURL string
	// SchemaVersion selects the `latest-<version>-nixos-<channel>` index alias.
	SchemaVersion int
	// HTTP is the underlying HTTP client.
	HTTP *http.Client
	// UserAgent is sent with every request.
	UserAgent string
}

// NewClient returns a Client configured for the public backend.
func NewClient() *Client {
	client := &Client{
		BaseURL:       DefaultBaseURL,
		SchemaVersion: DefaultSchemaVersion,
		HTTP:          &http.Client{Timeout: 30 * time.Second},
		UserAgent:     "nix-search/" + version.Version,
	}
	// A malformed value is ignored: the built-in default is always usable.
	if v, err := strconv.Atoi(os.Getenv(schemaVersionEnv)); err == nil && v > 0 {
		client.SchemaVersion = v
	}
	return client
}

// Search runs one query, resolving the index alias again if the configured
// schema version is no longer served.
func (c *Client) Search(ctx context.Context, q Query) (*Response, error) {
	payload, err := json.Marshal(q.body())
	if err != nil {
		return nil, fmt.Errorf("encoding query: %w", err)
	}

	index := c.indexName(q.Channel, c.SchemaVersion)
	resp, err := c.do(ctx, index, payload)
	if err == nil {
		return resp, nil
	}

	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusNotFound {
		return nil, err
	}

	// A 404 means the channel is gone or the mapping version moved. The alias
	// catalog knows which of the two happened.
	catalog, catalogErr := c.discover(ctx)
	if catalogErr != nil {
		return nil, err // keep the original 404: it is the more useful one
	}
	latest, ok := catalog.newest(q.Channel)
	if !ok {
		channels := catalog.channels()
		if len(channels) == 0 {
			return nil, fmt.Errorf("unknown channel %q", q.Channel)
		}
		return nil, fmt.Errorf("unknown channel %q (available: %s)",
			q.Channel, strings.Join(channels, ", "))
	}
	if resolved := c.indexName(q.Channel, latest); resolved != index {
		return c.do(ctx, resolved, payload)
	}
	return nil, err
}

// indexName is the alias to search for a channel at a mapping schema version.
func (c *Client) indexName(channel string, schemaVersion int) string {
	return fmt.Sprintf("latest-%d-nixos-%s", schemaVersion, channel)
}

// do POSTs a search body to one index alias.
func (c *Client) do(ctx context.Context, index string, payload []byte) (*Response, error) {
	url := c.baseURL() + "/" + index + "/_search"
	resp, err := c.send(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			URL:        url,
			Body:       truncate(string(body), maxErrorBody),
		}
	}
	return parseResponse(body)
}

// discover lists the index aliases the backend currently serves.
func (c *Client) discover(ctx context.Context) (indexCatalog, error) {
	url := c.baseURL() + "/_cat/aliases?h=alias"
	resp, err := c.send(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			URL:        url,
			Body:       truncate(string(body), maxErrorBody),
		}
	}
	return parseAliasCatalog(string(body)), nil
}

// send issues an authenticated request to the backend.
func (c *Client) send(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Basic "+
		base64.StdEncoding.EncodeToString([]byte(backendUser+":"+backendPassword)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach %s: %w", c.BaseURL, err)
	}
	return resp, nil
}

func (c *Client) baseURL() string {
	return strings.TrimSuffix(c.BaseURL, "/")
}
