// Package orderedjson reads and writes JSON while keeping the order of object
// keys, which encoding/json cannot do. justz needs it to rewrite command
// registries in a canonical order that is not alphabetical.
//
// It uses only the standard library. Numbers keep their original text, and
// output uses two-space indentation like jq does.
package orderedjson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Kind is the type of a JSON value.
type Kind int

const (
	Null Kind = iota
	Bool
	Number
	String
	Array
	Object
)

// Value is a JSON value. Which fields are used depends on Kind.
type Value struct {
	Kind Kind
	Bool bool      // Bool
	Str  string    // String: the decoded text. Number: the original text.
	Arr  []*Value  // Array
	Obj  []*Member // Object, in document order
}

// Member is one key and value of an object.
type Member struct {
	Key string
	Val *Value
}

// Get returns the value for key in an object, or nil.
func (v *Value) Get(key string) *Value {
	if v == nil || v.Kind != Object {
		return nil
	}
	for _, m := range v.Obj {
		if m.Key == key {
			return m.Val
		}
	}
	return nil
}

// Parse reads exactly one JSON value from data. Duplicate object keys are an
// error, because there is no single right answer for which one wins.
func Parse(data []byte) (*Value, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := parseValue(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("unexpected data after the top-level JSON value")
	}
	return v, nil
}

func parseValue(dec *json.Decoder) (*Value, error) {
	tok, err := dec.Token()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("unexpected end of JSON input")
		}
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			return parseObject(dec)
		case '[':
			return parseArray(dec)
		}
		return nil, fmt.Errorf("unexpected %q", rune(t))
	case string:
		return &Value{Kind: String, Str: t}, nil
	case json.Number:
		return &Value{Kind: Number, Str: t.String()}, nil
	case bool:
		return &Value{Kind: Bool, Bool: t}, nil
	case nil:
		return &Value{Kind: Null}, nil
	}
	return nil, fmt.Errorf("unexpected token %v", tok)
}

func parseObject(dec *json.Decoder) (*Value, error) {
	v := &Value{Kind: Object, Obj: []*Member{}}
	seen := map[string]bool{}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("expected an object key, got %v", keyTok)
		}
		if seen[key] {
			return nil, fmt.Errorf("duplicate object key %q", key)
		}
		seen[key] = true
		val, err := parseValue(dec)
		if err != nil {
			return nil, err
		}
		v.Obj = append(v.Obj, &Member{Key: key, Val: val})
	}
	if _, err := dec.Token(); err != nil { // the closing brace
		return nil, err
	}
	return v, nil
}

func parseArray(dec *json.Decoder) (*Value, error) {
	v := &Value{Kind: Array, Arr: []*Value{}}
	for dec.More() {
		el, err := parseValue(dec)
		if err != nil {
			return nil, err
		}
		v.Arr = append(v.Arr, el)
	}
	if _, err := dec.Token(); err != nil { // the closing bracket
		return nil, err
	}
	return v, nil
}

// Marshal writes v with two-space indentation and a trailing newline.
func Marshal(v *Value) ([]byte, error) {
	var b bytes.Buffer
	if err := write(&b, v, 0); err != nil {
		return nil, err
	}
	b.WriteByte('\n')
	return b.Bytes(), nil
}

func write(b *bytes.Buffer, v *Value, depth int) error {
	switch v.Kind {
	case Null:
		b.WriteString("null")
	case Bool:
		if v.Bool {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case Number:
		b.WriteString(v.Str)
	case String:
		return writeString(b, v.Str)
	case Array:
		if len(v.Arr) == 0 {
			b.WriteString("[]")
			return nil
		}
		b.WriteString("[\n")
		for i, el := range v.Arr {
			b.WriteString(strings.Repeat("  ", depth+1))
			if err := write(b, el, depth+1); err != nil {
				return err
			}
			if i < len(v.Arr)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString(strings.Repeat("  ", depth))
		b.WriteByte(']')
	case Object:
		if len(v.Obj) == 0 {
			b.WriteString("{}")
			return nil
		}
		b.WriteString("{\n")
		for i, m := range v.Obj {
			b.WriteString(strings.Repeat("  ", depth+1))
			if err := writeString(b, m.Key); err != nil {
				return err
			}
			b.WriteString(": ")
			if err := write(b, m.Val, depth+1); err != nil {
				return err
			}
			if i < len(v.Obj)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString(strings.Repeat("  ", depth))
		b.WriteByte('}')
	}
	return nil
}

// writeString writes a JSON string without escaping HTML characters.
func writeString(b *bytes.Buffer, s string) error {
	var tmp bytes.Buffer
	enc := json.NewEncoder(&tmp)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return err
	}
	b.Write(bytes.TrimRight(tmp.Bytes(), "\n"))
	return nil
}
