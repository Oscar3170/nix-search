package search

import (
	"encoding/json"
	"fmt"
)

// Document holds the `_source` fields the CLI renders. Anything else the
// backend returns is dropped here but kept verbatim in Hit.Raw, which is what
// `--output json` prints.
type Document struct {
	PackageAttrName    string   `json:"package_attr_name"`
	PackagePversion    string   `json:"package_pversion"`
	PackagePrograms    []string `json:"package_programs"`
	PackageMainProgram string   `json:"package_mainProgram"`
	PackageDescription string   `json:"package_description"`

	OptionName        string `json:"option_name"`
	OptionType        string `json:"option_type"`
	OptionDefault     string `json:"option_default"`
	OptionDescription string `json:"option_description"`
	OptionSource      string `json:"option_source"`
}

// Hit is one search result.
type Hit struct {
	// Doc is the parsed subset of `_source` used for display.
	Doc Document
	// Raw is the untouched `_source`, re-emitted by JSON output.
	Raw json.RawMessage
}

// Response is a parsed `_search` response.
type Response struct {
	// Total is the number of matching documents, which can exceed len(Hits).
	Total int
	Hits  []Hit
}

// total accepts both the object form Elasticsearch returns
// ({"value":3,"relation":"eq"}) and a bare number.
type total struct {
	value int
}

func (t *total) UnmarshalJSON(data []byte) error {
	var obj struct {
		Value int `json:"value"`
	}
	if err := json.Unmarshal(data, &obj); err == nil {
		t.value = obj.Value
		return nil
	}
	var n int
	if err := json.Unmarshal(data, &n); err != nil {
		return fmt.Errorf("invalid hits.total: %w", err)
	}
	t.value = n
	return nil
}

// parseResponse decodes a `_search` response body.
func parseResponse(body []byte) (*Response, error) {
	var payload struct {
		Hits struct {
			Total *total `json:"total"`
			Hits  []struct {
				Source json.RawMessage `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid search response: %w", err)
	}

	resp := &Response{Hits: make([]Hit, 0, len(payload.Hits.Hits))}
	for _, h := range payload.Hits.Hits {
		hit := Hit{Raw: h.Source}
		if len(h.Source) > 0 {
			if err := json.Unmarshal(h.Source, &hit.Doc); err != nil {
				return nil, fmt.Errorf("invalid hit _source: %w", err)
			}
		}
		resp.Hits = append(resp.Hits, hit)
	}

	// An absent total (a stub or proxied backend) is taken to be the number of
	// hits that came back.
	resp.Total = len(resp.Hits)
	if payload.Hits.Total != nil {
		resp.Total = payload.Hits.Total.value
	}
	return resp, nil
}
