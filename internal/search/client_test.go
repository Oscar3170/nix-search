package search

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSearchRequest checks the wire format the backend sees.
func TestSearchRequest(t *testing.T) {
	var gotPath, gotAuth, gotMethod string
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth, gotMethod = r.URL.Path, r.Header.Get("Authorization"), r.Method
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(readTestdata(t, "responses", "pkgs.json"))
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, SchemaVersion: 52, HTTP: server.Client()}
	resp, err := client.Search(t.Context(), Query{
		Text: "ripgrep", Channel: "unstable", Source: SourcePkgs,
		Mode: ModeFullText, Sort: SortRelevance, Size: 5,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if want := "/latest-52-nixos-unstable/_search"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if want := "Basic YVdWU0FMWHBadjpYOGdQSG56TDUyd0ZFZWt1eHNmUTljU2g="; gotAuth != want {
		t.Errorf("authorization = %q, want %q", gotAuth, want)
	}
	if body["size"] != float64(5) || body["from"] != float64(0) {
		t.Errorf("from/size = %v/%v, want 0/5", body["from"], body["size"])
	}
	if resp.Total != 11 || len(resp.Hits) != 5 {
		t.Errorf("total/hits = %d/%d, want 11/5", resp.Total, len(resp.Hits))
	}
	if got := resp.Hits[0].Doc.PackageAttrName; got != "ripgrep" {
		t.Errorf("first hit = %q, want ripgrep", got)
	}
}

// TestSearchResolvesStaleSchemaVersion covers the reason Search exists instead
// of a bare POST: the mapping version in the binary goes stale, and a 404 sends
// the client back to the alias catalog to find the version the backend serves.
func TestSearchResolvesStaleSchemaVersion(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		switch r.URL.Path {
		case "/_cat/aliases":
			io.WriteString(w, strings.Join([]string{
				"latest-51-nixos-unstable",
				"latest-52-nixos-unstable",
				"latest-52-nixos-26.05",
				"latest-52-group-manual",
				".kibana",
			}, "\n"))
		case "/latest-52-nixos-unstable/_search":
			w.Write(readTestdata(t, "responses", "pkgs.json"))
		default:
			http.Error(w, `{"error":"no such index"}`, http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, SchemaVersion: 1, HTTP: server.Client()}
	if _, err := client.Search(t.Context(), Query{
		Text: "ripgrep", Channel: "unstable", Source: SourcePkgs, Size: 5,
	}); err != nil {
		t.Fatalf("Search: %v", err)
	}

	want := []string{
		"/latest-1-nixos-unstable/_search",
		"/_cat/aliases",
		"/latest-52-nixos-unstable/_search",
	}
	if strings.Join(requests, ",") != strings.Join(want, ",") {
		t.Errorf("requests = %v, want %v", requests, want)
	}
}

// TestSearchUnknownChannel reports the channels the backend serves when the
// requested one has no index.
func TestSearchUnknownChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_cat/aliases" {
			io.WriteString(w, "latest-52-nixos-unstable\nlatest-52-nixos-26.05\n")
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, SchemaVersion: 52, HTTP: server.Client()}
	_, err := client.Search(t.Context(), Query{
		Text: "git", Channel: "25.05", Source: SourcePkgs, Size: 5,
	})
	if err == nil {
		t.Fatal("expected an error for a missing channel")
	}
	for _, want := range []string{`unknown channel "25.05"`, "26.05, unstable"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

// TestSearchHTTPError keeps the server's own error text, which is what makes a
// rejected query debuggable.
func TestSearchHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "parsing_exception: no [query] registered", http.StatusBadRequest)
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, SchemaVersion: 52, HTTP: server.Client()}
	_, err := client.Search(t.Context(), Query{
		Text: "git", Channel: "unstable", Source: SourcePkgs, Size: 5,
	})

	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("error = %v, want *HTTPError", err)
	}
	if httpErr.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", httpErr.StatusCode)
	}
	if !strings.Contains(httpErr.Body, "parsing_exception") {
		t.Errorf("body = %q, want the server message", httpErr.Body)
	}
}

// TestParseResponseAcceptsTotalForms covers the two shapes Elasticsearch uses
// for hits.total, plus a backend stub that omits it.
func TestParseResponseAcceptsTotalForms(t *testing.T) {
	cases := map[string]struct {
		body  string
		total int
	}{
		"object":  {`{"hits":{"total":{"value":42,"relation":"eq"},"hits":[]}}`, 42},
		"number":  {`{"hits":{"total":7,"hits":[]}}`, 7},
		"missing": {`{"hits":{"hits":[{"_source":{}},{"_source":{}}]}}`, 2},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resp, err := parseResponse([]byte(tc.body))
			if err != nil {
				t.Fatalf("parseResponse: %v", err)
			}
			if resp.Total != tc.total {
				t.Errorf("total = %d, want %d", resp.Total, tc.total)
			}
		})
	}
}

// TestNewClientSchemaVersionEnv documents the override the integration tests
// use to force alias resolution.
func TestNewClientSchemaVersionEnv(t *testing.T) {
	t.Setenv(schemaVersionEnv, "12")
	if got := NewClient().SchemaVersion; got != 12 {
		t.Errorf("SchemaVersion = %d, want 12", got)
	}
	t.Setenv(schemaVersionEnv, "not-a-number")
	if got := NewClient().SchemaVersion; got != DefaultSchemaVersion {
		t.Errorf("SchemaVersion = %d, want the default", got)
	}
}
