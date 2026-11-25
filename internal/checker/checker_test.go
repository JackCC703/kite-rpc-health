package checker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func serverFor(chain, block string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			tErr(w, 405)
			return
		}
		body := make([]byte, 512)
		n, _ := r.Body.Read(body)
		method := "eth_chainId"
		if strings.Contains(string(body[:n]), "eth_blockNumber") {
			method = "eth_blockNumber"
		}
		value := chain
		if method == "eth_blockNumber" {
			value = block
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":%q}`, value)
	}))
}
func tErr(w http.ResponseWriter, code int) { w.WriteHeader(code) }
func TestHealthyKite(t *testing.T) {
	srv := serverFor("0x940", "0x12")
	defer srv.Close()
	r := NewClient(time.Second).Check(context.Background(), srv.URL)
	if !r.Healthy || r.ChainID != 2368 || r.BlockNumber != 18 {
		t.Fatalf("unexpected check result: %+v", r)
	}
}
func TestWrongChain(t *testing.T) {
	srv := serverFor("0x1", "0x12")
	defer srv.Close()
	r := NewClient(time.Second).Check(context.Background(), srv.URL)
	if r.Healthy || r.ChainID != 1 || !strings.Contains(r.Error, "unexpected chain ID") {
		t.Fatalf("unexpected result: %+v", r)
	}
}
func TestRPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"unavailable"}}`)
	}))
	defer srv.Close()
	r := NewClient(time.Second).Check(context.Background(), srv.URL)
	if r.Healthy || !strings.Contains(r.Error, "unavailable") {
		t.Fatalf("unexpected result: %+v", r)
	}
}
func TestInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `invalid`) }))
	defer srv.Close()
	r := NewClient(time.Second).Check(context.Background(), srv.URL)
	if r.Healthy || !strings.Contains(r.Error, "invalid JSON") {
		t.Fatalf("unexpected result: %+v", r)
	}
}
func TestHTTPFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer srv.Close()
	r := NewClient(time.Second).Check(context.Background(), srv.URL)
	if r.Healthy || !strings.Contains(r.Error, "503") {
		t.Fatalf("unexpected result: %+v", r)
	}
}

func TestRPCTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":"0x940"}`)
	}))
	defer srv.Close()

	result := NewClient(10*time.Millisecond).Check(context.Background(), srv.URL)
	if result.Healthy || !strings.Contains(result.Error, "deadline exceeded") {
		t.Fatalf("expected timeout failure, got %+v", result)
	}
}

func TestMalformedRPCResponses(t *testing.T) {
	cases := []struct {
		name, body, expected string
	}{
		{"wrong response ID", `{"jsonrpc":"2.0","id":2,"result":"0x940"}`, "mismatched response ID"},
		{"wrong version", `{"jsonrpc":"1.0","id":1,"result":"0x940"}`, "invalid JSON-RPC version"},
		{"missing result", `{"jsonrpc":"2.0","id":1}`, "missing result"},
		{"invalid hex", `{"jsonrpc":"2.0","id":1,"result":"0xzz"}`, "invalid hex result"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, tc.body)
			}))
			defer srv.Close()
			result := NewClient(time.Second).Check(context.Background(), srv.URL)
			if result.Healthy || !strings.Contains(result.Error, tc.expected) {
				t.Fatalf("expected %q failure, got %+v", tc.expected, result)
			}
		})
	}
}

func TestBlockHeightFailurePreservesChainID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request: %v", err)
			return
		}
		if strings.Contains(string(body), "eth_blockNumber") {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":"0x940"}`)
	}))
	defer srv.Close()
	result := NewClient(time.Second).Check(context.Background(), srv.URL)
	if result.Healthy || result.ChainID != KiteTestnetChainID || !strings.Contains(result.Error, "503") {
		t.Fatalf("expected reported chain ID and failed block read, got %+v", result)
	}
}

func TestOversizedRPCResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, strings.Repeat("x", (1<<20)+1))
	}))
	defer srv.Close()
	result := NewClient(time.Second).Check(context.Background(), srv.URL)
	if result.Healthy || !strings.Contains(result.Error, "response too large") {
		t.Fatalf("expected body-size rejection, got %+v", result)
	}
}

func TestMinimumBlockThreshold(t *testing.T) {
	srv := serverFor("0x940", "0x12") // block 18
	defer srv.Close()
	for _, tc := range []struct {
		name    string
		minimum uint64
		healthy bool
	}{
		{"disabled", 0, true},
		{"below observed", 17, true},
		{"equal observed", 18, true},
		{"above observed", 19, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := NewClient(time.Second).CheckWithMinBlock(context.Background(), srv.URL, tc.minimum)
			if r.Healthy != tc.healthy || r.BlockNumber != 18 || r.ChainID != KiteTestnetChainID {
				t.Fatalf("unexpected result: %+v", r)
			}
			if !tc.healthy && !strings.Contains(r.Error, "below minimum") {
				t.Fatalf("missing stale-block diagnostic: %+v", r)
			}
		})
	}
}

func TestBlockHeightRPCFailurePrecedesMinimumBlockCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "eth_blockNumber") {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":"0x940"}`)
	}))
	defer srv.Close()
	r := NewClient(time.Second).CheckWithMinBlock(context.Background(), srv.URL, 100)
	if r.Healthy || !strings.Contains(r.Error, "503") || strings.Contains(r.Error, "below minimum") {
		t.Fatalf("should report the original RPC failure, got %+v", r)
	}
}
