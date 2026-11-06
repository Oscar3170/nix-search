package search

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// aliasRE matches the index aliases the backend exposes, e.g.
// `latest-52-nixos-unstable`.
var aliasRE = regexp.MustCompile(`^latest-(\d+)-nixos-(.+)$`)

// indexCatalog maps a channel to every mapping schema version that has an
// alias for it.
type indexCatalog map[string][]int

// parseAliasCatalog reads the alias names out of a `_cat/aliases?h=alias`
// response. The backend answers in plain text by default but switches to a JSON
// array when the request asks for JSON (which this client does), so both forms
// are accepted.
func parseAliasCatalog(out string) indexCatalog {
	catalog := make(indexCatalog)
	for _, alias := range aliasNames(out) {
		m := aliasRE.FindStringSubmatch(strings.TrimSpace(alias))
		if m == nil {
			continue
		}
		schemaVersion, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		catalog[m[2]] = append(catalog[m[2]], schemaVersion)
	}
	return catalog
}

// aliasNames returns the alias names of a catalog response.
func aliasNames(out string) []string {
	var items []struct {
		Alias string `json:"alias"`
	}
	if err := json.Unmarshal([]byte(out), &items); err == nil {
		names := make([]string, 0, len(items))
		for _, item := range items {
			names = append(names, item.Alias)
		}
		return names
	}
	return strings.Split(out, "\n")
}

// newest returns the highest alias schema version for channel.
func (c indexCatalog) newest(channel string) (int, bool) {
	versions := c[channel]
	if len(versions) == 0 {
		return 0, false
	}
	latest := versions[0]
	for _, v := range versions[1:] {
		if v > latest {
			latest = v
		}
	}
	return latest, true
}

// channels lists the channels that have at least one alias, sorted.
func (c indexCatalog) channels() []string {
	channels := make([]string, 0, len(c))
	for channel := range c {
		channels = append(channels, channel)
	}
	sort.Strings(channels)
	return channels
}
