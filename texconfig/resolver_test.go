// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package texconfig

import "testing"

func TestResolveTexConvertPathSeparators(t *testing.T) {
	cfg := TexConvertConfig{Hints: []TextureHint{{ClassName: "NormalMap", Pattern: "*_nohq.paa"}}}
	paths := []string{
		"dir/sub/foo_nohq.paa",
		`dir\sub\foo_nohq.paa`,
		`dir/sub\foo_nohq.paa`,
		"foo_nohq.paa",
		`DIR\Sub\FOO_NOHQ.PAA`,
	}

	for _, name := range paths {
		t.Run(name, func(t *testing.T) {
			hint, ok := Resolve(name, cfg)
			if !ok {
				t.Fatalf("Resolve(%q) did not match", name)
			}
			if hint.ClassName != "NormalMap" {
				t.Fatalf("Resolve(%q) class = %q, want NormalMap", name, hint.ClassName)
			}
		})
	}
}
