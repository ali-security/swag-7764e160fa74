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
	"encoding/json"
	"strings"
	"testing"

	"github.com/mailru/easyjson/jlexer"
	"github.com/mailru/easyjson/jwriter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONMapSlice(t *testing.T) {
	t.Run("should unmarshal and marshal MapSlice", func(t *testing.T) {
		t.Run("with object", func(t *testing.T) {
			const sd = `{"1":"the int key value","name":"a string value","y":"some value"}`
			var data JSONMapSlice
			require.NoError(t, json.Unmarshal([]byte(sd), &data))

			jazon, err := json.Marshal(data)
			require.NoError(t, err)

			assert.JSONEq(t, sd, string(jazon))
		})

		t.Run("with nested object", func(t *testing.T) {
			const sd = `
		{
		  "1":"the int key value",
		  "name":"a string value",
		  "y":{
		    "a":"some value",
		    "b":[
		     {"x":1,"y":2},
		     {"z":4,"w":5}
		    ]
		  }
		}
		`
			var data JSONMapSlice
			require.NoError(t, json.Unmarshal([]byte(sd), &data))

			jazon, err := json.Marshal(data)
			require.NoError(t, err)

			assert.JSONEq(t, sd, string(jazon))
		})

		t.Run("with nested array", func(t *testing.T) {
			const sd = `
	[
	  {
			"1":"the int key value",
	    "name":"a string value"
		},
		{
	    "y":{
	      "a":"some value",
	      "b": [
	         {"x":1,"y":2},
	         {"z":4,"w":5}
	      ],
			  "c": false,
			  "d": null
	    },
	    "z": true
	  },
		{
			"v": [true, "string", 10.35]
		}
	]
	`
			var data []JSONMapSlice
			require.NoError(t, json.Unmarshal([]byte(sd), &data))

			jazon, err := json.Marshal(data)
			require.NoError(t, err)

			assert.JSONEq(t, sd, string(jazon))
		})

		t.Run("with empty array", func(t *testing.T) {
			const sd = `{"a":[]}`
			var data JSONMapSlice
			require.NoError(t, json.Unmarshal([]byte(sd), &data))

			jazon, err := json.Marshal(data)
			require.NoError(t, err)

			assert.JSONEq(t, sd, string(jazon))
		})

		t.Run("with empty object", func(t *testing.T) {
			const sd = `{}`
			var data JSONMapSlice
			require.NoError(t, json.Unmarshal([]byte(sd), &data))

			jazon, err := json.Marshal(data)
			require.NoError(t, err)

			assert.JSONEq(t, sd, string(jazon))
		})

		t.Run("with null value", func(t *testing.T) {
			const sd = `null`
			var data JSONMapSlice
			require.NoError(t, json.Unmarshal([]byte(sd), &data))
			assert.Nil(t, data)

			jazon, err := json.Marshal(data)
			require.NoError(t, err)

			assert.JSONEq(t, sd, string(jazon))
		})
	})

	t.Run("should keep the order of keys", func(t *testing.T) {
		const sd = `{"a":1,"b":2,"c":3,"d":4}`
		var data JSONMapSlice
		require.NoError(t, json.Unmarshal([]byte(sd), &data))
		jazon, err := json.Marshal(data)
		require.NoError(t, err)

		require.Equal(t, sd, string(jazon)) // specifically check the same order, not JSONEq()

		t.Run("should Read/Write JSON using easyJSON", func(t *testing.T) {
			var obj interface{}
			require.NoError(t, FromDynamicJSON(data, &obj))

			asMap, ok := obj.(map[string]interface{})
			require.True(t, ok)
			assert.Len(t, asMap, 4) // 3 fields in struct

			var target JSONMapSlice
			require.NoError(t, FromDynamicJSON(obj, &target))
			// the order of keys may be altered, since the intermediary representation is a map[string]interface{}
		})
	})

	t.Run("UnmarshalEasyJSON with error cases", func(t *testing.T) {
		// test directly this endpoint, as the json standard library
		// performs a preventive early check for well-formed JSON.
		t.Run("on invalid token (1)", func(t *testing.T) {
			const sd = `{"a":|,"b":2,"c":3,"d":4}`
			var data JSONMapSlice
			require.Error(t, json.Unmarshal([]byte(sd), &data))
		})
		t.Run("on invalid token (2)", func(t *testing.T) {
			const sd = `{"a":{ai+b,"b":2,"c":3,"d":4}`
			var data JSONMapSlice
			require.Error(t, json.Unmarshal([]byte(sd), &data))
		})
		t.Run("on invalid token (3)", func(t *testing.T) {
			const sd = `{"a":[ai+b,"b":2,"c":3,"d":4}`
			data := make(JSONMapSlice, 0)
			l := jlexer.Lexer{Data: []byte(sd)}
			data.UnmarshalEasyJSON(&l)
			require.Error(t, l.Error())
		})
		t.Run("on invalid delimiter (1)", func(t *testing.T) {
			const sd = `{"a":1`
			data := make(JSONMapSlice, 0)
			l := jlexer.Lexer{Data: []byte(sd)}
			data.UnmarshalEasyJSON(&l)
			require.Error(t, l.Error())
		})
		t.Run("on invalid delimiter (2)", func(t *testing.T) {
			const sd = `{"a":[1}`
			data := make(JSONMapSlice, 0)
			l := jlexer.Lexer{Data: []byte(sd)}
			data.UnmarshalEasyJSON(&l)
			require.Error(t, l.Error())
		})
		t.Run("on invalid delimiter (3)", func(t *testing.T) {
			const sd = `{"a":[1,]}`
			data := make(JSONMapSlice, 0)
			l := jlexer.Lexer{Data: []byte(sd)}
			data.UnmarshalEasyJSON(&l)
			require.Error(t, l.Error())
		})
		t.Run("on invalid delimiter (4)", func(t *testing.T) {
			const sd = `{"a":[1],}`
			data := make(JSONMapSlice, 0)
			l := jlexer.Lexer{Data: []byte(sd)}
			data.UnmarshalEasyJSON(&l)
			require.Error(t, l.Error())
		})
		t.Run("on invalid delimiter (4)", func(t *testing.T) {
			const sd = `{"a":{"b":1}`
			data := make(JSONMapSlice, 0)
			l := jlexer.Lexer{Data: []byte(sd)}
			data.UnmarshalEasyJSON(&l)
			require.Error(t, l.Error())
		})
	})
}

