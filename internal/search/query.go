package search

import (
	"fmt"
	"strings"
)

// The field list used by the website's package full-text query.
var fullTextField = []string{
	"package_attr_name^9", "package_attr_name.*^5.3999999999999995",
	"package_programs^9", "package_programs.*^5.3999999999999995",
	"package_mainProgram^9", "package_mainProgram.*^5.3999999999999995",
	"package_pname^6", "package_pname.*^3.5999999999999996",
	"package_description^1.3", "package_description.*^0.78",
	"package_longDescription^1", "package_longDescription.*^0.6",
}

// requestBody is the JSON document POSTed to the `_search` endpoint.
type requestBody struct {
	From    int              `json:"from"`
	Size    int              `json:"size"`
	Query   map[string]any   `json:"query"`
	Sort    []map[string]any `json:"sort,omitempty"`
	Rescore *rescore         `json:"rescore,omitempty"`
}

// rescore boosts short attribute paths, which the website uses to rank
// `ripgrep` above `emacsPackages.ripgrep`.
type rescore struct {
	WindowSize int      `json:"window_size"`
	Query      rescoreQ `json:"query"`
}

type rescoreQ struct {
	RescoreQuery       map[string]any `json:"rescore_query"`
	RescoreQueryWeight float64        `json:"rescore_query_weight"`
}

// body renders the Elasticsearch request body for the query.
func (q Query) body() requestBody {
	body := requestBody{From: 0, Size: q.Size, Query: q.query()}

	if q.Source != SourcePkgs {
		if q.Sort == SortAlpha {
			body.Sort = []map[string]any{{"option_name": "asc"}}
			return body
		}
		body.Sort = []map[string]any{{"_score": "desc"}}
		body.Rescore = lengthRescore(
			"doc['option_name'].size() == 0 ? 0 : 1.0 / doc['option_name'].value.length()",
			22.3,
		)
		return body
	}

	if q.Sort == SortAlpha {
		body.Sort = []map[string]any{{
			"package_attr_name": "asc",
			"package_pversion":  "asc",
		}}
		return body
	}
	body.Sort = []map[string]any{{"_score": "desc"}}
	body.Rescore = lengthRescore("1.0 / doc['package_attr_name'].value.length()", 20)
	return body
}

// lengthRescore scores each hit by a script, used to favour shorter names.
func lengthRescore(source string, weight float64) *rescore {
	return &rescore{
		WindowSize: 100,
		Query: rescoreQ{
			RescoreQuery: map[string]any{
				"function_score": map[string]any{
					"script_score": map[string]any{
						"script": map[string]any{"source": source},
					},
				},
			},
			RescoreQueryWeight: weight,
		},
	}
}

// query selects the query shape for the source and match mode.
func (q Query) query() map[string]any {
	if q.Source != SourcePkgs {
		return optionsQuery(q.Text, q.Source.DocType())
	}
	switch q.Mode {
	case ModeName:
		return nameQuery(q.Text)
	case ModePrograms:
		return programsQuery(q.Text)
	default:
		return fullTextQuery(q.Text)
	}
}

// fullTextQuery mirrors the website's main package search: a dis_max over a
// cross-fields match, a fuzzy match and an attribute-name wildcard, plus
// should clauses that boost exact names and popular packages.
func fullTextQuery(text string) map[string]any {
	return map[string]any{
		"bool": map[string]any{
			"filter":   []any{termClause("type", "package", "filter_packages")},
			"must_not": []any{},
			"must": []any{
				map[string]any{
					"dis_max": map[string]any{
						"tie_breaker": 0.7,
						"queries": []any{
							map[string]any{
								"multi_match": map[string]any{
									"type":                                "cross_fields",
									"query":                               text,
									"analyzer":                            "whitespace",
									"auto_generate_synonyms_phrase_query": false,
									"operator":                            "and",
									"_name":                               "multi_match_strings",
									"fields":                              fullTextField,
								},
							},
							map[string]any{
								"multi_match": map[string]any{
									"type":          "best_fields",
									"query":         text,
									"fuzziness":     "1",
									"prefix_length": 1,
									"operator":      "and",
									"_name":         "fuzzy_strings",
									"fields": []string{
										"package_attr_name^0.45",
										"package_programs^0.45",
										"package_mainProgram^0.45",
										"package_pname^0.30000000000000004",
									},
								},
							},
							map[string]any{
								"wildcard": map[string]any{
									"package_attr_name": map[string]any{
										"value":            "*" + text + "*",
										"case_insensitive": true,
									},
								},
							},
						},
					},
				},
			},
			"should": []any{
				map[string]any{
					"term": map[string]any{
						"package_attr_name": map[string]any{"value": text, "boost": 100},
					},
				},
				map[string]any{
					"prefix": map[string]any{
						"package_attr_name": map[string]any{
							"value":            text,
							"boost":            20,
							"case_insensitive": true,
						},
					},
				},
				rankFeature("package_repology_repos", 5, "popularity_package_repology_repos", 20),
				rankFeature("package_dep_count", 5, "popularity_package_dep_count", 1000),
			},
		},
	}
}

// nameQuery is the edge-ngram attribute/package name match.
func nameQuery(text string) map[string]any {
	return map[string]any{
		"bool": map[string]any{
			"filter": []any{termClause("type", "package", "")},
			"must": []any{
				map[string]any{
					"multi_match": map[string]any{
						"query":    text,
						"type":     "best_fields",
						"operator": "and",
						"fields":   []string{"package_attr_name.edge^4", "package_pname.edge^3"},
					},
				},
			},
		},
	}
}

