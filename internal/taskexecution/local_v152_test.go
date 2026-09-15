package taskexecution

import (
	"context"
	"runtime"
	"strings"
	"testing"
)

func TestV152LocalDiscoveryKeepsCrossPlatformOSArchitectureBaseline(t *testing.T) {
	profile, err := (LocalEnvironment{}).Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(profile.OS) == "" {
		t.Fatal("local OS capability is empty")
	}
	if normalizeArchitecture(profile.Architecture) != normalizeArchitecture(runtime.GOARCH) {
		t.Fatalf("architecture=%q GOARCH=%q", profile.Architecture, runtime.GOARCH)
	}
}

func TestV152ArchitectureAliasesNormalize(t *testing.T) {
	cases := map[string]string{
		"x86_64": "amd64",
		"x64":    "amd64",
		"amd64":  "amd64",
		"aarch64": "arm64",
		"arm64":   "arm64",
	}
	for input, want := range cases {
		if got := normalizeArchitecture(input); got != want {
			t.Fatalf("normalizeArchitecture(%q)=%q want %q", input, got, want)
		}
	}
}