// deepObject builds a JSON document nesting depth objects: {"a":{"a":...{}...}}.
func deepObject(depth int) []byte {
	return []byte(strings.Repeat(`{"a":`, depth) + `{}` + strings.Repeat(`}`, depth))
}

// deepArray builds a JSON document nesting depth arrays under one key: {"a":[[...[]...]]}.
func deepArray(depth int) []byte {
	return []byte(`{"a":` + strings.Repeat(`[`, depth) + strings.Repeat(`]`, depth) + `}`)
}

// deepMapSlice builds an in-memory JSONMapSlice nested depth levels deep.
func deepMapSlice(depth int) JSONMapSlice {
	m := JSONMapSlice{{Key: "leaf", Value: "x"}}
	for i := 0; i < depth; i++ {
		m = JSONMapSlice{{Key: "n", Value: m}}
	}

	return m
}

func TestMaxNestingDepthUnmarshal(t *testing.T) {
	t.Run("object nesting within the limit should unmarshal", func(t *testing.T) {
		var m JSONMapSlice
		require.NoError(t, m.UnmarshalJSON(deepObject(100)))
	})

	t.Run("object nesting at the default limit should unmarshal", func(t *testing.T) {
		var m JSONMapSlice
		require.NoError(t, m.UnmarshalJSON(deepObject(defaultMaxNestingDepth-1)))
	})

	t.Run("object nesting just beyond the default limit should error", func(t *testing.T) {
		var m JSONMapSlice
		err := m.UnmarshalJSON(deepObject(defaultMaxNestingDepth))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMaxNestingDepth)
	})

	t.Run("object nesting beyond the default limit should error, not crash", func(t *testing.T) {
		var m JSONMapSlice
		err := m.UnmarshalJSON(deepObject(defaultMaxNestingDepth + 5))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMaxNestingDepth)
	})

	t.Run("array nesting at the default limit should unmarshal", func(t *testing.T) {
		var m JSONMapSlice
		require.NoError(t, m.UnmarshalJSON(deepArray(defaultMaxNestingDepth-1)))
	})

	t.Run("array nesting beyond the default limit should error, not crash", func(t *testing.T) {
		var m JSONMapSlice
		err := m.UnmarshalJSON(deepArray(defaultMaxNestingDepth + 5))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMaxNestingDepth)
	})

	t.Run("mixed object and array nesting beyond the default limit should error, not crash", func(t *testing.T) {
		const depth = defaultMaxNestingDepth
		payload := []byte(strings.Repeat(`{"a":[`, depth) + `{}` + strings.Repeat(`]}`, depth))

		var m JSONMapSlice
		err := m.UnmarshalJSON(payload)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMaxNestingDepth)
	})

	t.Run("UnmarshalEasyJSON beyond the default limit should error, not crash", func(t *testing.T) {
		var m JSONMapSlice
		l := jlexer.Lexer{Data: deepArray(defaultMaxNestingDepth + 5)}
		m.UnmarshalEasyJSON(&l)
		require.Error(t, l.Error())
		assert.ErrorIs(t, l.Error(), ErrMaxNestingDepth)
	})

	t.Run("json.Unmarshal path beyond the default limit should error, not crash", func(t *testing.T) {
		var m JSONMapSlice
		require.Error(t, json.Unmarshal(deepObject(defaultMaxNestingDepth+5), &m))
	})

	t.Run("configurable limit should be honored", func(t *testing.T) {
		const maxDepth = 5

		var okMap JSONMapSlice
		okLexer := jlexer.Lexer{Data: deepObject(4)}
		okMap.unmarshalEasyJSON(&okLexer, maxDepth)
		require.NoError(t, okLexer.Error())

		var badMap JSONMapSlice
		badLexer := jlexer.Lexer{Data: deepObject(10)}
		badMap.unmarshalEasyJSON(&badLexer, maxDepth)
		require.Error(t, badLexer.Error())
		assert.ErrorIs(t, badLexer.Error(), ErrMaxNestingDepth)
	})
}

