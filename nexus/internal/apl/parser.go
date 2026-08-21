package apl

import (
	"fmt"
	"strings"

	"github.com/autorix/nexus/internal/core"
)

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent
	TokenString
	TokenClass
	TokenImplements
	TokenNamespace
	TokenThis
	TokenCtx
	TokenSubject
	TokenRelated
	TokenPermits
	TokenIncludes
	TokenTraverse
	TokenLBrace
	TokenRBrace
	TokenLParen
	TokenRParen
	TokenLBracket
	TokenRBracket
	TokenColon
	TokenSemicolon
	TokenComma
	TokenEqual
	TokenArrow
	TokenDot
	TokenOr
	TokenAnd
	TokenError
)

type Token struct {
	Type    TokenType
	Literal string
}

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte
}

func NewLexer(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	switch l.ch {
	case '{':
		tok = newToken(TokenLBrace, l.ch)
	case '}':
		tok = newToken(TokenRBrace, l.ch)
	case '(':
		tok = newToken(TokenLParen, l.ch)
	case ')':
		tok = newToken(TokenRParen, l.ch)
	case '[':
		tok = newToken(TokenLBracket, l.ch)
	case ']':
		tok = newToken(TokenRBracket, l.ch)
	case ':':
		tok = newToken(TokenColon, l.ch)
	case ';':
		tok = newToken(TokenSemicolon, l.ch)
	case ',':
		tok = newToken(TokenComma, l.ch)
	case '.':
		tok = newToken(TokenDot, l.ch)
	case '=':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenArrow, Literal: string(ch) + string(l.ch)}
		} else {
			tok = newToken(TokenEqual, l.ch)
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenOr, Literal: string(ch) + string(l.ch)}
		} else {
			tok = newToken(TokenError, l.ch)
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenAnd, Literal: string(ch) + string(l.ch)}
		} else {
			tok = newToken(TokenError, l.ch)
		}
	case '\'', '"', '`':
		tok.Type = TokenString
		tok.Literal = l.readString(l.ch)
	case 0:
		tok.Type = TokenEOF
		tok.Literal = ""
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = lookupIdent(tok.Literal)
			return tok
		} else {
			tok = newToken(TokenError, l.ch)
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readString(quote byte) string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == quote || l.ch == 0 {
			break
		}
	}
	return l.input[position:l.position]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
	// skip comments
	for l.ch == '/' && (l.peekChar() == '/' || l.peekChar() == '*') {
		if l.peekChar() == '/' {
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
		} else if l.peekChar() == '*' {
			l.readChar() // *
			l.readChar()
			for !(l.ch == '*' && l.peekChar() == '/') && l.ch != 0 {
				l.readChar()
			}
			l.readChar() // *
			l.readChar() // /
		}
		for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
			l.readChar()
		}
	}
}

func isLetter(ch byte) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func newToken(tokenType TokenType, ch byte) Token {
	return Token{Type: tokenType, Literal: string(ch)}
}

var keywords = map[string]TokenType{
	"class":      TokenClass,
	"implements": TokenImplements,
	"Namespace":  TokenNamespace,
	"this":       TokenThis,
	"ctx":        TokenCtx,
	"subject":    TokenSubject,
	"related":    TokenRelated,
	"permits":    TokenPermits,
	"includes":   TokenIncludes,
	"traverse":   TokenTraverse,
}

func lookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return TokenIdent
}

type Parser struct {
	l         *Lexer
	curToken  Token
	peekToken Token
	errors    []string
}

func NewParser(l *Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
	
	// Skip imports completely
	if p.curToken.Type == TokenIdent && p.curToken.Literal == "import" {
		for p.curToken.Type != TokenSemicolon && p.curToken.Type != TokenEOF {
			p.curToken = p.peekToken
			p.peekToken = p.l.NextToken()
		}
		if p.curToken.Type == TokenSemicolon {
			p.curToken = p.peekToken
			p.peekToken = p.l.NextToken()
		}
	}
}

