// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jsonutils

import (
	"errors"
	"strconv"
	"strings"

	"github.com/mailru/easyjson/jlexer"
	"github.com/mailru/easyjson/jwriter"
)

// defaultMaxNestingDepth is the default maximum number of nested JSON containers
// ('{' or '[') that the ordered-JSON marshaler and unmarshaler will process before
// returning an error.
//
// It mirrors the limit enforced by the standard library's [encoding/json] decoder,
// guarding against stack-overflow crashes on deeply nested (possibly adversarial) input.
const defaultMaxNestingDepth = 10000

// ErrMaxNestingDepth is returned when a JSON document or in-memory structure nests
// deeper than the maximum supported depth. It guards against stack-overflow crashes
// on adversarially deep input.
var ErrMaxNestingDepth = errors.New("maximum JSON nesting depth exceeded")

// JSONMapSlice represents a JSON object, with the order of keys maintained.
type JSONMapSlice []JSONMapItem

// MarshalJSON renders a [JSONMapSlice] as JSON bytes, preserving the order of keys.
func (s JSONMapSlice) MarshalJSON() ([]byte, error) {
	w := &jwriter.Writer{Flags: jwriter.NilMapAsEmpty | jwriter.NilSliceAsEmpty}
	s.MarshalEasyJSON(w)

	return w.BuildBytes()
}

// MarshalEasyJSON renders a [JSONMapSlice] as JSON bytes, using easyJSON
func (s JSONMapSlice) MarshalEasyJSON(w *jwriter.Writer) {
	s.marshalEasyJSON(w, defaultMaxNestingDepth)
}

// UnmarshalJSON builds a [JSONMapSlice] from JSON bytes, preserving the order of keys.
//
// Inner objects are unmarshaled as [JSONMapSlice] slices and not map[string]any.
func (s *JSONMapSlice) UnmarshalJSON(data []byte) error {
	l := jlexer.Lexer{Data: data}
	s.UnmarshalEasyJSON(&l)

	return l.Error()
}

// UnmarshalEasyJSON builds a [JSONMapSlice] from JSON bytes, using easyJSON
func (s *JSONMapSlice) UnmarshalEasyJSON(in *jlexer.Lexer) {
	s.unmarshalEasyJSON(in, defaultMaxNestingDepth)
}

// marshalEasyJSON writes the object to w, decreasing budget for every nested container
// to guard against stack overflow on deeply nested structures.
func (s JSONMapSlice) marshalEasyJSON(w *jwriter.Writer, budget int) {
	if s == nil {
		w.RawString("null")

		return
	}

	if budget <= 0 {
		w.Error = ErrMaxNestingDepth

		return
	}

	w.RawByte('{')

	if len(s) == 0 {
		w.RawByte('}')

		return
	}

	s[0].marshalEasyJSON(w, budget)

	for i := 1; i < len(s); i++ {
		w.RawByte(',')
		s[i].marshalEasyJSON(w, budget)
	}

	w.RawByte('}')
}

// unmarshalEasyJSON parses an object, decreasing budget for every nested container
// so that adversarially deep documents return an error instead of overflowing the stack.
func (s *JSONMapSlice) unmarshalEasyJSON(in *jlexer.Lexer, budget int) {
	if in.IsNull() {
		in.Skip()

		return
	}

	if budget <= 0 {
		in.AddError(ErrMaxNestingDepth)

		return
	}

	result := make(JSONMapSlice, 0)
	in.Delim('{')
	for !in.IsDelim('}') {
		var mi JSONMapItem
		mi.unmarshalEasyJSON(in, budget)
		result = append(result, mi)
	}
	in.Delim('}')

	*s = result
}

// JSONMapItem represents the value of a key in a JSON object held by [JSONMapSlice].
//
// Notice that JSONMapItem should not be marshaled to or unmarshaled from JSON directly,
// use this type as part of a [JSONMapSlice] when dealing with JSON bytes.
type JSONMapItem struct {
	Key   string
	Value any
}

// MarshalEasyJSON renders a [JSONMapItem] as JSON bytes, using easyJSON
func (s JSONMapItem) MarshalEasyJSON(w *jwriter.Writer) {
	s.marshalEasyJSON(w, defaultMaxNestingDepth)
}

// UnmarshalEasyJSON builds a [JSONMapItem] from JSON bytes, using easyJSON
func (s *JSONMapItem) UnmarshalEasyJSON(in *jlexer.Lexer) {
	s.unmarshalEasyJSON(in, defaultMaxNestingDepth)
}

func (s JSONMapItem) marshalEasyJSON(w *jwriter.Writer, budget int) {
	w.String(s.Key)
	w.RawByte(':')

	// Recurse internally for nested ordered maps so the depth guard survives, instead
	// of dispatching to MarshalEasyJSON which would reset it and re-enable overflow.
	if nested, ok := s.Value.(JSONMapSlice); ok {
		nested.marshalEasyJSON(w, budget-1)

		return
	}

	w.Raw(WriteJSON(s.Value))
}

func (s *JSONMapItem) unmarshalEasyJSON(in *jlexer.Lexer, budget int) {
	key := in.UnsafeString()
	in.WantColon()
	value := s.asInterface(in, budget-1)
	in.WantComma()

	s.Key = key
	s.Value = value
}

// asInterface is very much like [jlexer.Lexer.Interface], but unmarshals an object
// into a [JSONMapSlice], not a map[string]any.
//
// We have to force parsing errors somehow, since [jlexer.Lexer] doesn't let us
// set a parsing error directly.
func (s *JSONMapItem) asInterface(in *jlexer.Lexer, budget int) any {
	tokenKind := in.CurrentToken()

	if !in.Ok() {
		return nil
	}

	switch tokenKind {
	case jlexer.TokenString:
		return in.String()

	case jlexer.TokenNumber:
		// determine if we may use an integer type
		n := in.JsonNumber().String()
		if strings.ContainsRune(n, '.') {
			f, _ := strconv.ParseFloat(n, 64)
			return f
		}

		i, _ := strconv.ParseInt(n, 10, 64)
		return i

	case jlexer.TokenBool:
		return in.Bool()

	case jlexer.TokenNull:
		in.Null()
		return nil

	case jlexer.TokenDelim:
		if in.IsDelim('{') {
			ret := make(JSONMapSlice, 0)
			ret.unmarshalEasyJSON(in, budget)

			if in.Ok() {
				return ret
			}

			// lexer is in an error state: will exhaust
			return nil
		}

		if in.IsDelim('[') {
			if budget <= 0 {
				in.AddError(ErrMaxNestingDepth)

				return nil
			}

			in.Delim('[') // consume

			ret := []interface{}{}
			for !in.IsDelim(']') {
				ret = append(ret, s.asInterface(in, budget-1))
				in.WantComma()
			}
			in.Delim(']')

			if in.Ok() {
				return ret
			}

			// lexer is in an error state: will exhaust
			return nil
		}

		if in.Ok() {
			in.Delim('{') // force error
		}

		return nil

	case jlexer.TokenUndef:
		fallthrough
	default:
		if in.Ok() {
			in.Delim('{') // force error
		}

		return nil
	}
}
