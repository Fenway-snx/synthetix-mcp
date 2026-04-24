package marshal

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helper types
// ---------------------------------------------------------------------------

type testTextMarshaler struct {
	Value string
}

func (t testTextMarshaler) MarshalText() ([]byte, error) {
	return []byte("custom:" + t.Value), nil
}

type testJSONMarshaler struct {
	Value int
}

func (j testJSONMarshaler) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]int{"custom": j.Value})
}

type testPointerReceiverMarshaler struct {
	Value string
}

func (p *testPointerReceiverMarshaler) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"ptr": p.Value})
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func Test_SafeMarshalJSON_NilInput(t *testing.T) {
	b, err := SafeMarshalJSON(nil)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, `null`, string(b))
}

func Test_SafeMarshalJSON_Primitives(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"int", 42, "42"},
		{"int64", int64(123), "123"},
		{"uint", uint(7), "7"},
		{"uint64", uint64(999), "999"},
		{"float64", 3.14, "3.14"},
		{"string", "hello", `"hello"`},
		{"empty string", "", `""`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, err := SafeMarshalJSON(tc.input)
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.JSONEq(t, tc.expected, string(b))
		})
	}
}

func Test_SafeMarshalJSON_NonFiniteFloats(t *testing.T) {
	tests := []struct {
		name  string
		input float64
	}{
		{"NaN", math.NaN()},
		{"+Inf", math.Inf(1)},
		{"-Inf", math.Inf(-1)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, err := SafeMarshalJSON(tc.input)
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.JSONEq(t, "null", string(b))
		})
	}
}

func Test_SafeMarshalJSON_SimpleStruct(t *testing.T) {
	type Simple struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	b, err := SafeMarshalJSON(Simple{Name: "Alice", Age: 30})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, `{"name":"Alice","age":30}`, string(b))
}

func Test_SafeMarshalJSON_StructWithFuncField(t *testing.T) {
	type WithFunc struct {
		Name string `json:"name"`
		Fn   func() `json:"fn"`
	}

	b, err := SafeMarshalJSON(WithFunc{Name: "test", Fn: func() {}})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(b, &parsed))
	assert.Equal(t, "test", parsed["name"])
	_, hasFn := parsed["fn"]
	assert.False(t, hasFn, "func field should be omitted")
}

func Test_SafeMarshalJSON_StructWithChanField(t *testing.T) {
	type WithChan struct {
		Name string   `json:"name"`
		Ch   chan int `json:"ch"`
	}

	b, err := SafeMarshalJSON(WithChan{Name: "test", Ch: make(chan int)})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(b, &parsed))
	assert.Equal(t, "test", parsed["name"])
	_, hasCh := parsed["ch"]
	assert.False(t, hasCh, "chan field should be omitted")
}

func Test_SafeMarshalJSON_StructWithComplexField(t *testing.T) {
	type WithComplex struct {
		Name string     `json:"name"`
		C    complex128 `json:"c"`
	}

	b, err := SafeMarshalJSON(WithComplex{Name: "test", C: 1 + 2i})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(b, &parsed))
	assert.Equal(t, "test", parsed["name"])
	_, hasC := parsed["c"]
	assert.False(t, hasC, "complex field should be omitted")
}

func Test_SafeMarshalJSON_StructWithAllUnsafeFieldTypes(t *testing.T) {
	type AllUnsafe struct {
		Fn  func()     `json:"fn"`
		Ch  chan int   `json:"ch"`
		C64 complex64  `json:"c64"`
		C   complex128 `json:"c"`
	}

	b, err := SafeMarshalJSON(AllUnsafe{
		Fn:  func() {},
		Ch:  make(chan int),
		C64: 1 + 2i,
		C:   3 + 4i,
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, `{}`, string(b))
}

func Test_SafeMarshalJSON_StructTags(t *testing.T) {
	type Tagged struct {
		Exported   string `json:"exported"`
		Renamed    string `json:"custom_name"`
		Skipped    string `json:"-"`
		DashName   string `json:"-,"`
		unexported string //nolint:unused
	}

	b, err := SafeMarshalJSON(Tagged{
		Exported:   "yes",
		Renamed:    "renamed",
		Skipped:    "should not appear",
		DashName:   "dash",
		unexported: "also hidden",
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(b, &result))

	assert.Equal(t, "yes", result["exported"])
	assert.Equal(t, "renamed", result["custom_name"])
	assert.Equal(t, "dash", result["-"])
	_, hasSkipped := result["Skipped"]
	assert.False(t, hasSkipped)
	_, hasUnexported := result["unexported"]
	assert.False(t, hasUnexported)
}

func Test_SafeMarshalJSON_Omitempty(t *testing.T) {
	type WithOmitempty struct {
		Name   string   `json:"name,omitempty"`
		Age    int      `json:"age,omitempty"`
		Active bool     `json:"active,omitempty"`
		Tags   []string `json:"tags,omitempty"`
	}

	t.Run("all zero values omitted", func(t *testing.T) {
		b, err := SafeMarshalJSON(WithOmitempty{})
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.JSONEq(t, `{}`, string(b))
	})

	t.Run("non-zero values present", func(t *testing.T) {
		b, err := SafeMarshalJSON(WithOmitempty{
			Name:   "Alice",
			Age:    30,
			Active: true,
			Tags:   []string{"admin"},
		})
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.JSONEq(t, `{"name":"Alice","age":30,"active":true,"tags":["admin"]}`, string(b))
	})
}

func Test_SafeMarshalJSON_NilPointer(t *testing.T) {
	var p *int
	b, err := SafeMarshalJSON(p)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, "null", string(b))
}

func Test_SafeMarshalJSON_NonNilPointer(t *testing.T) {
	v := 42
	b, err := SafeMarshalJSON(&v)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, "42", string(b))
}

