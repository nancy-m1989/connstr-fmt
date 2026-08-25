package connstr

import "strings"

// aliasGroups lists sets of keys that different drivers accept
// interchangeably for the same setting, such as "Server" and "Data Source"
// both meaning the host to connect to. The first entry in each group is its
// canonical form.
var aliasGroups = [][]string{
	{"server", "data source"},
	{"uid", "user id"},
}

var aliasCanon = func() map[string]string {
	m := make(map[string]string)
	for _, group := range aliasGroups {
		for _, alias := range group {
			m[alias] = group[0]
		}
	}
	return m
}()

// canonicalKey returns the representative name for key's alias group, so
// that keys meaning the same setting compare equal regardless of which
// alias was used. Keys outside any known group are just normalized.
func canonicalKey(key string) string {
	norm := normalizeKey(key)
	if canon, ok := aliasCanon[norm]; ok {
		return canon
	}
	return norm
}

// normalizeKey folds case and collapses internal whitespace, so "User Id"
// and "user  id" compare equal.
func normalizeKey(key string) string {
	return strings.ToLower(strings.Join(strings.Fields(key), " "))
}
