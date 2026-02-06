package texconfig

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type cfgValueKind int // Kind is used to identify the type of the value.

const (
	cfgString cfgValueKind = iota
	cfgNumber
	cfgIdent
)

type cfgValue struct {
	Str  string
	Kind cfgValueKind
	Int  int
}

// String returns the string representation of the value.
func (v cfgValue) String() string {
	switch v.Kind {
	case cfgString:
		return v.Str
	case cfgNumber:
		return strconv.Itoa(v.Int)
	case cfgIdent:
		return v.Str
	default:
		return v.Str
	}
}

type cfgClass struct {
	Name    string
	Base    string
	Props   map[string]cfgValue
	Classes []*cfgClass
}

type cfgFile struct {
	Assignments map[string]cfgValue
	Classes     []*cfgClass
}

// ParseTexConvertConfig parses the original TexConvert.cfg format from text.
// It supports class inheritance and returns a flattened TexConvertConfig.
func ParseTexConvertConfig(text string) (TexConvertConfig, error) {
	cfg, err := parseCfg(text)
	if err != nil {
		return TexConvertConfig{}, err
	}

	return buildTexConvertConfig(cfg)
}

// LoadTexConvertConfig reads and parses TexConvert.cfg from disk.
func LoadTexConvertConfig(path string) (TexConvertConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return TexConvertConfig{}, err
	}

	return ParseTexConvertConfig(string(data))
}

// buildTexConvertConfig builds a TexConvertConfig from a cfgFile.
func buildTexConvertConfig(cfg *cfgFile) (TexConvertConfig, error) {
	out := TexConvertConfig{}
	if v, ok := cfg.Assignments["convertVersion"]; ok && v.Kind == cfgNumber {
		out.ConvertVersion = v.Int
	}

	var hintsClass *cfgClass
	for _, c := range cfg.Classes {
		if strings.EqualFold(c.Name, "TextureHints") {
			hintsClass = c
			break
		}
	}
	if hintsClass == nil {
		return TexConvertConfig{}, fmt.Errorf("TextureHints class not found")
	}

	// Create a map of class names to classes for efficient lookup.
	classMap := make(map[string]*cfgClass, len(hintsClass.Classes))
	for _, c := range hintsClass.Classes {
		classMap[c.Name] = c
	}

	// Create a memoization map to avoid infinite recursion in class inheritance.
	memo := make(map[string]map[string]cfgValue)

	// Resolve is a recursive function that resolves the properties of a class.
	var resolve func(*cfgClass, map[string]bool) (map[string]cfgValue, error)
	resolve = func(c *cfgClass, stack map[string]bool) (map[string]cfgValue, error) {
		if props, ok := memo[c.Name]; ok {
			return props, nil
		}
		if stack[c.Name] {
			return nil, fmt.Errorf("inheritance cycle at %q", c.Name)
		}

		stack[c.Name] = true
		props := make(map[string]cfgValue)
		if c.Base != "" {
			base, ok := classMap[c.Base]
			if !ok {
				return nil, fmt.Errorf("unknown base class %q for %q", c.Base, c.Name)
			}

			baseProps, err := resolve(base, stack)
			if err != nil {
				return nil, err
			}

			for k, v := range baseProps {
				props[k] = v
			}
		}

		for k, v := range c.Props {
			props[k] = v
		}

		stack[c.Name] = false
		memo[c.Name] = props

		return props, nil
	}

	// Create a slice of TextureHint to store the resolved hints.
	out.Hints = make([]TextureHint, 0, len(hintsClass.Classes))
	for _, cls := range hintsClass.Classes {
		props, err := resolve(cls, map[string]bool{})
		if err != nil {
			return TexConvertConfig{}, err
		}

		hint, err := hintFromProps(cls, props)
		if err != nil {
			return TexConvertConfig{}, err
		}

		out.Hints = append(out.Hints, hint)
	}

	return out, nil
}

