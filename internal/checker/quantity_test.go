package checker

import (
	"encoding/json"
	"testing"
)

func TestParseQuantityCanonicalEncoding(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		value uint64
		valid bool
	}{
		{`"0x0"`, 0, true},
		{`"0x940"`, 2368, true},
		{`"0xFF"`, 255, true},
		{`"0xffffffffffffffff"`, ^uint64(0), true},
		{`"0x00"`, 0, false},
		{`"0x0940"`, 0, false},
		{`"0x+1"`, 0, false},
		{`"0x-1"`, 0, false},
		{`"0x1_0"`, 0, false},
		{`"0x"`, 0, false},
		{`"0x10000000000000000"`, 0, false},
		{`null`, 0, false},
		{`123`, 0, false},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			got, err := parseQuantity(json.RawMessage(tc.raw), "test quantity")
			if (err == nil) != tc.valid || (tc.valid && got != tc.value) {
				t.Fatalf("got %d, %v; want value %d valid %v", got, err, tc.value, tc.valid)
			}
		})
	}
}
