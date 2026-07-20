package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestAccountTestServiceValidateUpstreamBaseURLAllowsConfiguredHTTP(t *testing.T) {
	svc := &AccountTestService{cfg: &config.Config{}}
	svc.cfg.Security.URLAllowlist.Enabled = false
	svc.cfg.Security.URLAllowlist.AllowInsecureHTTP = true

	got, err := svc.validateUpstreamBaseURL("http://167.172.84.63:8080/")
	if err != nil {
		t.Fatalf("expected configured HTTP upstream to pass, got %v", err)
	}
	if got != "http://167.172.84.63:8080" {
		t.Fatalf("normalized URL = %q, want %q", got, "http://167.172.84.63:8080")
	}
}

func TestAccountTestServiceValidateUpstreamBaseURLRejectsHTTPWhenDisabled(t *testing.T) {
	svc := &AccountTestService{cfg: &config.Config{}}
	svc.cfg.Security.URLAllowlist.Enabled = false
	svc.cfg.Security.URLAllowlist.AllowInsecureHTTP = false

	if _, err := svc.validateUpstreamBaseURL("http://167.172.84.63:8080/"); err == nil {
		t.Fatal("expected HTTP upstream to be rejected when insecure HTTP is disabled")
	}
}
