package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestVersionLockstep pins the tool-version surfaces together: the main.version
// source default, the VERSION file, and web/site.json's
// meta.implementation_version (the surface the product site renders) must all
// agree, so a bump that touches only one surface fails here. The release
// workflow overrides main.version at build time via -ldflags; this test guards
// the source default that `go run`/source builds report.
func TestVersionLockstep(t *testing.T) {
	// go test runs each package with its directory as cwd; the repo root holds
	// VERSION and web/site.json.
	versionFile, err := os.ReadFile("../../VERSION")
	if err != nil {
		t.Fatalf("read VERSION: %v", err)
	}
	if v := strings.TrimSpace(string(versionFile)); v != version {
		t.Errorf("VERSION = %q but main.version = %q — bump both in lockstep", v, version)
	}

	siteData, err := os.ReadFile("../../web/site.json")
	if err != nil {
		t.Fatalf("read web/site.json: %v", err)
	}
	var site struct {
		Meta struct {
			ImplementationVersion string `json:"implementation_version"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(siteData, &site); err != nil {
		t.Fatalf("parse web/site.json: %v", err)
	}
	if site.Meta.ImplementationVersion != version {
		t.Errorf("web/site.json implementation_version = %q but main.version = %q — bump both in lockstep",
			site.Meta.ImplementationVersion, version)
	}
}