func Test_SafeMarshalJSON_DoublePointer(t *testing.T) {
	v := 42
	p := &v
	b, err := SafeMarshalJSON(&p)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, "42", string(b))
}

func Test_SafeMarshalJSON_NilSlice(t *testing.T) {
	var s []int
	b, err := SafeMarshalJSON(s)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, "null", string(b))
}

func Test_SafeMarshalJSON_EmptySlice(t *testing.T) {
	s := []int{}
	b, err := SafeMarshalJSON(s)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, "[]", string(b))
}

func Test_SafeMarshalJSON_Slice(t *testing.T) {
	s := []int{1, 2, 3}
	b, err := SafeMarshalJSON(s)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, "[1,2,3]", string(b))
}

func Test_SafeMarshalJSON_Array(t *testing.T) {
	arr := [3]int{10, 20, 30}
	b, err := SafeMarshalJSON(arr)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, "[10,20,30]", string(b))
}

func Test_SafeMarshalJSON_NilMap(t *testing.T) {
	var m map[string]int
	b, err := SafeMarshalJSON(m)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, "null", string(b))
}

func Test_SafeMarshalJSON_Map(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	b, err := SafeMarshalJSON(m)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, `{"a":1,"b":2}`, string(b))
}

func Test_SafeMarshalJSON_MapWithFuncValue(t *testing.T) {
	m := map[string]any{
		"name": "test",
		"fn":   func() {},
	}
	b, err := SafeMarshalJSON(m)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(b, &result))
	assert.Equal(t, "test", result["name"])
	_, hasFn := result["fn"]
	assert.False(t, hasFn, "func map entry should be omitted")
}

func Test_SafeMarshalJSON_MapWithIntKeys(t *testing.T) {
	m := map[int]string{1: "one", 2: "two"}
	b, err := SafeMarshalJSON(m)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var result map[string]string
	require.NoError(t, json.Unmarshal(b, &result))
	assert.Equal(t, "one", result["1"])
	assert.Equal(t, "two", result["2"])
}

func Test_SafeMarshalJSON_ByteSlice(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	b, err := SafeMarshalJSON(data)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var s string
	require.NoError(t, json.Unmarshal(b, &s))
	assert.Equal(t, "AQID", s, "[]byte should be base64-encoded")
}

func Test_SafeMarshalJSON_NilByteSlice(t *testing.T) {
	var data []byte
	b, err := SafeMarshalJSON(data)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, "null", string(b))
}

func Test_SafeMarshalJSON_JSONMarshaler(t *testing.T) {
	v := testJSONMarshaler{Value: 42}
	b, err := SafeMarshalJSON(v)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, `{"custom":42}`, string(b))
}

func Test_SafeMarshalJSON_TextMarshaler(t *testing.T) {
	v := testTextMarshaler{Value: "hello"}
	b, err := SafeMarshalJSON(v)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, `"custom:hello"`, string(b))
}

func Test_SafeMarshalJSON_PointerReceiverMarshaler(t *testing.T) {
	v := testPointerReceiverMarshaler{Value: "boxed"}
	b, err := SafeMarshalJSON(v)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, `{"ptr":"boxed"}`, string(b))
}

func Test_SafeMarshalJSON_Time(t *testing.T) {
	ts := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	b, err := SafeMarshalJSON(ts)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var s string
	require.NoError(t, json.Unmarshal(b, &s))
	assert.Contains(t, s, "2025-01-15")
}