// hintFromProps creates a TextureHint from the properties of a class.
func hintFromProps(cls *cfgClass, props map[string]cfgValue) (TextureHint, error) {
	h := TextureHint{
		ClassName: cls.Name,
		Extends:   cls.Base,
	}

	if v, ok := props["name"]; ok {
		h.Pattern = v.String()
	}

	if v, ok := props["format"]; ok {
		format, ok := ParseTexFormat(v.String())
		if !ok {
			return TextureHint{}, fmt.Errorf("unknown format %q in %s", v.String(), cls.Name)
		}
		h.Format = format
	}

	if v, ok := props["mipmapFilter"]; ok {
		filter, ok := ParseMipmapFilter(v.String())
		if !ok {
			return TextureHint{}, fmt.Errorf("unknown mipmapFilter %q in %s", v.String(), cls.Name)
		}
		h.MipmapFilter = filter
	}

	if v, ok := props["errorMetrics"]; ok {
		metrics, ok := ParseErrorMetrics(v.String())
		if !ok {
			return TextureHint{}, fmt.Errorf("unknown errorMetrics %q in %s", v.String(), cls.Name)
		}
		h.ErrorMetrics = metrics
	}

	if v, ok := props["enableDXT"]; ok {
		b, err := cfgBool(v)
		if err != nil {
			return TextureHint{}, fmt.Errorf("enableDXT in %s: %w", cls.Name, err)
		}
		h.EnableDXT = &b
	}

	if v, ok := props["dynRange"]; ok {
		b, err := cfgBool(v)
		if err != nil {
			return TextureHint{}, fmt.Errorf("dynRange in %s: %w", cls.Name, err)
		}
		h.DynRange = &b
	}

	if v, ok := props["autoreduce"]; ok {
		b, err := cfgBool(v)
		if err != nil {
			return TextureHint{}, fmt.Errorf("autoreduce in %s: %w", cls.Name, err)
		}
		h.AutoReduce = &b
	}

	if v, ok := props["virtualSwizzle"]; ok {
		b, err := cfgBool(v)
		if err != nil {
			return TextureHint{}, fmt.Errorf("virtualSwizzle in %s: %w", cls.Name, err)
		}
		h.VirtualSwz = &b
	}

	if v, ok := props["dithering"]; ok {
		b, err := cfgBool(v)
		if err != nil {
			return TextureHint{}, fmt.Errorf("dithering in %s: %w", cls.Name, err)
		}
		h.Dithering = &b
	}

	if v, ok := props["limitSize"]; ok {
		n, err := cfgInt(v)
		if err != nil {
			return TextureHint{}, fmt.Errorf("limitSize in %s: %w", cls.Name, err)
		}
		h.LimitSize = n
	}

	if v, ok := props["channelSwizzleR"]; ok {
		expr, err := ParseSwizzleExpr(v.String())
		if err != nil {
			return TextureHint{}, fmt.Errorf("channelSwizzleR in %s: %w", cls.Name, err)
		}
		h.Swizzle.R = expr
	}

	if v, ok := props["channelSwizzleG"]; ok {
		expr, err := ParseSwizzleExpr(v.String())
		if err != nil {
			return TextureHint{}, fmt.Errorf("channelSwizzleG in %s: %w", cls.Name, err)
		}
		h.Swizzle.G = expr
	}

	if v, ok := props["channelSwizzleB"]; ok {
		expr, err := ParseSwizzleExpr(v.String())
		if err != nil {
			return TextureHint{}, fmt.Errorf("channelSwizzleB in %s: %w", cls.Name, err)
		}
		h.Swizzle.B = expr
	}

	if v, ok := props["channelSwizzleA"]; ok {
		expr, err := ParseSwizzleExpr(v.String())
		if err != nil {
			return TextureHint{}, fmt.Errorf("channelSwizzleA in %s: %w", cls.Name, err)
		}
		h.Swizzle.A = expr
	}

	return h, nil
}

// cfgBool converts a cfgValue to a boolean.
func cfgBool(v cfgValue) (bool, error) {
	switch v.Kind {
	case cfgNumber:
		return v.Int != 0, nil

	case cfgIdent:
		s := strings.ToLower(strings.TrimSpace(v.Str))
		switch s {
		case "true", "1":
			return true, nil
		case "false", "0":
			return false, nil
		default:
			return false, fmt.Errorf("invalid bool %q", v.Str)
		}

	case cfgString:
		s := strings.ToLower(strings.TrimSpace(v.Str))
		switch s {
		case "true", "1":
			return true, nil
		case "false", "0":
			return false, nil
		default:
			return false, fmt.Errorf("invalid bool %q", v.Str)
		}

	default:
		return false, fmt.Errorf("invalid bool value")
	}
}

