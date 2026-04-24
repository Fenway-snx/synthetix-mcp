package types

import (
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_utils_test "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/test"
)

func Test_OrderIdArray_Scan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   any
		want    OrderIdArray
		wantErr bool
	}{
		{
			name:  "scan nil",
			value: nil,
			want:  OrderIdArray{},
		},
		{
			name:  "scan empty array",
			value: []byte("[]"),
			want:  OrderIdArray{},
		},
		{
			name:  "scan single element with both fields",
			value: []byte(`[{"void":42,"cloid":"client-123"}]`),
			want: OrderIdArray{
				{VenueId: 42, ClientId: "client-123"},
			},
		},
		{
			name:  "scan single element with empty client id",
			value: []byte(`[{"void":42,"cloid":""}]`),
			want: OrderIdArray{
				{VenueId: 42, ClientId: ""},
			},
		},
		{
			name:  "scan single element without client id field",
			value: []byte(`[{"void":42}]`),
			want: OrderIdArray{
				{VenueId: 42, ClientId: ""},
			},
		},
		{
			name:  "scan multiple elements",
			value: []byte(`[{"void":1,"cloid":"a"},{"void":2,"cloid":"b"},{"void":3,"cloid":"c"}]`),
			want: OrderIdArray{
				{VenueId: 1, ClientId: "a"},
				{VenueId: 2, ClientId: "b"},
				{VenueId: 3, ClientId: "c"},
			},
		},
		{
			name:    "scan zero venue id",
			value:   []byte(`[{"void":0,"cloid":"zero-id"}]`),
			wantErr: true,
		},
		{
			name:  "scan max uint64 venue id",
			value: []byte(`[{"void":18446744073709551615,"cloid":"max-id"}]`),
			want: OrderIdArray{
				{VenueId: 18446744073709551615, ClientId: "max-id"},
			},
		},
		{
			name:  "scan long client id",
			value: []byte(`[{"void":100,"cloid":"very-long-client-identifier-string-12345"}]`),
			want: OrderIdArray{
				{VenueId: 100, ClientId: "very-long-client-identifier-string-12345"},
			},
		},
		{
			name:  "scan special characters in client id",
			value: []byte(`[{"void":200,"cloid":"client-id-with-special-chars-!@#$%^&*()"}]`),
			want: OrderIdArray{
				{VenueId: 200, ClientId: "client-id-with-special-chars-!@#$%^&*()"},
			},
		},
		{
			name:  "scan unicode in client id",
			value: []byte(`[{"void":300,"cloid":"client-🚀-emoji"}]`),
			want: OrderIdArray{
				{VenueId: 300, ClientId: "client-🚀-emoji"},
			},
		},
		{
			name:    "scan invalid json",
			value:   []byte("invalid json"),
			wantErr: true,
		},
		{
			name:    "scan malformed array",
			value:   []byte(`[{"void":1,"cloid":"a"},`),
			wantErr: true,
		},
		{
			name:    "scan non-byte type",
			value:   "not bytes",
			wantErr: true,
		},
		{
			name:    "scan int type",
			value:   42,
			wantErr: true,
		},
		{
			name:    "scan string type",
			value:   "string",
			wantErr: true,
		},
		{
			name:    "scan json with missing venue id",
			value:   []byte(`[{"cloid":"missing-void"}]`),
			wantErr: true,
		},
		{
			name:    "scan json with invalid venue id type",
			value:   []byte(`[{"void":"not-a-number","cloid":"test"}]`),
			wantErr: true,
		},
		{
			name:    "scan json with negative venue id",
			value:   []byte(`[{"void":-1,"cloid":"test"}]`),
			wantErr: true,
		},
		{
			name:    "scan json with float venue id",
			value:   []byte(`[{"void":1.5,"cloid":"test"}]`),
			wantErr: true,
		},
		{
			name:    "scan json with invalid client id type",
			value:   []byte(`[{"void":1,"cloid":123}]`),
			wantErr: true,
		},
		{
			name:    "scan json with array instead of object",
			value:   []byte(`[[1,"test"]]`),
			wantErr: true,
		},
		{
			name:  "scan single object (not wrapped in array)",
			value: []byte(`{"void":42,"cloid":"client-123"}`),
			want: OrderIdArray{
				{VenueId: 42, ClientId: "client-123"},
			},
		},
		{
			name:  "scan single object without client id",
			value: []byte(`{"void":42}`),
			want: OrderIdArray{
				{VenueId: 42, ClientId: ""},
			},
		},
		{
			name:    "scan single object with zero venue id",
			value:   []byte(`{"void":0,"cloid":"test"}`),
			wantErr: true,
		},
		{
			name:  "scan string with valid array",
			value: `[{"void":42,"cloid":"from-string"}]`,
			want: OrderIdArray{
				{VenueId: 42, ClientId: "from-string"},
			},
		},
		{
			name:  "scan string with valid single object",
			value: `{"void":42,"cloid":"from-string"}`,
			want: OrderIdArray{
				{VenueId: 42, ClientId: "from-string"},
			},
		},
		{
			name:  "scan json null bytes",
			value: []byte("null"),
			want:  OrderIdArray{},
		},
		{
			name:  "scan json null string",
			value: "null",
			want:  OrderIdArray{},
		},
		{
			name:  "scan empty string",
			value: "",
			want:  OrderIdArray{},
		},
		{
			name:  "scan string empty array",
			value: "[]",
			want:  OrderIdArray{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var arr OrderIdArray
			err := arr.Scan(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
				assert.Equal(t, tt.want, arr)
			}
		})
	}
}

