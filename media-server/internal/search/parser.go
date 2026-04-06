package search

import (
	"fmt"
	"strings"
	"unicode"
)

// Parser parses boolean query strings into AST
type Parser struct {
	input  string
	pos    int
	tokens []token
	cur    int
}

type tokenType int

const (
	tokenTag tokenType = iota
	tokenAnd
	tokenOr
	tokenNot
	tokenLParen
	tokenRParen
	tokenEOF
)

type token struct {
	typ   tokenType
	value string
	pos   int
}

// Parse parses a query string and returns the root QueryNode
func Parse(query string) (QueryNode, error) {
	p := &Parser{
		input: query,
		pos:   0,
	}

	// Tokenize
	if err := p.tokenize(); err != nil {
		return nil, err
	}

	// Parse expression
	node, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Ensure we consumed all tokens
	if p.current().typ != tokenEOF {
		return nil, fmt.Errorf("unexpected token at position %d: %s", p.current().pos, p.current().value)
	}

	return node, nil
}

// tokenize converts the input string into tokens
func (p *Parser) tokenize() error {
	p.tokens = []token{}
	runes := []rune(p.input)
	runePos := 0

	for runePos < len(runes) {
		// Skip whitespace
		if unicode.IsSpace(runes[runePos]) {
			runePos++
			continue
		}

		start := runePos
		ch := runes[runePos]

		switch {
		case ch == '(':
			p.tokens = append(p.tokens, token{typ: tokenLParen, value: "(", pos: start})
			runePos++
		case ch == ')':
			p.tokens = append(p.tokens, token{typ: tokenRParen, value: ")", pos: start})
			runePos++
		case ch == '"':
			// Handle quoted string (for multi-word tags)
			runePos++ // skip opening quote
			start = runePos
			for runePos < len(runes) && runes[runePos] != '"' {
				runePos++
			}
			if runePos >= len(runes) {
				return fmt.Errorf("unterminated quoted string at position %d", start-1)
			}
			value := string(runes[start:runePos])
			runePos++ // skip closing quote
			p.tokens = append(p.tokens, token{typ: tokenTag, value: value, pos: start - 1})
		default:
			// Read identifier (tag name or keyword)
			// Allow any printable character except whitespace and special parser chars
			for runePos < len(runes) {
				ch := runes[runePos]
				if unicode.IsSpace(ch) || ch == '(' || ch == ')' || ch == '"' {
					break
				}
				runePos++
			}

			value := string(runes[start:runePos])
			if value == "" {
				return fmt.Errorf("unexpected character at position %d: %c", runePos, ch)
			}

			upper := strings.ToUpper(value)

			switch upper {
			case "AND":
				p.tokens = append(p.tokens, token{typ: tokenAnd, value: value, pos: start})
			case "OR":
				p.tokens = append(p.tokens, token{typ: tokenOr, value: value, pos: start})
			case "NOT":
				p.tokens = append(p.tokens, token{typ: tokenNot, value: value, pos: start})
			default:
				p.tokens = append(p.tokens, token{typ: tokenTag, value: value, pos: start})
			}
		}
	}

	p.tokens = append(p.tokens, token{typ: tokenEOF, value: "", pos: runePos})
	return nil
}

// current returns the current token
func (p *Parser) current() token {
	if p.cur >= len(p.tokens) {
		return token{typ: tokenEOF, value: "", pos: len(p.input)}
	}
	return p.tokens[p.cur]
}

// advance moves to the next token
func (p *Parser) advance() {
	if p.cur < len(p.tokens) {
		p.cur++
	}
}

// expect checks if current token matches expected type and advances
func (p *Parser) expect(typ tokenType) error {
	if p.current().typ != typ {
		return fmt.Errorf("expected %v at position %d, got %v", typ, p.current().pos, p.current().typ)
	}
	p.advance()
	return nil
}

// parseExpression parses: term (OR term)*
func (p *Parser) parseExpression() (QueryNode, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}

	for p.current().typ == tokenOr {
		p.advance() // consume OR
		right, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		left = &OrNode{Left: left, Right: right}
	}

	return left, nil
}

// parseTerm parses: factor (AND factor)*
func (p *Parser) parseTerm() (QueryNode, error) {
	left, err := p.parseFactor()
	if err != nil {
		return nil, err
	}

	for p.current().typ == tokenAnd {
		p.advance() // consume AND
		right, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		left = &AndNode{Left: left, Right: right}
	}

	return left, nil
}

// parseFactor parses: NOT factor | primary
func (p *Parser) parseFactor() (QueryNode, error) {
	if p.current().typ == tokenNot {
		p.advance() // consume NOT
		child, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		return &NotNode{Child: child}, nil
	}

	return p.parsePrimary()
}

// parsePrimary parses: tag | ( expression )
func (p *Parser) parsePrimary() (QueryNode, error) {
	tok := p.current()

	switch tok.typ {
	case tokenTag:
		p.advance()
		return &TagNode{TagName: tok.value}, nil

	case tokenLParen:
		p.advance() // consume (
		node, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if err := p.expect(tokenRParen); err != nil {
			return nil, err
		}
		return node, nil

	default:
		return nil, fmt.Errorf("unexpected token at position %d: %s", tok.pos, tok.value)
	}
}
