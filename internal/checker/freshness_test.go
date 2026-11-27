package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func freshnessServer(t *testing.T, block string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			Params []any
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		switch req.Method {
		case "eth_chainId":
			fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":"0x940"}`)
		case "eth_getBlockByNumber":
			if len(req.Params) != 2 || req.Params[0] != "latest" || req.Params[1] != false {
				t.Errorf("unexpected params: %v", req.Params)
			}
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":%s}`, block)
		default:
			t.Errorf("unexpected method %s", req.Method)
			w.WriteHeader(400)
		}
	}))
}

func TestBlockFreshness(t *testing.T) {
	now := time.Now().Unix()
	for _, tc := range []struct{ name, block, wantError string }{
		{"fresh", fmt.Sprintf(`{"number":"0x12","timestamp":"0x%x"}`, now-2), ""},
		{"stale", fmt.Sprintf(`{"number":"0x12","timestamp":"0x%x"}`, now-300), "exceeds maximum"},
		{"future", fmt.Sprintf(`{"number":"0x12","timestamp":"0x%x"}`, now+300), "in the future"},
		{"missing", `{"number":"0x12"}`, "timestamp"},
		{"malformed", `{"number":"0x12","timestamp":"0xzz"}`, "invalid hex"},
		{"null", "null", "missing result"},
		{"overflow", `{"number":"0x12","timestamp":"0x10000000000000000"}`, "invalid hex"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := freshnessServer(t, tc.block)
			defer srv.Close()
			r := NewClient(time.Second).CheckWithFreshness(context.Background(), srv.URL, 0, time.Minute)
			if r.Healthy != (tc.wantError == "") || !strings.Contains(r.Error, tc.wantError) {
				t.Fatalf("unexpected result: %+v", r)
			}
			if tc.name == "fresh" && (r.BlockNumber != 18 || r.BlockTimestamp == nil || r.BlockAgeSeconds == nil) {
				t.Fatalf("missing diagnostics: %+v", r)
			}
		})
	}
}

func TestCompareFreshnessExcludesStaleReference(t *testing.T) {
	now := time.Now().Unix()
	fresh := freshnessServer(t, fmt.Sprintf(`{"number":"0x12","timestamp":"0x%x"}`, now-2))
	defer fresh.Close()
	stale := freshnessServer(t, fmt.Sprintf(`{"number":"0xff","timestamp":"0x%x"}`, now-300))
	defer stale.Close()
	r := NewClient(time.Second).CompareWithFreshness(context.Background(), []string{fresh.URL, stale.URL}, 0, 5, time.Minute)
	if r.Healthy || r.ReferenceBlock == nil || *r.ReferenceBlock != 18 || !r.Results[0].Healthy || r.Results[1].Healthy {
		t.Fatalf("unexpected comparison: %+v", r)
	}
}