func Test_SafeMarshalJSON_CyclicPointer(t *testing.T) {
	type Node struct {
		Name string `json:"name"`
		Next *Node  `json:"next"`
	}

	a := &Node{Name: "a"}
	nodeB := &Node{Name: "b"}
	a.Next = nodeB
	nodeB.Next = a

	result, err := SafeMarshalJSON(a)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(result, &parsed))
	assert.Equal(t, "a", parsed["name"])

	next, ok := parsed["next"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "b", next["name"])
	assert.Nil(t, next["next"], "cycle should be broken with null")
}

func Test_SafeMarshalJSON_SharedPointer(t *testing.T) {
	type Leaf struct {
		Value string `json:"value"`
	}
	type Root struct {
		A *Leaf `json:"a"`
		B *Leaf `json:"b"`
	}

	shared := &Leaf{Value: "shared"}
	root := Root{A: shared, B: shared}

	b, err := SafeMarshalJSON(root)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(b, &parsed))

	aVal, ok := parsed["a"].(map[string]any)
	require.True(t, ok)
	bVal, ok := parsed["b"].(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "shared", aVal["value"])
	assert.Equal(t, "shared", bVal["value"], "shared pointer should be serialized for both references")
}

func Test_SafeMarshalJSON_NestedUnsafeFields(t *testing.T) {
	type Inner struct {
		Safe   string `json:"safe"`
		Unsafe func() `json:"unsafe"`
	}
	type Outer struct {
		Inner Inner  `json:"inner"`
		Name  string `json:"name"`
	}

	b, err := SafeMarshalJSON(Outer{
		Inner: Inner{Safe: "ok", Unsafe: func() {}},
		Name:  "test",
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(b, &parsed))
	assert.Equal(t, "test", parsed["name"])

	inner, ok := parsed["inner"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ok", inner["safe"])
	_, hasUnsafe := inner["unsafe"]
	assert.False(t, hasUnsafe, "nested func field should be omitted")
}

func Test_SafeMarshalJSON_EmbeddedStruct(t *testing.T) {
	type Base struct {
		ID string `json:"id"`
	}
	type Extended struct {
		Base
		Name string `json:"name"`
	}

	b, err := SafeMarshalJSON(Extended{
		Base: Base{ID: "123"},
		Name: "test",
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(b, &parsed))
	assert.Equal(t, "123", parsed["id"], "embedded field should be promoted")
	assert.Equal(t, "test", parsed["name"])
}

func Test_SafeMarshalJSON_EmbeddedStructWithExplicitName(t *testing.T) {
	type Base struct {
		ID string `json:"id"`
	}
	type Container struct {
		Base Base `json:"base"`
	}

	b, err := SafeMarshalJSON(Container{
		Base: Base{ID: "123"},
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(b, &parsed))

	base, ok := parsed["base"].(map[string]any)
	require.True(t, ok, "explicitly named embedded field should be nested")
	assert.Equal(t, "123", base["id"])
}

func Test_SafeMarshalJSON_EmbeddedPointerStruct(t *testing.T) {
	type Base struct {
		ID string `json:"id"`
	}
	type Extended struct {
		*Base
		Name string `json:"name"`
	}

	t.Run("non-nil embedded pointer", func(t *testing.T) {
		b, err := SafeMarshalJSON(Extended{
			Base: &Base{ID: "456"},
			Name: "test",
		})
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(b, &parsed))
		assert.Equal(t, "456", parsed["id"])
		assert.Equal(t, "test", parsed["name"])
	})

	t.Run("nil embedded pointer", func(t *testing.T) {
		b, err := SafeMarshalJSON(Extended{
			Base: nil,
			Name: "test",
		})
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(b, &parsed))
		_, hasID := parsed["id"]
		assert.False(t, hasID, "nil embedded pointer fields should not appear")
		assert.Equal(t, "test", parsed["name"])
	})
}

func Test_SafeMarshalJSON_EmptyStruct(t *testing.T) {
	type Empty struct{}
	b, err := SafeMarshalJSON(Empty{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, `{}`, string(b))
}

func Test_SafeMarshalJSON_InterfaceFieldWithFunc(t *testing.T) {
	type Container struct {
		Value any `json:"value"`
	}

	b, err := SafeMarshalJSON(Container{Value: func() {}})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(b, &parsed))
	assert.Nil(t, parsed["value"], "interface containing func should encode as null")
}

func Test_SafeMarshalJSON_InterfaceFieldWithSafeValue(t *testing.T) {
	type Container struct {
		Value any `json:"value"`
	}

	b, err := SafeMarshalJSON(Container{Value: "hello"})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, `{"value":"hello"}`, string(b))
}

func Test_SafeMarshalJSON_StructWithNonFiniteFloat(t *testing.T) {
	type WithFloat struct {
		Name  string  `json:"name"`
		Value float64 `json:"value"`
	}

	b, err := SafeMarshalJSON(WithFloat{Name: "test", Value: math.NaN()})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(b, &parsed))
	assert.Equal(t, "test", parsed["name"])
	assert.Nil(t, parsed["value"], "NaN should be encoded as null")
}

