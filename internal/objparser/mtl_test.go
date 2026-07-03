package objparser

import (
	"os"
	"testing"

	"scop/internal/mathgl"
)

func TestParseMTL42(t *testing.T) {
	data, err := os.ReadFile("../../models/42.mtl")
	if err != nil {
		t.Fatal(err)
	}
	mats, err := parseMTL(data, "../../models")
	if err != nil {
		t.Fatal(err)
	}
	m, ok := mats["Material"]
	if !ok {
		t.Fatalf("material %q not found; have %v", "Material", mats)
	}
	if !nearEqF(m.Ns, 96.078431) {
		t.Errorf("Ns = %v, want 96.078431", m.Ns)
	}
	if m.Kd != (mathgl.Vec3{0.64, 0.64, 0.64}) {
		t.Errorf("Kd = %v", m.Kd)
	}
	if m.Ks != (mathgl.Vec3{0.5, 0.5, 0.5}) {
		t.Errorf("Ks = %v", m.Ks)
	}
	if m.Ka != (mathgl.Vec3{0, 0, 0}) {
		t.Errorf("Ka = %v", m.Ka)
	}
	if m.MapKd != "" {
		t.Errorf("MapKd = %q, want empty", m.MapKd)
	}
}

func nearEqF(a, b float32) bool {
	d := a - b
	return d < 1e-4 && d > -1e-4
}

func TestParseMultiMaterial(t *testing.T) {
	m, err := ParseOBJ("testdata/multi.obj")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Groups) != 2 {
		t.Fatalf("groups = %d, want 2", len(m.Groups))
	}
	total := 0
	byName := map[string]Group{}
	for _, g := range m.Groups {
		total += g.IndexCount
		byName[g.Material.Name] = g
	}
	if total != len(m.Indices) {
		t.Fatalf("group counts sum %d != %d indices", total, len(m.Indices))
	}
	red, ok := byName["red"]
	if !ok {
		t.Fatalf("no red group: %v", byName)
	}
	if red.IndexCount != 6 { // two triangles use red
		t.Errorf("red count = %d, want 6", red.IndexCount)
	}
	if red.Material.Kd != (mathgl.Vec3{1, 0, 0}) {
		t.Errorf("red Kd = %v", red.Material.Kd)
	}
	if red.Material.MapKd != "testdata/tex.png" {
		t.Errorf("red MapKd = %q, want testdata/tex.png", red.Material.MapKd)
	}
	blue := byName["blue"]
	if blue.IndexCount != 3 {
		t.Errorf("blue count = %d, want 3", blue.IndexCount)
	}
	if blue.Material.MapKd != "" {
		t.Errorf("blue MapKd = %q", blue.Material.MapKd)
	}
	// Groups are sorted by texture then name so draws batch state changes:
	// "" (blue, untextured) sorts before "testdata/tex.png" (red).
	if m.Groups[0].Material.Name != "blue" || m.Groups[1].Material.Name != "red" {
		t.Errorf("group order = %q, %q; want blue, red",
			m.Groups[0].Material.Name, m.Groups[1].Material.Name)
	}
	// Index ranges must be contiguous and non-overlapping.
	if m.Groups[0].IndexOffset != 0 || m.Groups[1].IndexOffset != m.Groups[0].IndexCount {
		t.Errorf("offsets: %d then %d", m.Groups[0].IndexOffset, m.Groups[1].IndexOffset)
	}
}

func TestParseMissingMTLFallsBack(t *testing.T) {
	m, err := ParseOBJ("testdata/missingmtl.obj")
	if err != nil {
		t.Fatalf("missing mtl must not be fatal: %v", err)
	}
	if len(m.Groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(m.Groups))
	}
	g := m.Groups[0]
	if g.Material.Kd != DefaultMaterial().Kd {
		t.Errorf("fallback Kd = %v", g.Material.Kd)
	}
	if g.IndexCount != 3 || g.IndexOffset != 0 {
		t.Errorf("group range = %d+%d", g.IndexOffset, g.IndexCount)
	}
}