func (p *Parser) expectPeek(t TokenType) bool {
	if p.peekToken.Type == t {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t TokenType) {
	msg := fmt.Sprintf("expected next token to be %v, got %v (%s)", t, p.peekToken.Type, p.peekToken.Literal)
	p.errors = append(p.errors, msg)
}

func Parse(input string) (*core.NamespaceSchema, error) {
	idx := strings.Index(input, "class ")
	if idx != -1 {
		input = input[idx:]
	}
	l := NewLexer(input)
	p := NewParser(l)
	return p.ParseNamespace()
}

func (p *Parser) ParseNamespace() (*core.NamespaceSchema, error) {
	ns := &core.NamespaceSchema{
		Relations: make(map[string]core.RelationDefinition),
	}

	for p.curToken.Type == TokenIdent && p.curToken.Literal == "import" {
		p.nextToken()
	}

	if p.curToken.Type != TokenClass {
		return nil, fmt.Errorf("expected 'class', got %s", p.curToken.Literal)
	}

	if !p.expectPeek(TokenIdent) {
		return nil, fmt.Errorf("expected class name")
	}
	ns.Name = p.curToken.Literal

	if p.peekToken.Type == TokenImplements {
		p.nextToken()
		if !p.expectPeek(TokenNamespace) {
			return nil, fmt.Errorf("expected 'Namespace' after implements")
		}
	}

	if !p.expectPeek(TokenLBrace) {
		return nil, fmt.Errorf("expected '{'")
	}
	p.nextToken()

	for p.curToken.Type != TokenRBrace && p.curToken.Type != TokenEOF {
		if p.curToken.Type == TokenRelated {
			p.parseRelated(ns)
		} else if p.curToken.Type == TokenPermits {
			err := p.parsePermits(ns)
			if err != nil {
				return nil, err
			}
		} else {
			p.nextToken()
		}
	}

	if len(p.errors) > 0 {
		return nil, fmt.Errorf("parser errors: %s", strings.Join(p.errors, ", "))
	}

	return ns, nil
}

func (p *Parser) parseRelated(ns *core.NamespaceSchema) {
	if !p.expectPeek(TokenColon) {
		return
	}
	if !p.expectPeek(TokenLBrace) {
		return
	}
	p.nextToken()
	for p.curToken.Type != TokenRBrace && p.curToken.Type != TokenEOF {
		p.nextToken()
	}
	p.nextToken() // skip '}'
	if p.curToken.Type == TokenSemicolon || p.curToken.Type == TokenComma {
		p.nextToken()
	}
}

func (p *Parser) parsePermits(ns *core.NamespaceSchema) error {
	if !p.expectPeek(TokenEqual) {
		return fmt.Errorf("expected '=' after permits")
	}
	if !p.expectPeek(TokenLBrace) {
		return fmt.Errorf("expected '{' after permits =")
	}
	p.nextToken()

	for p.curToken.Type != TokenRBrace && p.curToken.Type != TokenEOF {
		if p.curToken.Type == TokenIdent {
			relName := p.curToken.Literal
			
			if p.peekToken.Type == TokenColon {
				p.nextToken()
			}
			p.nextToken() // move to '(' or arg

			if p.curToken.Type == TokenLParen {
				for p.curToken.Type != TokenRParen && p.curToken.Type != TokenEOF {
					p.nextToken()
				}
				p.nextToken() // skip ')'
			} else if p.curToken.Type == TokenCtx || p.curToken.Type == TokenIdent {
				p.nextToken() // skip 'ctx'
			}

			if p.curToken.Type == TokenColon {
				p.nextToken()
				p.nextToken()
			}

			if p.curToken.Type != TokenArrow {
				return fmt.Errorf("expected '=>' for %s, got %s", relName, p.curToken.Literal)
			}
			p.nextToken() // skip '=>'

			rule, err := p.parseExpr()
			if err != nil {
				return err
			}

			ns.Relations[relName] = core.RelationDefinition{
				Rewrite: rule,
			}

			if p.curToken.Type == TokenComma || p.curToken.Type == TokenSemicolon {
				p.nextToken()
			}
		} else {
			p.nextToken()
		}
	}
	p.nextToken() // skip '}'
	if p.curToken.Type == TokenSemicolon {
		p.nextToken()
	}
	return nil
}

func (p *Parser) parseExpr() (*core.RewriteRule, error) {
	return p.parseOrExpr()
}

func (p *Parser) parseOrExpr() (*core.RewriteRule, error) {
	left, err := p.parseAndExpr()
	if err != nil {
		return nil, err
	}

	for p.curToken.Type == TokenOr {
		p.nextToken() // skip ||
		right, err := p.parseAndExpr()
		if err != nil {
			return nil, err
		}
		
		if left.Type == "union" {
			left.Children = append(left.Children, right)
		} else {
			left = &core.RewriteRule{
				Type:     "union",
				Children: []*core.RewriteRule{left, right},
			}
		}
	}
	return left, nil
}

func (p *Parser) parseAndExpr() (*core.RewriteRule, error) {
	left, err := p.parsePrimaryExpr()
	if err != nil {
		return nil, err
	}

	for p.curToken.Type == TokenAnd {
		p.nextToken() // skip &&
		right, err := p.parsePrimaryExpr()
		if err != nil {
			return nil, err
		}
		
		if left.Type == "intersection" {
			left.Children = append(left.Children, right)
		} else {
			left = &core.RewriteRule{
				Type:     "intersection",
				Children: []*core.RewriteRule{left, right},
			}
		}
	}
	return left, nil
}

func (p *Parser) parsePrimaryExpr() (*core.RewriteRule, error) {
	if p.curToken.Type == TokenLParen {
		p.nextToken()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if p.curToken.Type != TokenRParen {
			return nil, fmt.Errorf("expected ')'")
		}
		p.nextToken()
		return expr, nil
	}

	if p.curToken.Type == TokenThis {
		if p.peekToken.Type != TokenDot {
			return nil, fmt.Errorf("expected '.' after this")
		}
		p.nextToken()
		if p.peekToken.Type != TokenRelated {
			return nil, fmt.Errorf("expected 'related' after this.")
		}
		p.nextToken()
		if p.peekToken.Type != TokenDot {
			return nil, fmt.Errorf("expected '.' after this.related")
		}
		p.nextToken()
		// it could be a keyword used as a relation name (e.g. 'owner' might be ident, 'view' might be ident)
		p.nextToken()
		
		relName := p.curToken.Literal

		if p.peekToken.Type == TokenDot {
			p.nextToken()
			p.nextToken() // now on the method name
			methodName := p.curToken.Literal
			
			if methodName == "includes" {
				for p.curToken.Type != TokenRParen && p.curToken.Type != TokenEOF {
					p.nextToken()
				}
				p.nextToken() // skip ')'
				return &core.RewriteRule{
					Type:     "computed_userset",
					Relation: relName,
				}, nil
			} else if methodName == "traverse" {
				if !p.expectPeek(TokenLParen) {
					return nil, fmt.Errorf("expected '(' after traverse")
				}
				p.nextToken()
				
				var paramName string
				if p.curToken.Type == TokenIdent {
					paramName = p.curToken.Literal
					p.nextToken()
				} else if p.curToken.Type == TokenLParen {
					p.nextToken()
					paramName = p.curToken.Literal
					p.nextToken() // skip ident
					if p.curToken.Type == TokenColon {
						p.nextToken()
						p.nextToken() // skip type
					}
					if p.curToken.Type != TokenRParen {
						return nil, fmt.Errorf("expected ')' in traverse param")
					}
					p.nextToken()
				}
				_ = paramName

				if p.curToken.Type != TokenArrow {
					return nil, fmt.Errorf("expected '=>' in traverse")
				}
				p.nextToken()

				if p.curToken.Type == TokenIdent { // p
					p.nextToken()
				}
				if p.curToken.Type != TokenDot {
					return nil, fmt.Errorf("expected '.' in traverse body")
				}
				p.nextToken()
				if p.curToken.Type != TokenRelated {
					return nil, fmt.Errorf("expected 'related' in traverse body")
				}
				p.nextToken()
				if p.curToken.Type != TokenDot {
					return nil, fmt.Errorf("expected '.' after related in traverse body")
				}
				p.nextToken()
				computedRel := p.curToken.Literal
				p.nextToken()

				if p.curToken.Type != TokenRParen {
					return nil, fmt.Errorf("expected ')' after traverse body")
				}
				p.nextToken()

				return &core.RewriteRule{
					Type:             "tuple_to_userset",
					TuplesetRelation: relName,
					ComputedRelation: computedRel,
				}, nil
			}
		}

		p.nextToken() // advance past the relName
		return &core.RewriteRule{
			Type:     "computed_userset",
			Relation: relName,
		}, nil
	}

	return nil, fmt.Errorf("unexpected token in expression: %s", p.curToken.Literal)
}