func Test_SafeMarshalJSON_TopLevelFunc(t *testing.T) {
	b, err := SafeMarshalJSON(func() {})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, "null", string(b))
}

func Test_SafeMarshalJSON_TopLevelChan(t *testing.T) {
	b, err := SafeMarshalJSON(make(chan int))
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, "null", string(b))
}

func Test_SafeMarshalJSON_RawMessage(t *testing.T) {
	raw := json.RawMessage(`{"precomputed":true}`)
	b, err := SafeMarshalJSON(raw)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.JSONEq(t, `{"precomputed":true}`, string(b))
}

func Test_SafeMarshalJSON_SliceOfMixedInterfaces(t *testing.T) {
	s := []any{
		"hello",
		42,
		func() {},
		true,
		make(chan int),
		3.14,
	}
	b, err := SafeMarshalJSON(s)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var parsed []any
	require.NoError(t, json.Unmarshal(b, &parsed))
	assert.Equal(t, "hello", parsed[0])
	assert.Equal(t, float64(42), parsed[1])
	assert.Nil(t, parsed[2], "func in slice should be null")
	assert.Equal(t, true, parsed[3])
	assert.Nil(t, parsed[4], "chan in slice should be null")
	assert.Equal(t, 3.14, parsed[5])
}

func Test_SafeMarshalJSON_DeeplyNestedStruct(t *testing.T) {
	type Level3 struct {
		Value string `json:"value"`
		Fn    func() `json:"fn"`
	}
	type Level2 struct {
		L3 Level3 `json:"l3"`
	}
	type Level1 struct {
		L2 Level2 `json:"l2"`
	}

	b, err := SafeMarshalJSON(Level1{
		L2: Level2{
			L3: Level3{Value: "deep", Fn: func() {}},
		},
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(b, &parsed))

	l2 := parsed["l2"].(map[string]any)
	l3 := l2["l3"].(map[string]any)
	assert.Equal(t, "deep", l3["value"])
	_, hasFn := l3["fn"]
	assert.False(t, hasFn)
}

func Test_SafeMarshalJSON_StructFieldWithMarshalerNotDecomposed(t *testing.T) {
	type Container struct {
		Custom testJSONMarshaler `json:"custom"`
		Name   string            `json:"name"`
	}

	b, err := SafeMarshalJSON(Container{
		Custom: testJSONMarshaler{Value: 99},
		Name:   "test",
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(b, &parsed))
	assert.Equal(t, "test", parsed["name"])

	custom, ok := parsed["custom"].(map[string]any)
	require.True(t, ok, "custom marshaler should produce its own representation")
	assert.Equal(t, float64(99), custom["custom"])
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func Benchmark_SafeMarshalJSON_SimpleStruct(b *testing.B) {
	type Simple struct {
		Name  string `json:"name"`
		Age   int    `json:"age"`
		Email string `json:"email"`
	}

	v := Simple{Name: "Alice", Age: 30, Email: "alice@example.com"}
	for b.Loop() {
		_, _ = SafeMarshalJSON(v)
	}
}

func Benchmark_SafeMarshalJSON_StructWithUnsafeFields(b *testing.B) {
	type Mixed struct {
		Name string `json:"name"`
		Fn   func()
		Ch   chan int
		Age  int `json:"age"`
	}

	v := Mixed{Name: "Alice", Fn: func() {}, Ch: make(chan int), Age: 30}
	for b.Loop() {
		_, _ = SafeMarshalJSON(v)
	}
}

func Benchmark_SafeMarshalJSON_vs_StdlibMarshal(b *testing.B) {
	type Simple struct {
		Name  string `json:"name"`
		Age   int    `json:"age"`
		Email string `json:"email"`
	}

	v := Simple{Name: "Alice", Age: 30, Email: "alice@example.com"}

	b.Run("SafeMarshalJSON", func(b *testing.B) {
		for b.Loop() {
			_, _ = SafeMarshalJSON(v)
		}
	})

	b.Run("json.Marshal", func(b *testing.B) {
		for b.Loop() {
			_, _ = json.Marshal(v)
		}
	})
}
