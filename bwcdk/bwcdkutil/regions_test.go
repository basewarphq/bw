package bwcdkutil_test

import (
	"testing"

	"github.com/basewarphq/bwapp/bwcdk/bwcdkutil"
)

func TestRegionForIdent_AllRegionsRoundTrip(t *testing.T) {
	for region, ident := range bwcdkutil.RegionIdents {
		got, ok := bwcdkutil.RegionForIdent(ident)
		if !ok {
			t.Errorf("RegionForIdent(%q): not found, want %q", ident, region)
			continue
		}
		if got != region {
			t.Errorf("RegionForIdent(%q) = %q, want %q", ident, got, region)
		}
	}
}

func TestRegionForIdent_Unknown(t *testing.T) {
	_, ok := bwcdkutil.RegionForIdent("Zzz9")
	if ok {
		t.Error("RegionForIdent(\"Zzz9\"): expected false, got true")
	}
}

func TestExtractRegionIdent(t *testing.T) {
	tests := []struct {
		stackName string
		want      string
	}{
		{"bwappEuw1Stag", "Euw1"},
		{"bwappUse1Prod", "Use1"},
		{"bwappEuc2Shared", "Euc2"},
		{"notastack", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := bwcdkutil.ExtractRegionIdent(tt.stackName)
		if got != tt.want {
			t.Errorf("ExtractRegionIdent(%q) = %q, want %q", tt.stackName, got, tt.want)
		}
	}
}
