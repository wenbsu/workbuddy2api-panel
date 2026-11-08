package main

import "testing"

func TestJetBrainsDefaultConfig(t *testing.T) {
	for _, raw := range []string{`{}`, `{"upstream":{"timeout_seconds":60}}`} {
		c, err := ParseConfig([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if c.Upstream.JetBrains == nil || c.Upstream.JetBrains.ProductVersion != "4.12.42888899" || c.Upstream.JetBrains.Platform != "JetBrainsIntelliJIdea" {
			t.Fatal("plugin profile missing from default config")
		}
	}
	c, err := ParseConfig([]byte(`{"upstream":{"jetbrains":null}}`))
	if err != nil {
		t.Fatal(err)
	}
	if c.Upstream.JetBrains != nil {
		t.Fatal("explicit null must select desktop")
	}
	c, err = ParseConfig([]byte(`{"upstream":{"jetbrains":{"platform_version":"2026.2"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if c.Upstream.JetBrains.PlatformVersion != "2026.2" || c.Upstream.JetBrains.ProductName != "CodeBuddy" {
		t.Fatal("partial override lost defaults")
	}
}
