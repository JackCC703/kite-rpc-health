package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCompareLagAndWrongChain(t *testing.T) {
	fast := serverFor("0x940", "0x64")
	defer fast.Close()
	slow := serverFor("0x940", "0x5a")
	defer slow.Close()
	wrong := serverFor("0x1", "0xffff")
	defer wrong.Close()
	client := NewClient(time.Second)
	report := client.Compare(context.Background(), []string{slow.URL, wrong.URL, fast.URL}, 0, 5)
	if report.Healthy || report.ReferenceBlock == nil || *report.ReferenceBlock != 100 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if r := report.Results[0]; r.Healthy || r.Lag == nil || *r.Lag != 10 || !strings.Contains(r.Error, "exceeds maximum") {
		t.Fatalf("unexpected slow result: %+v", r)
	}
	if r := report.Results[1]; r.Healthy || r.Lag != nil {
		t.Fatalf("wrong chain must not participate: %+v", r)
	}
	if r := report.Results[2]; !r.Healthy || r.Lag == nil || *r.Lag != 0 {
		t.Fatalf("unexpected reference result: %+v", r)
	}
	report = client.Compare(context.Background(), []string{slow.URL, fast.URL}, 0, 10)
	if !report.Healthy {
		t.Fatalf("equal lag threshold should pass: %+v", report)
	}
	report = client.Compare(context.Background(), []string{slow.URL, fast.URL}, 95, 10)
	if report.Healthy || !strings.Contains(report.Results[0].Error, "below minimum") {
		t.Fatalf("minimum threshold should still apply: %+v", report)
	}
}

func TestCompareFailedEndpoints(t *testing.T) {
	failed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer failed.Close()
	wrong := serverFor("0x1", "0x20")
	defer wrong.Close()
	report := NewClient(time.Second).Compare(context.Background(), []string{failed.URL, wrong.URL}, 0, 5)
	if report.Healthy || report.ReferenceBlock != nil || report.Results[0].Lag != nil || !strings.Contains(report.Results[0].Error, "503") {
		t.Fatalf("failed endpoints must not establish a reference: %+v", report)
	}
}

func TestCompareZeroHeight(t *testing.T) {
	a := serverFor("0x940", "0x0")
	defer a.Close()
	b := serverFor("0x940", "0x0")
	defer b.Close()
	report := NewClient(time.Second).Compare(context.Background(), []string{a.URL, b.URL}, 0, 0)
	if !report.Healthy || report.ReferenceBlock == nil || *report.ReferenceBlock != 0 {
		t.Fatalf("valid zero height must be distinguishable from absent reference: %+v", report)
	}
}
