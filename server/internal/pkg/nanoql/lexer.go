package nanoql

import (
	"strings"
	"unicode"
	"unicode/utf8"
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

	r, size := utf8.DecodeRuneInString(l.input[l.pos:])

	// Single-character tokens
	switch r {
	case ':':
		l.pos += size
		return Token{Type: TokenColon, Value: ":"}
	case '(':
		l.pos += size
		return Token{Type: TokenLParen, Value: "("}
	case ')':
		l.pos += size
		return Token{Type: TokenRParen, Value: ")"}
	case '!':
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.pos += 2
			return Token{Type: TokenNeq, Value: "!="}
		}
		// Single '!' not followed by '=' is treated as ident
		return l.readIdent()
	case '"':
		return l.readString()
	}

	// Keywords and identifiers (including non-ASCII like Chinese)
	if isIdentStart(r) {
		return l.readIdent()
	}

	// Unknown character, skip
	l.pos += size
	return l.NextToken()
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.pos:])
		if !unicode.IsSpace(r) {
			break
		}
		l.pos += size
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
	for l.pos < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.pos:])
		if !isIdentChar(r) {
			break
		}
		l.pos += size
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

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.'
}