// programsQuery finds packages by the executables they ship.
func programsQuery(text string) map[string]any {
	return map[string]any{
		"bool": map[string]any{
			"filter": []any{termClause("type", "package", "")},
			"should": []any{
				map[string]any{
					"multi_match": map[string]any{
						"query":         text,
						"type":          "best_fields",
						"fuzziness":     "1",
						"prefix_length": 1,
						"operator":      "and",
						"fields": []string{
							"package_programs^4",
							"package_mainProgram^4",
							"package_programs.*^2",
							"package_mainProgram.*^2",
						},
					},
				},
				map[string]any{
					"wildcard": map[string]any{
						"package_programs": map[string]any{
							"value":            "*" + text + "*",
							"case_insensitive": true,
						},
					},
				},
			},
			"minimum_should_match": 1,
		},
	}
}

// optionsQuery mirrors the website's Options tab: a boosted dis_max over
// reversed attribute-path prefixes and `option_name.edge` wildcards, plus
// heavily boosted should clauses for the exact dotted path, the trailing word
// and an "enable" prior.
func optionsQuery(text, docType string) map[string]any {
	words := strings.Fields(text)
	joined := strings.Join(words, " ")
	dotted := strings.Join(words, ".")

	// The website drops the last word for the reversed attribute-path prefix
	// and keeps everything but the first word for the phrase matches.
	var head, tail string
	if len(words) > 1 {
		head = words[0]
		tail = strings.Join(words[1:], " ")
	}

	// `_`/`-` spelling variants, as option names mix both conventions.
	var variants []string
	seen := make(map[string]struct{}, 3*len(words))
	for _, w := range words {
		for _, v := range []string{strings.ReplaceAll(w, "_", "-"), strings.ReplaceAll(w, "-", "_"), w} {
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			variants = append(variants, v)
		}
	}

	disMax := []any{
		map[string]any{
			"prefix": map[string]any{
				"option_name.attr_path_reverse": map[string]any{
					"value": "-" + strings.Join(words, "-"),
					"boost": 26.4,
				},
			},
		},
	}
	disMax = append(disMax, edgeWildcards(variants)...)
	disMax = append(disMax, multiMatch("best_fields", joined,
		[]string{"option_description^1"}, 1e-4,
		map[string]any{"fuzziness": "1", "prefix_length": 1}))

	must := []any{
		map[string]any{
			"dis_max": map[string]any{
				"boost":       8.33,
				"tie_breaker": 0.509,
				"queries":     disMax,
			},
		},
	}

	should := []any{
		map[string]any{
			"wildcard": map[string]any{
				"option_name.attr_path": map[string]any{
					"value": "*" + dotted + "*",
					"boost": 0.546,
				},
			},
		},
		map[string]any{
			"wildcard": map[string]any{
				"option_name.attr_path_reverse": map[string]any{
					"value": "*" + dotted + ".package*",
					"boost": 0.111,
				},
			},
		},
		constantScore(0.128, multiMatch(
			"phrase_prefix", "enable",
			[]string{"option_name.edge^11.6", "option_name.attr_path_reverse^682"}, 5.45,
			map[string]any{"operator": "or"},
		)),
	}
	if tail != "" {
		should = append(should, map[string]any{
			"prefix": map[string]any{
				"option_name.attr_path_reverse": map[string]any{
					"value":            tail,
					"boost":            206,
					"case_insensitive": true,
				},
			},
		})
	}
	if head != "" {
		should = append(should,
			constantScore(0.0682, multiMatch(
				"phrase", head,
				[]string{"option_description^11.6", "option_description.*^0.65", "option_name^630"},
				105, nil,
			)),
			constantScore(0.0975, multiMatch(
				"cross_fields", head,
				[]string{
					"option_name.edge^11.6",
					"option_description.*^0.542",
					"option_name.attr_path_reverse^746",
				}, 39.3, nil,
			)),
		)
	}

	return map[string]any{
		"bool": map[string]any{
			"filter": []any{termClause("type", docType, filterName(docType))},
			"must":   must,
			"should": should,
		},
	}
}

// filterName is the `_name` the website gives the document type filter, which
// shows up in profiling output.
func filterName(docType string) string {
	return "filter_" + strings.ReplaceAll(docType, "-", "_") + "s"
}

// termClause builds a term filter, optionally named.
func termClause(field, value, name string) map[string]any {
	if name == "" {
		return map[string]any{"term": map[string]any{field: value}}
	}
	return map[string]any{"term": map[string]any{field: map[string]any{
		"value": value,
		"_name": name,
	}}}
}

// edgeWildcards builds one `option_name.edge` wildcard per spelling variant.
func edgeWildcards(variants []string) []any {
	clauses := make([]any, 0, len(variants))
	for _, v := range variants {
		clauses = append(clauses, map[string]any{
			"wildcard": map[string]any{
				"option_name.edge": map[string]any{"value": "*" + v + "*"},
			},
		})
	}
	return clauses
}

func multiMatch(mtype, text string, fields []string, boost float64, extra map[string]any) map[string]any {
	body := map[string]any{
		"type":   mtype,
		"query":  text,
		"fields": fields,
		"boost":  boost,
	}
	for k, v := range extra {
		body[k] = v
	}
	return map[string]any{"multi_match": body}
}

func constantScore(boost float64, filter map[string]any) map[string]any {
	return map[string]any{
		"constant_score": map[string]any{"boost": boost, "filter": filter},
	}
}

func rankFeature(field string, boost float64, name string, pivot int) map[string]any {
	return map[string]any{
		"rank_feature": map[string]any{
			"field":      field,
			"boost":      boost,
			"_name":      name,
			"saturation": map[string]any{"pivot": pivot},
		},
	}
}

// String describes the query for error messages and tests.
func (q Query) String() string {
	return fmt.Sprintf("%s/%s/%s/%s", q.Source, q.Mode, q.Sort, q.Channel)
}