// cfgInt converts a cfgValue to an integer.
func cfgInt(v cfgValue) (int, error) {
	switch v.Kind {
	case cfgNumber:
		return v.Int, nil
	case cfgIdent:
		return strconv.Atoi(v.Str)
	case cfgString:
		return strconv.Atoi(v.Str)
	default:
		return 0, fmt.Errorf("invalid int value")
	}
}

// parseCfg parses the original TexConvert.cfg format from text.
func parseCfg(text string) (*cfgFile, error) {
	lex := newLexer(text)
	cfg := &cfgFile{Assignments: make(map[string]cfgValue)}
	for {
		tok, err := lex.peek()
		if err != nil {
			return nil, err
		}

		if tok.typ == tokEOF {
			break
		}

		if tok.typ == tokIdent && strings.EqualFold(tok.val, "class") {
			cls, err := parseClass(lex)
			if err != nil {
				return nil, err
			}

			cfg.Classes = append(cfg.Classes, cls)
			continue
		}

		if tok.typ == tokIdent {
			key, val, err := parseAssignment(lex)
			if err != nil {
				return nil, err
			}

			cfg.Assignments[key] = val
			continue
		}

		return nil, fmt.Errorf("unexpected token %q", tok.val)
	}

	return cfg, nil
}

// parseClass parses a class from the lexer.
func parseClass(lex *lexer) (*cfgClass, error) {
	if _, err := lex.expectIdent("class"); err != nil {
		return nil, err
	}

	nameTok, err := lex.expect(tokIdent)
	if err != nil {
		return nil, err
	}

	cls := &cfgClass{Name: nameTok.val, Props: make(map[string]cfgValue)}

	tok, err := lex.peek()
	if err != nil {
		return nil, err
	}

	// Parse the base class of the class.
	if tok.typ == tokSymbol && tok.val == ":" {
		_, _ = lex.next()
		baseTok, err := lex.expect(tokIdent)
		if err != nil {
			return nil, err
		}
		cls.Base = baseTok.val
	}

	// Expect the opening brace of the class properties.
	if err := lex.expectSymbol("{"); err != nil {
		return nil, err
	}

	// Parse the properties of the class.
	for {
		tok, err = lex.peek()
		if err != nil {
			return nil, err
		}

		if tok.typ == tokSymbol && tok.val == "}" {
			_, _ = lex.next()
			break
		}

		if tok.typ == tokIdent && strings.EqualFold(tok.val, "class") {
			child, err := parseClass(lex)
			if err != nil {
				return nil, err
			}

			cls.Classes = append(cls.Classes, child)
			continue
		}

		if tok.typ == tokIdent {
			key, val, err := parseAssignment(lex)
			if err != nil {
				return nil, err
			}

			cls.Props[key] = val
			continue
		}

		return nil, fmt.Errorf("unexpected token in class %s: %q", cls.Name, tok.val)
	}

	// optional semicolon after class
	if tok, err := lex.peek(); err == nil && tok.typ == tokSymbol && tok.val == ";" {
		_, _ = lex.next()
	}

	return cls, nil
}

// parseAssignment parses an assignment from the lexer and returns the key and value.
func parseAssignment(lex *lexer) (string, cfgValue, error) {
	keyTok, err := lex.expect(tokIdent)
	if err != nil {
		return "", cfgValue{}, err
	}

	if err := lex.expectSymbol("="); err != nil {
		return "", cfgValue{}, err
	}

	valTok, err := lex.next()
	if err != nil {
		return "", cfgValue{}, err
	}

	// Parse the value of the assignment.
	var val cfgValue
	switch valTok.typ {
	case tokString:
		val = cfgValue{Kind: cfgString, Str: valTok.val}

	case tokNumber:
		n, err := strconv.Atoi(valTok.val)
		if err != nil {
			return "", cfgValue{}, fmt.Errorf("invalid number %q", valTok.val)
		}
		val = cfgValue{Kind: cfgNumber, Int: n}

	case tokIdent:
		val = cfgValue{Kind: cfgIdent, Str: valTok.val}

	default:
		return "", cfgValue{}, fmt.Errorf("unexpected value token %q", valTok.val)
	}

	if err := lex.expectSymbol(";"); err != nil {
		return "", cfgValue{}, err
	}

	return keyTok.val, val, nil
}
