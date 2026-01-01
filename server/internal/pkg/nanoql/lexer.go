package nanoql

import (
	"strings"
	"unicode"
)

// TokenType represents the type of a lexical token.
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent
	TokenString
	TokenColon
	TokenLParen
	TokenRParen
	TokenAnd
	TokenOr
	TokenNot
	TokenNeq // !=
)

// Token represents a lexical token.
type Token struct {
	Type  TokenType
	Value string
}

// Lexer tokenizes NanoQL input.
type Lexer struct {
	input string
	pos   int
}

// NewLexer creates a new Lexer for the given input.
func NewLexer(input string) *Lexer {
	return &Lexer{input: input, pos: 0}
}

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF}
	}

	ch := l.input[l.pos]

	// Single-character tokens
	switch ch {
	case ':':
		l.pos++
		return Token{Type: TokenColon, Value: ":"}
	case '(':
		l.pos++
		return Token{Type: TokenLParen, Value: "("}
	case ')':
		l.pos++
		return Token{Type: TokenRParen, Value: ")"}
	case '!':
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.pos += 2
			return Token{Type: TokenNeq, Value: "!="}
		}
		// Single '!' not followed by '=' is an error, treat as ident for now
		return l.readIdent()
	case '"':
		return l.readString()
	}

	// Keywords and identifiers
	if isIdentStart(ch) {
		return l.readIdent()
	}

	// Unknown character, skip
	l.pos++
	return l.NextToken()
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) {
		l.pos++
	}
}

func (l *Lexer) readString() Token {
	l.pos++ // skip opening quote
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != '"' {
		if l.input[l.pos] == '\\' && l.pos+1 < len(l.input) {
			l.pos += 2 // skip escaped char
			continue
		}
		l.pos++
	}
	value := l.input[start:l.pos]
	if l.pos < len(l.input) {
		l.pos++ // skip closing quote
	}
	return Token{Type: TokenString, Value: value}
}

func (l *Lexer) readIdent() Token {
	start := l.pos
	for l.pos < len(l.input) && isIdentChar(l.input[l.pos]) {
		l.pos++
	}
	value := l.input[start:l.pos]

	// Check for keywords
	upper := strings.ToUpper(value)
	switch upper {
	case "AND":
		return Token{Type: TokenAnd, Value: upper}
	case "OR":
		return Token{Type: TokenOr, Value: upper}
	case "NOT":
		return Token{Type: TokenNot, Value: upper}
	}

	return Token{Type: TokenIdent, Value: value}
}

func isIdentStart(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}

func isIdentChar(ch byte) bool {
	r := rune(ch)
	return unicode.IsLetter(r) || unicode.IsDigit(r) || ch == '_' || ch == '-' || ch == '.'
}
