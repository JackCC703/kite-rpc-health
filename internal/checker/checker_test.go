package checker

import (
	"context"
	"fmt"
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
	if r.Healthy || !strings.Contains(r.Error, "unexpected chain ID") {
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
