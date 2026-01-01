package nanoql

// Node is the interface implemented by all AST nodes.
type Node interface {
	node() // marker method
}

// BinaryExpr represents a binary logical expression (AND, OR).
type BinaryExpr struct {
	Op    string // "AND" or "OR"
	Left  Node
	Right Node
}

func (BinaryExpr) node() {}

// MatchExpr represents a key:value match expression.
// If Key is empty, it represents a full-text search across all fields.
type MatchExpr struct {
	Key   string // Field name (e.g., "service", "level"). Empty for full-text.
	Value string // The value to match.
	Op    string // "=", "!=", or "CONTAINS"
}

func (MatchExpr) node() {}

// NotExpr represents a NOT expression that negates its inner expression.
type NotExpr struct {
	Expr Node
}

func (NotExpr) node() {}
