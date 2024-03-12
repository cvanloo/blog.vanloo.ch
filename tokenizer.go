package main

import "fmt"

type TokenType int
const (
	TypeFormStart TokenType = iota
	TypeAtom
	TypeText
	TypeFormEnd
)

type Token struct {
	Type TokenType
	Text string
	Pos int
}

func (t Token) String() string {
	switch (t.Type) {
	case TypeFormStart:
		return fmt.Sprintf("FormStart{%d: `%s`}", t.Pos, visibleString(t.Text))
	case TypeAtom:
		return fmt.Sprintf("Atom{%d: `%s`}", t.Pos, visibleString(t.Text))
	case TypeText:
		return fmt.Sprintf("Text{%d: `%s`}", t.Pos, visibleString(t.Text))
	case TypeFormEnd:
		return fmt.Sprintf("FormEnd{%d: `%s`}", t.Pos, visibleString(t.Text))
	}
	return "Invalid{~}"
}

func visibleString(s string) string {
	asciiSpecialLookup := [...]string{
		"<NUL>",
		"<SOH>",
		"<STX>",
		"<ETX>",
		"<EOT>",
		"<ENQ>",
		"<ACK>",
		"\\a",
		"\\b",
		"\\t",
		"\\n",
		"\\v",
		"\\f",
		"\\r",
		"<SO>",
		"<SI>",
		"<DLE>",
		"<DC1>",
		"<DC2>",
		"<DC3>",
		"<DC4>",
		"<NAK>",
		"<SYN>",
		"<ETB>",
		"<CAN>",
		"<EM>",
		"<SUB>",
		"<ESC>",
		"<FS>",
		"<GS>",
		"<RS>",
		"<US>",
	}
	v := ""
	for _, r := range s {
		if r >= 32 && r <= 126 {
			v += string(r)
		} else if r == 127 {
			v += "<DEL>"
		} else if r >= 0 && r <= 31 {
			v += asciiSpecialLookup[r]
		} else /* is unicode (probably) */{
			v += fmt.Sprintf("<%U>", r)
		}
	}
	return v
}

type tokFunc func() tokFunc

type Tokenizer struct {
	bs []rune
	pos int
	tokens []Token
	state tokFunc
}

func NewTokenizer(bs []rune) *Tokenizer {
	return &Tokenizer{
		bs: bs,
	}
}

func (t *Tokenizer) Tokenize() ([]Token, error) {
	l := len(t.bs)
	t.state = t.tokTextOrForm
	for t.pos < l && t.state != nil {
		t.state = t.state()
	}
	return t.tokens, nil
}

func (t *Tokenizer) tokTextOrForm() tokFunc { // initial state
	t.skipWhitespace()
	if t.pos >= len(t.bs) {
		return t.tokEOF
	}
	if t.bs[t.pos] == '(' {
		return t.tokForm
	}
	return t.tokText
}

func (t *Tokenizer) tokText() tokFunc { // parse text
	t.skipWhitespace()
	if t.pos >= len(t.bs) {
		return t.tokEOF
	}
	textEnd := t.pos
	quoted := false
	for textEnd < len(t.bs) && ((t.bs[textEnd] != ')' && t.bs[textEnd] != '(') || quoted) {
		if t.bs[textEnd] == '\\' {
			if textEnd+1 < len(t.bs) && (t.bs[textEnd+1] == '(' || t.bs[textEnd+1] == ')' || t.bs[textEnd+1] == '\\') {
				// @todo: remove `\` ?
				textEnd++
			} else if textEnd+1 < len(t.bs) && t.bs[textEnd+1] == '+' {
				// @todo: remove `\+` ?
				textEnd++
				quoted = !quoted
			} else {
				panic("invalid escape character") // @todo: error handling
			}
		}
		textEnd++
	}
	t.tokens = append(t.tokens, Token{
		Type: TypeText,
		Text: string(t.bs[t.pos:textEnd]),
		Pos: t.pos,
	})
	t.pos = textEnd

	return t.tokNilOrTextOrForm
}

func (t *Tokenizer) tokForm() tokFunc { // parse form start
	t.tokens = append(t.tokens, Token{
		Type: TypeFormStart,
		Text: "(",
		Pos: t.pos,
	})
	t.pos++

	return t.tokNilOrAtom
}

func (t *Tokenizer) tokNilOrAtom() tokFunc {
	t.skipWhitespace()
	if t.pos >= len(t.bs) {
		return t.tokEOF
	}
	r := t.bs[t.pos]
	if r == '(' {
		panic("invalid: cannot start form / expected atom or nil") // @todo: error handling
	}
	if r == ')' {
		return t.tokNil
	}
	return t.tokAtom
}

func (t *Tokenizer) tokNil() tokFunc { // parse form end
	t.tokens = append(t.tokens, Token{
		Type: TypeFormEnd,
		Text: ")",
		Pos: t.pos,
	})
	t.pos++

	return t.tokNilOrTextOrForm
}

func (t *Tokenizer) tokAtom() tokFunc { // parse atom
	atomEnd := t.pos
	for atomEnd < len(t.bs) && isAlphaNum(t.bs[atomEnd]) {
		atomEnd++
	}
	t.tokens = append(t.tokens, Token{
		Type: TypeAtom,
		Text: string(t.bs[t.pos:atomEnd]),
		Pos: t.pos,
	})
	t.pos = atomEnd

	return t.tokNilOrTextOrForm
}

func (t *Tokenizer) tokNilOrTextOrForm() tokFunc {
	t.skipWhitespace()
	if t.pos >= len(t.bs) {
		return t.tokEOF
	}
	r := t.bs[t.pos]
	if r == ')' {
		return t.tokNil
	}
	if r == '(' {
		return t.tokForm
	}
	return t.tokText
}

func (t *Tokenizer) tokEOF() tokFunc {
	t.tokens = append(
		t.tokens,
		Token{
			Type: TypeFormStart,
			Text: "(",
			Pos: t.pos,
		},
		Token{
			Type: TypeAtom,
			Text: "eof",
			Pos: t.pos,
		},
		Token{
			Type: TypeFormEnd,
			Text: ")",
			Pos: t.pos,
		},
	)

	return nil
}

func (t *Tokenizer) skipWhitespace() {
	for t.pos < len(t.bs) && isWhitespace(t.bs[t.pos]) {
		t.pos++
	}
}

func isWhitespace(r rune) bool {
	ws := []rune{' ', '\n', '\r', '\t', '\v', '\f'}
	for _, w := range ws {
		if r == w {
			return true
		}
	}
	return false
}

func isAlpha(r rune) bool {
	return r >= 97 && r <= 122 || r == '-'
}

func isNum(r rune) bool {
	return r >= 48 && r <= 57
}

func isAlphaNum(r rune) bool {
	return isAlpha(r) || isNum(r)
}