func Test_OrderIdArray_Value(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		arr     OrderIdArray
		want    driver.Value
		wantErr bool
	}{
		{
			name: "value empty array",
			arr:  OrderIdArray{},
			want: "[]",
		},
		{
			name: "value nil array",
			arr:  nil,
			want: "[]",
		},
		{
			name: "value single element with both fields",
			arr: OrderIdArray{
				{VenueId: 42, ClientId: "client-123"},
			},
			want: snx_lib_utils_test.MustMarshalJSON(OrderIdArray{
				{VenueId: 42, ClientId: "client-123"},
			}),
		},
		{
			name: "value single element with empty client id",
			arr: OrderIdArray{
				{VenueId: 42, ClientId: ""},
			},
			want: snx_lib_utils_test.MustMarshalJSON(OrderIdArray{
				{VenueId: 42, ClientId: ""},
			}),
		},
		{
			name: "value multiple elements",
			arr: OrderIdArray{
				{VenueId: 1, ClientId: "a"},
				{VenueId: 2, ClientId: "b"},
				{VenueId: 3, ClientId: "c"},
			},
			want: snx_lib_utils_test.MustMarshalJSON(OrderIdArray{
				{VenueId: 1, ClientId: "a"},
				{VenueId: 2, ClientId: "b"},
				{VenueId: 3, ClientId: "c"},
			}),
		},
		{
			name: "value zero venue id",
			arr: OrderIdArray{
				{VenueId: 0, ClientId: "zero-id"},
			},
			want: snx_lib_utils_test.MustMarshalJSON(OrderIdArray{
				{VenueId: 0, ClientId: "zero-id"},
			}),
		},
		{
			name: "value max uint64 venue id",
			arr: OrderIdArray{
				{VenueId: 18446744073709551615, ClientId: "max-id"},
			},
			want: snx_lib_utils_test.MustMarshalJSON(OrderIdArray{
				{VenueId: 18446744073709551615, ClientId: "max-id"},
			}),
		},
		{
			name: "value large array",
			arr: OrderIdArray{
				{VenueId: 100, ClientId: "a"},
				{VenueId: 200, ClientId: "b"},
				{VenueId: 300, ClientId: "c"},
				{VenueId: 400, ClientId: "d"},
				{VenueId: 500, ClientId: "e"},
			},
			want: snx_lib_utils_test.MustMarshalJSON(OrderIdArray{
				{VenueId: 100, ClientId: "a"},
				{VenueId: 200, ClientId: "b"},
				{VenueId: 300, ClientId: "c"},
				{VenueId: 400, ClientId: "d"},
				{VenueId: 500, ClientId: "e"},
			}),
		},
		{
			name: "value duplicate venue ids",
			arr: OrderIdArray{
				{VenueId: 1, ClientId: "a"},
				{VenueId: 1, ClientId: "b"},
				{VenueId: 2, ClientId: "c"},
			},
			want: snx_lib_utils_test.MustMarshalJSON(OrderIdArray{
				{VenueId: 1, ClientId: "a"},
				{VenueId: 1, ClientId: "b"},
				{VenueId: 2, ClientId: "c"},
			}),
		},
		{
			name: "value mixed empty and non-empty client ids",
			arr: OrderIdArray{
				{VenueId: 1, ClientId: ""},
				{VenueId: 2, ClientId: "non-empty"},
				{VenueId: 3, ClientId: ""},
			},
			want: snx_lib_utils_test.MustMarshalJSON(OrderIdArray{
				{VenueId: 1, ClientId: ""},
				{VenueId: 2, ClientId: "non-empty"},
				{VenueId: 3, ClientId: ""},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.arr.Value()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_OrderIdArray_ROUND_TRIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		arr  OrderIdArray
	}{
		{
			name: "round trip empty",
			arr:  OrderIdArray{},
		},
		{
			name: "round trip single with both fields",
			arr: OrderIdArray{
				{VenueId: 42, ClientId: "client-123"},
			},
		},
		{
			name: "round trip single with empty client id",
			arr: OrderIdArray{
				{VenueId: 42, ClientId: ""},
			},
		},
		{
			name: "round trip multiple",
			arr: OrderIdArray{
				{VenueId: 1, ClientId: "a"},
				{VenueId: 2, ClientId: "b"},
				{VenueId: 3, ClientId: "c"},
			},
		},
		{
			name: "round trip max uint64 venue id",
			arr: OrderIdArray{
				{VenueId: 18446744073709551615, ClientId: "max-id"},
			},
		},
		{
			name: "round trip large array",
			arr: OrderIdArray{
				{VenueId: 100, ClientId: "a"},
				{VenueId: 200, ClientId: "b"},
				{VenueId: 300, ClientId: "c"},
				{VenueId: 400, ClientId: "d"},
				{VenueId: 500, ClientId: "e"},
			},
		},
		{
			name: "round trip with special characters",
			arr: OrderIdArray{
				{VenueId: 100, ClientId: "client-🚀-emoji"},
			},
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			// Convert to driver.Value
			value, err := tt.arr.Value()
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

			// Convert back from driver.Value
			var result OrderIdArray
			err = result.Scan(value)
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

			// Verify round trip
			assert.Equal(t, tt.arr, result)
		})
	}
}
