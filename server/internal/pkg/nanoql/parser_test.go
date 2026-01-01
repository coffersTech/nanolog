package nanoql

import (
	"testing"
)

// testLogRow implements LogRecord for testing
type testLogRow struct {
	timestamp int64
	level     uint8
	service   string
	host      string
	message   string
}

func (r *testLogRow) GetTimestamp() int64 { return r.timestamp }
func (r *testLogRow) GetLevel() uint8     { return r.level }
func (r *testLogRow) GetService() string  { return r.service }
func (r *testLogRow) GetHost() string     { return r.host }
func (r *testLogRow) GetMessage() string  { return r.message }

func TestLexer(t *testing.T) {
	tests := []struct {
		input    string
		expected []TokenType
	}{
		{"service:order", []TokenType{TokenIdent, TokenColon, TokenIdent, TokenEOF}},
		{`level:"ERROR"`, []TokenType{TokenIdent, TokenColon, TokenString, TokenEOF}},
		{"a AND b", []TokenType{TokenIdent, TokenAnd, TokenIdent, TokenEOF}},
		{"a OR b", []TokenType{TokenIdent, TokenOr, TokenIdent, TokenEOF}},
		{"NOT a", []TokenType{TokenNot, TokenIdent, TokenEOF}},
		{"(a)", []TokenType{TokenLParen, TokenIdent, TokenRParen, TokenEOF}},
		{`key!="value"`, []TokenType{TokenIdent, TokenNeq, TokenString, TokenEOF}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			for i, expected := range tt.expected {
				tok := lexer.NextToken()
				if tok.Type != expected {
					t.Errorf("token %d: expected %v, got %v (%q)", i, expected, tok.Type, tok.Value)
				}
			}
		})
	}
}

func TestParseSimple(t *testing.T) {
	tests := []struct {
		input string
		check func(Node) bool
	}{
		{
			input: "service:order",
			check: func(n Node) bool {
				m, ok := n.(MatchExpr)
				return ok && m.Key == "service" && m.Value == "order" && m.Op == "="
			},
		},
		{
			input: `level:"ERROR"`,
			check: func(n Node) bool {
				m, ok := n.(MatchExpr)
				return ok && m.Key == "level" && m.Value == "ERROR" && m.Op == "="
			},
		},
		{
			input: `"timeout"`,
			check: func(n Node) bool {
				m, ok := n.(MatchExpr)
				return ok && m.Key == "" && m.Value == "timeout" && m.Op == "CONTAINS"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if !tt.check(node) {
				t.Errorf("check failed for input %q, got: %+v", tt.input, node)
			}
		})
	}
}

func TestParseCompound(t *testing.T) {
	node, err := Parse("service:order AND level:ERROR")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	bin, ok := node.(BinaryExpr)
	if !ok || bin.Op != "AND" {
		t.Fatalf("expected BinaryExpr AND, got %+v", node)
	}

	left, ok := bin.Left.(MatchExpr)
	if !ok || left.Key != "service" || left.Value != "order" {
		t.Errorf("left expected service:order, got %+v", left)
	}

	right, ok := bin.Right.(MatchExpr)
	if !ok || right.Key != "level" || right.Value != "ERROR" {
		t.Errorf("right expected level:ERROR, got %+v", right)
	}
}

func TestParseParentheses(t *testing.T) {
	node, err := Parse("service:order AND (level:ERROR OR level:WARN)")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	bin, ok := node.(BinaryExpr)
	if !ok || bin.Op != "AND" {
		t.Fatalf("expected AND at root, got %+v", node)
	}

	rightBin, ok := bin.Right.(BinaryExpr)
	if !ok || rightBin.Op != "OR" {
		t.Errorf("expected OR on right, got %+v", bin.Right)
	}
}

func TestParseNot(t *testing.T) {
	node, err := Parse("NOT level:DEBUG")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	not, ok := node.(NotExpr)
	if !ok {
		t.Fatalf("expected NotExpr, got %+v", node)
	}

	m, ok := not.Expr.(MatchExpr)
	if !ok || m.Key != "level" || m.Value != "DEBUG" {
		t.Errorf("expected level:DEBUG, got %+v", not.Expr)
	}
}

func TestMatch(t *testing.T) {
	row := &testLogRow{
		timestamp: 1234567890,
		level:     3, // ERROR
		service:   "order-service",
		host:      "192.168.1.1",
		message:   "Connection timeout occurred",
	}

	tests := []struct {
		query    string
		expected bool
	}{
		{"service:order-service", true},
		{"service:payment", false},
		{"level:ERROR", true},
		{"level:INFO", false},
		{`"timeout"`, true},
		{`"success"`, false},
		{"service:order-service AND level:ERROR", true},
		{"service:order-service AND level:INFO", false},
		{"service:payment OR level:ERROR", true},
		{"NOT level:DEBUG", true},
		{"NOT level:ERROR", false},
		{`host:"192.168.1.1"`, true},
		{`msg:"timeout"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			node, err := Parse(tt.query)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			result := Match(node, row)
			if result != tt.expected {
				t.Errorf("Match(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}

func TestMatchCaseInsensitive(t *testing.T) {
	row := &testLogRow{
		level:   3,
		service: "OrderService",
		message: "REQUEST completed",
	}

	tests := []struct {
		query    string
		expected bool
	}{
		{"service:orderservice", true},
		{"service:ORDERSERVICE", true},
		{"level:error", true},
		{"level:Error", true},
		{`"request"`, true},
		{`"REQUEST"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			node, err := Parse(tt.query)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if Match(node, row) != tt.expected {
				t.Errorf("Match(%q) failed", tt.query)
			}
		})
	}
}
