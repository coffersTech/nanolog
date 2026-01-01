package engine

import (
	"github.com/coffersTech/nanolog/server/internal/pkg/nanoql"
)

// MatchNanoQL is a bridge function that calls the NanoQL matcher.
func MatchNanoQL(node interface{}, row *LogRow) bool {
	if node == nil {
		return true
	}
	if n, ok := node.(nanoql.Node); ok {
		return nanoql.Match(n, row)
	}
	return true // If node is not valid NanoQL, pass through
}

// ParseNanoQL parses a query string into a NanoQL AST node.
// Returns nil if query is empty or parsing fails.
func ParseNanoQL(query string) (interface{}, error) {
	if query == "" {
		return nil, nil
	}
	return nanoql.Parse(query)
}
