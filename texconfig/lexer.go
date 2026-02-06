package texconfig

import (
	"fmt"
	"strings"
)

type tokenType int // Type is used to identify the type of the token.

const (
	tokEOF tokenType = iota
	tokIdent
	tokNumber
	tokString
	tokSymbol
)

type token struct {
	val string    // The value of the token.
	typ tokenType // The type of the token.
	pos int       // The position of the token in the source string.
}

type lexer struct {
	peekTok *token // The next token to peek.
	src     []rune // The source string to lex.
	pos     int    // The current position in the source string.
}

// newLexer creates a new lexer for the given string.
func newLexer(s string) *lexer {
	return &lexer{src: []rune(s)}
}

// peek returns the next token without consuming it.
func (l *lexer) peek() (token, error) {
	if l.peekTok != nil {
		return *l.peekTok, nil
	}

	tok, err := l.next()
	if err != nil {
		return token{}, err
	}

	l.peekTok = &tok
	return tok, nil
}

// next returns the next token and consumes it.
func (l *lexer) next() (token, error) {
	if l.peekTok != nil {
		tok := *l.peekTok
		l.peekTok = nil
		return tok, nil
	}

	// Skip space and comments.
	if err := l.skipSpaceAndComments(); err != nil {
		return token{}, err
	}
	if l.pos >= len(l.src) {
		return token{typ: tokEOF}, nil
	}

	// Parse the next token.
	r := l.src[l.pos]
	if isIdentStart(r) {
		start := l.pos
		l.pos++
		for l.pos < len(l.src) && isIdentPart(l.src[l.pos]) {
			l.pos++
		}

		return token{typ: tokIdent, val: string(l.src[start:l.pos]), pos: start}, nil
	}

	// Parse the next token as a number.
	if isDigit(r) {
		start := l.pos
		l.pos++
		for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
			l.pos++
		}

		return token{typ: tokNumber, val: string(l.src[start:l.pos]), pos: start}, nil
	}

	// Parse the next token as a string.
	if r == '"' {
		start := l.pos
		l.pos++
		var sb strings.Builder
		for l.pos < len(l.src) {
			c := l.src[l.pos]
			if c == '\\' {
				if l.pos+1 >= len(l.src) {
					return token{}, fmt.Errorf("unterminated escape in string")
				}

				next := l.src[l.pos+1]
				sb.WriteRune(next)
				l.pos += 2
				continue
			}

			if c == '"' {
				l.pos++
				return token{typ: tokString, val: sb.String(), pos: start}, nil
			}

			// Parse the next token as a symbol.
			sb.WriteRune(c)
			l.pos++
		}

		return token{}, fmt.Errorf("unterminated string")
	}

	// symbols
	sym := string(r)
	l.pos++

	// Return the token.
	switch sym {
	case "{", "}", ":", "=", ";":
		return token{typ: tokSymbol, val: sym}, nil
	default:
		return token{}, fmt.Errorf("unexpected character %q", sym)
	}
}

// expect returns the next token of the given type and consumes it.
func (l *lexer) expect(tt tokenType) (token, error) {
	tok, err := l.next()
	if err != nil {
		return token{}, err
	}

	if tok.typ != tt {
		return token{}, fmt.Errorf("expected %v, got %v (%q)", tt, tok.typ, tok.val)
	}

	return tok, nil
}

// expectIdent returns the next token of type tokIdent and consumes it.
func (l *lexer) expectIdent(expected string) (token, error) {
	tok, err := l.expect(tokIdent)
	if err != nil {
		return token{}, err
	}

	if !strings.EqualFold(tok.val, expected) {
		return token{}, fmt.Errorf("expected %q, got %q", expected, tok.val)
	}

	return tok, nil
}

// expectSymbol returns the next token of type tokSymbol and consumes it.
func (l *lexer) expectSymbol(sym string) error {
	tok, err := l.expect(tokSymbol)
	if err != nil {
		return err
	}

	if tok.val != sym {
		return fmt.Errorf("expected symbol %q, got %q", sym, tok.val)
	}

	return nil
}

// skipSpaceAndComments skips space and comments.
func (l *lexer) skipSpaceAndComments() error {
	for l.pos < len(l.src) {
		r := l.src[l.pos]
		if isSpace(r) {
			l.pos++
			continue
		}

		// Parse the next token as a line comment.
		if r == '/' && l.pos+1 < len(l.src) {
			next := l.src[l.pos+1]
			if next == '/' {
				l.pos += 2
				for l.pos < len(l.src) && l.src[l.pos] != '\n' {
					l.pos++
				}
				continue
			}

			// Parse the next token as a block comment.
			if next == '*' {
				l.pos += 2
				for l.pos+1 < len(l.src) {
					if l.src[l.pos] == '*' && l.src[l.pos+1] == '/' {
						l.pos += 2
						break
					}
					l.pos++
				}

				// Check if the block comment is unterminated.
				if l.pos >= len(l.src) {
					return fmt.Errorf("unterminated block comment")
				}

				continue
			}
		}

		// Break out of the loop if we encounter a non-space or non-comment character.
		break
	}

	return nil
}

// isSpace checks if the given rune is a space character.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

// isDigit checks if the given rune is a digit character.
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// isIdentStart checks if the given rune is the start of an identifier.
func isIdentStart(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_'
}

// isIdentPart checks if the given rune is a part of an identifier.
func isIdentPart(r rune) bool {
	return isIdentStart(r) || isDigit(r)
}
