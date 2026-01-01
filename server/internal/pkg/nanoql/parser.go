package nanoql

import (
	"fmt"
)

// Parser parses NanoQL queries into an AST.
type Parser struct {
	lexer   *Lexer
	current Token
}

// Parse parses the input string and returns the AST root node.
func Parse(input string) (Node, error) {
	if input == "" {
		return nil, nil
	}
	p := &Parser{lexer: NewLexer(input)}
	p.advance()
	return p.parseOr()
}

func (p *Parser) advance() {
	p.current = p.lexer.NextToken()
}

// parseOr handles OR expressions (lowest precedence).
func (p *Parser) parseOr() (Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.current.Type == TokenOr {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = BinaryExpr{Op: "OR", Left: left, Right: right}
	}

	return left, nil
}

// parseAnd handles AND expressions.
func (p *Parser) parseAnd() (Node, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}

	for p.current.Type == TokenAnd {
		p.advance()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = BinaryExpr{Op: "AND", Left: left, Right: right}
	}

	return left, nil
}

// parseNot handles NOT expressions.
func (p *Parser) parseNot() (Node, error) {
	if p.current.Type == TokenNot {
		p.advance()
		expr, err := p.parseNot() // NOT is right-associative
		if err != nil {
			return nil, err
		}
		return NotExpr{Expr: expr}, nil
	}
	return p.parsePrimary()
}

// parsePrimary handles primary expressions: (expr), key:value, "string".
func (p *Parser) parsePrimary() (Node, error) {
	switch p.current.Type {
	case TokenLParen:
		p.advance()
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.current.Type != TokenRParen {
			return nil, fmt.Errorf("expected ')' but got %v", p.current)
		}
		p.advance()
		return expr, nil

	case TokenString:
		// Full-text search: "some text"
		value := p.current.Value
		p.advance()
		return MatchExpr{Key: "", Value: value, Op: "CONTAINS"}, nil

	case TokenIdent:
		key := p.current.Value
		p.advance()

		// Check for colon (key:value pattern)
		if p.current.Type == TokenColon {
			p.advance()
			return p.parseValue(key, "=")
		}

		// Check for != (key != value pattern)
		if p.current.Type == TokenNeq {
			p.advance()
			return p.parseValue(key, "!=")
		}

		// Bare identifier: treat as full-text search
		return MatchExpr{Key: "", Value: key, Op: "CONTAINS"}, nil

	case TokenEOF:
		return nil, nil

	default:
		return nil, fmt.Errorf("unexpected token: %v", p.current)
	}
}

// parseValue parses the value part after key: or key!=
func (p *Parser) parseValue(key, op string) (Node, error) {
	var value string

	switch p.current.Type {
	case TokenString:
		value = p.current.Value
		p.advance()
	case TokenIdent:
		value = p.current.Value
		p.advance()
	default:
		return nil, fmt.Errorf("expected value after '%s%s' but got %v", key, op, p.current)
	}

	return MatchExpr{Key: key, Value: value, Op: op}, nil
}
