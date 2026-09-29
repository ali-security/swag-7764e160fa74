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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type SharedCounters struct {
	Counter1 int64 `json:"counter1,omitempty"`
	Counter2 int64 `json:"counter2:,omitempty"` // the ":" in the json field name is left on-purpose for this test
}

type AggregationObject struct {
	SharedCounters

	Count int64 `json:"count,omitempty"`
}

func (m *AggregationObject) UnmarshalJSON(raw []byte) error {
	// AO0
	var aO0 SharedCounters
	if err := ReadJSON(raw, &aO0); err != nil {
		return err
	}

	m.SharedCounters = aO0

	// now for regular properties
	var propsAggregationObject struct {
		Count int64 `json:"count,omitempty"`
	}
	if err := ReadJSON(raw, &propsAggregationObject); err != nil {
		return err
	}

	m.Count = propsAggregationObject.Count

	return nil
}

// MarshalJSON marshals this object to a JSON structure
func (m AggregationObject) MarshalJSON() ([]byte, error) {
	_parts := make([][]byte, 0, 1)

	aO0, err := WriteJSON(m.SharedCounters)
	if err != nil {
		return nil, err
	}
	_parts = append(_parts, aO0)

	// now for regular properties
	var propsAggregationObject struct {
		Count int64 `json:"count,omitempty"`
	}
	propsAggregationObject.Count = m.Count

	jsonDataPropsAggregationObject, errAggregationObject := WriteJSON(propsAggregationObject)
	if errAggregationObject != nil {
		return nil, errAggregationObject
	}
	_parts = append(_parts, jsonDataPropsAggregationObject)

	return ConcatJSON(_parts...), nil
}

func TestReadWriteJSON(t *testing.T) {
	obj := AggregationObject{Count: 290, SharedCounters: SharedCounters{Counter1: 304, Counter2: 948}}

	t.Run("with default adapter", func(t *testing.T) {
		t.Run("should WriteJSON from struct", func(t *testing.T) {
			rtjson, err := WriteJSON(obj)
			require.NoError(t, err)

			t.Run("should MarshalJSON using WriteJSON from this type", func(t *testing.T) {
				otjson, err := obj.MarshalJSON()
				require.NoError(t, err)

				t.Run("both marshaling methods should be equivalent", func(t *testing.T) {
					require.JSONEq(t, string(rtjson), string(otjson))
				})
			})

			t.Run("should MarshalJSON using the standard library", func(t *testing.T) {
				otjson, err := json.Marshal(obj)
				require.NoError(t, err)

				t.Run("both marshaling methods should be equivalent", func(t *testing.T) {
					require.JSONEq(t, string(rtjson), string(otjson))
				})
			})

			t.Run("should ReadJSON into new struct", func(t *testing.T) {
				var obj1 AggregationObject
				require.NoError(t, ReadJSON(rtjson, &obj1))

				t.Run("this should copy the object", func(t *testing.T) {
					require.Equal(t, obj, obj1)
				})
			})

			t.Run("should UnmarshalJSON using ReadJSON into new struct", func(t *testing.T) {
				var obj11 AggregationObject
				require.NoError(t, obj11.UnmarshalJSON(rtjson))

				t.Run("this should copy the object", func(t *testing.T) {
					require.Equal(t, obj, obj11)
				})
			})

			t.Run("should UnmarshalJSON using the standard library", func(t *testing.T) {
				var obj11 AggregationObject
				require.NoError(t, json.Unmarshal(rtjson, &obj11))

				t.Run("this should copy the object", func(t *testing.T) {
					require.Equal(t, obj, obj11)
				})
			})
		})

		t.Run("with counters", func(t *testing.T) {
			t.Run("should ReadJSON into struct", func(t *testing.T) {
				jsons := `{"counter1":123,"counter2:":456,"count":999}`
				var obj2 AggregationObject

				require.NoError(t, ReadJSON([]byte(jsons), &obj2))
				require.Equal(t, AggregationObject{SharedCounters: SharedCounters{Counter1: 123, Counter2: 456}, Count: 999}, obj2)
			})
		})
		t.Run("using FromDynamicJSON", func(t *testing.T) {
			const epsilon = 1e-6
			var obj2 interface{}

			require.NoError(t, FromDynamicJSON(obj, &obj2))
			asMap, ok := obj2.(map[string]interface{})
			require.True(t, ok)
			assert.Len(t, asMap, 3) // 3 fields in struct
			c1, ok := asMap["counter1"]
			require.True(t, ok)
			assert.InDelta(t, float64(304), c1, epsilon)

			c2, ok := asMap["counter2:"]
			require.True(t, ok)
			assert.InDelta(t, float64(948), c2, epsilon)

			c, ok := asMap["count"]
			require.True(t, ok)
			assert.InDelta(t, float64(290), c, epsilon)
		})

		t.Run("error in FromDynamicJSON (1)", func(t *testing.T) {
			var obj2 interface{}

			require.Error(t, FromDynamicJSON(obj, obj2)) // target is not a pointer
		})

		t.Run("error in FromDynamicJSON (2)", func(t *testing.T) {
			var obj2 interface{}
			var source struct {
				A int `json:"a"`
				B func()
			}
			require.Error(t, FromDynamicJSON(source, obj2))
		})
	})
}

