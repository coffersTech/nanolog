package nanoql

import (
	"strconv"
	"strings"
)

// LogRecord is an interface for log entries that can be matched.
// This decouples nanoql from the engine package.
type LogRecord interface {
	GetTimestamp() int64
	GetLevel() uint8
	GetService() string
	GetHost() string
	GetMessage() string
}

// Match evaluates the AST node against a LogRecord and returns true if it matches.
func Match(node Node, row LogRecord) bool {
	if node == nil {
		return true // No filter means match all
	}

	switch n := node.(type) {
	case BinaryExpr:
		return evalBinary(n, row)
	case MatchExpr:
		return evalMatch(n, row)
	case NotExpr:
		return !Match(n.Expr, row)
	default:
		return false
	}
}

func evalBinary(expr BinaryExpr, row LogRecord) bool {
	left := Match(expr.Left, row)
	right := Match(expr.Right, row)

	switch expr.Op {
	case "AND":
		return left && right
	case "OR":
		return left || right
	default:
		return false
	}
}

func evalMatch(expr MatchExpr, row LogRecord) bool {
	// Full-text search (no key specified)
	if expr.Key == "" {
		return matchFullText(expr.Value, row)
	}

	// Get the field value
	fieldValue := getFieldValue(expr.Key, row)

	// Evaluate based on operator
	switch expr.Op {
	case "=":
		return matchEqual(fieldValue, expr.Value)
	case "!=":
		return !matchEqual(fieldValue, expr.Value)
	case "CONTAINS":
		return containsIgnoreCase(fieldValue, expr.Value)
	default:
		return matchEqual(fieldValue, expr.Value)
	}
}

// getFieldValue returns the value of a field by name.
func getFieldValue(key string, row LogRecord) string {
	switch strings.ToLower(key) {
	case "service", "svc":
		return row.GetService()
	case "host", "ip", "hostname":
		return row.GetHost()
	case "message", "msg":
		return row.GetMessage()
	case "level", "lvl":
		return levelToString(row.GetLevel())
	case "timestamp", "ts":
		return strconv.FormatInt(row.GetTimestamp(), 10)
	default:
		return ""
	}
}

// matchEqual performs case-insensitive equality check.
func matchEqual(fieldValue, queryValue string) bool {
	return strings.EqualFold(fieldValue, queryValue)
}

// containsIgnoreCase checks if haystack contains needle (case-insensitive).
func containsIgnoreCase(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

// matchFullText searches across all fields.
func matchFullText(query string, row LogRecord) bool {
	q := strings.ToLower(query)
	fields := []string{
		row.GetService(),
		row.GetHost(),
		row.GetMessage(),
		levelToString(row.GetLevel()),
	}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}

// levelToString converts log level number to string.
func levelToString(level uint8) string {
	switch level {
	case 0:
		return "DEBUG"
	case 1:
		return "INFO"
	case 2:
		return "WARN"
	case 3:
		return "ERROR"
	case 4:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}
