// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package texconfig

import "testing"

func cloneTestBoolPointer(value bool) *bool {
	return &value
}

func TestTexConvertConfigCloneCopiesGlobalsAndDeepCopiesHints(t *testing.T) {
	source := TexConvertConfig{
		ConvertVersion:           7,
		ApplyDefaultErrorMetrics: true,
		UseSRGBFromDynRange:      true,
		DisableAutoReduce:        true,
		DisableLZO:               true,
		Hints: []TextureHint{{
			EnableDXT:  cloneTestBoolPointer(true),
			DynRange:   cloneTestBoolPointer(false),
			AutoReduce: cloneTestBoolPointer(true),
			VirtualSwz: cloneTestBoolPointer(false),
			Dithering:  cloneTestBoolPointer(true),
			ClassName:  "sample",
			Extends:    "base",
			Pattern:    "*_sample*",
		}},
	}
	clone := source.Clone()

	if clone.ConvertVersion != source.ConvertVersion ||
		clone.ApplyDefaultErrorMetrics != source.ApplyDefaultErrorMetrics ||
		clone.UseSRGBFromDynRange != source.UseSRGBFromDynRange ||
		clone.DisableAutoReduce != source.DisableAutoReduce ||
		clone.DisableLZO != source.DisableLZO {
		t.Fatalf("clone lost config fields: %#v", clone)
	}
	if len(clone.Hints) != 1 || clone.Hints[0].ClassName != source.Hints[0].ClassName ||
		clone.Hints[0].Extends != source.Hints[0].Extends || clone.Hints[0].Pattern != source.Hints[0].Pattern {
		t.Fatalf("clone lost hint values: %#v", clone.Hints)
	}

	sourcePointers := []*bool{
		source.Hints[0].EnableDXT,
		source.Hints[0].DynRange,
		source.Hints[0].AutoReduce,
		source.Hints[0].VirtualSwz,
		source.Hints[0].Dithering,
	}
	clonePointers := []*bool{
		clone.Hints[0].EnableDXT,
		clone.Hints[0].DynRange,
		clone.Hints[0].AutoReduce,
		clone.Hints[0].VirtualSwz,
		clone.Hints[0].Dithering,
	}
	for i, sourcePointer := range sourcePointers {
		if clonePointers[i] == sourcePointer || *clonePointers[i] != *sourcePointer {
			t.Fatalf("hint bool %d was not copied independently", i)
		}
	}

	*clone.Hints[0].EnableDXT = false
	*source.Hints[0].DynRange = true
	if !*source.Hints[0].EnableDXT || *clone.Hints[0].DynRange {
		t.Fatal("mutating one config changed the other")
	}
}

func TestTexConvertConfigClonePreservesNilPointers(t *testing.T) {
	clone := (TexConvertConfig{Hints: []TextureHint{{}}}).Clone()
	hint := clone.Hints[0]
	if hint.EnableDXT != nil || hint.DynRange != nil || hint.AutoReduce != nil || hint.VirtualSwz != nil || hint.Dithering != nil {
		t.Fatalf("clone did not preserve nil pointers: %#v", hint)
	}
}

func TestDefaultTexConvertConfigReturnsIndependentClone(t *testing.T) {
	original, err := DefaultTexConvertConfig()
	if err != nil {
		t.Fatalf("DefaultTexConvertConfig() error = %v", err)
	}
	defer SetDefaultTexConvertConfig(original)

	want := TexConvertConfig{
		ConvertVersion:           9,
		ApplyDefaultErrorMetrics: true,
		UseSRGBFromDynRange:      true,
		DisableAutoReduce:        true,
		DisableLZO:               true,
		Hints: []TextureHint{{
			ClassName:  "custom",
			EnableDXT:  cloneTestBoolPointer(true),
			DynRange:   cloneTestBoolPointer(false),
			AutoReduce: cloneTestBoolPointer(true),
			VirtualSwz: cloneTestBoolPointer(false),
			Dithering:  cloneTestBoolPointer(true),
		}},
	}
	SetDefaultTexConvertConfig(want)

	first, err := DefaultTexConvertConfig()
	if err != nil {
		t.Fatalf("DefaultTexConvertConfig() after set error = %v", err)
	}
	if first.ConvertVersion != want.ConvertVersion || !first.ApplyDefaultErrorMetrics ||
		!first.UseSRGBFromDynRange || !first.DisableAutoReduce || !first.DisableLZO {
		t.Fatalf("SetDefaultTexConvertConfig lost fields: %#v", first)
	}
	*first.Hints[0].EnableDXT = false
	first.DisableLZO = false

	second, err := DefaultTexConvertConfig()
	if err != nil {
		t.Fatalf("second DefaultTexConvertConfig() error = %v", err)
	}
	if !*second.Hints[0].EnableDXT || !second.DisableLZO {
		t.Fatal("mutating returned default changed stored config")
	}
}