func TestMaxNestingDepthMarshal(t *testing.T) {
	t.Run("MarshalJSON beyond the default limit should error, not crash", func(t *testing.T) {
		_, err := deepMapSlice(defaultMaxNestingDepth + 5).MarshalJSON()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMaxNestingDepth)
	})

	t.Run("MarshalJSON within the limit should marshal", func(t *testing.T) {
		_, err := deepMapSlice(100).MarshalJSON()
		require.NoError(t, err)
	})

	t.Run("MarshalEasyJSON beyond the default limit should error, not crash", func(t *testing.T) {
		w := &jwriter.Writer{}
		deepMapSlice(defaultMaxNestingDepth + 5).MarshalEasyJSON(w)
		_, err := w.BuildBytes()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMaxNestingDepth)
	})

	t.Run("json.Marshal path beyond the default limit should error, not crash", func(t *testing.T) {
		_, err := json.Marshal(deepMapSlice(defaultMaxNestingDepth + 5))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMaxNestingDepth)
	})

	t.Run("document accepted by the unmarshaler should marshal back", func(t *testing.T) {
		var m JSONMapSlice
		require.NoError(t, m.UnmarshalJSON(deepObject(defaultMaxNestingDepth-1)))

		b, err := m.MarshalJSON()
		require.NoError(t, err)
		assert.Equal(t, string(deepObject(defaultMaxNestingDepth-1)), string(b))
	})

	t.Run("configurable limit should be honored", func(t *testing.T) {
		w := &jwriter.Writer{}
		deepMapSlice(10).marshalEasyJSON(w, 5)
		_, err := w.BuildBytes()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMaxNestingDepth)
	})
}