// TestReadJSONDeepNestingDoesNotCrash exercises the advisory scenario end-to-end:
// a deeply nested document must return an error instead of driving the runtime to a
// non-recoverable stack overflow (CWE-674, unchecked recursion).
func TestReadJSONDeepNestingDoesNotCrash(t *testing.T) {
	t.Run("deeply nested arrays should error, not crash", func(t *testing.T) {
		const depth = 20000 // well beyond the default 10000 limit
		payload := []byte(`{"a":` + strings.Repeat("[", depth) + strings.Repeat("]", depth) + `}`)

		var v JSONMapSlice
		require.Error(t, ReadJSON(payload, &v))
	})

	t.Run("deeply nested objects should error, not crash", func(t *testing.T) {
		const depth = 20000
		payload := []byte(strings.Repeat(`{"a":`, depth) + `{}` + strings.Repeat(`}`, depth))

		var v JSONMapSlice
		require.Error(t, ReadJSON(payload, &v))
	})

	t.Run("adversarial multi-megabyte nesting should error, not overflow the stack", func(t *testing.T) {
		// without a depth guard, this many nested containers exhausts the goroutine stack
		// limit and aborts the whole process with a fatal, non-recoverable error.
		const depth = 10_000_000
		payload := []byte(`{"a":` + strings.Repeat("[", depth))

		var v JSONMapSlice
		err := ReadJSON(payload, &v)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMaxNestingDepth)
	})

	t.Run("moderately nested document should round-trip cleanly", func(t *testing.T) {
		const depth = 200
		payload := []byte(strings.Repeat(`{"a":`, depth) + `{}` + strings.Repeat(`}`, depth))

		var v JSONMapSlice
		require.NoError(t, ReadJSON(payload, &v))

		b, err := WriteJSON(v)
		require.NoError(t, err)
		assert.Equal(t, string(payload), string(b))
	})
}

// TestWriteJSONDeepNestingDoesNotCrash covers the marshal path: a deep in-memory
// JSONMapSlice must error rather than overflow the stack.
func TestWriteJSONDeepNestingDoesNotCrash(t *testing.T) {
	t.Run("deeply nested value should error, not crash", func(t *testing.T) {
		const depth = 20000
		v := JSONMapSlice{{Key: "leaf", Value: "x"}}
		for i := 0; i < depth; i++ {
			v = JSONMapSlice{{Key: "n", Value: v}}
		}

		_, err := WriteJSON(v)
		require.Error(t, err)
	})
}
