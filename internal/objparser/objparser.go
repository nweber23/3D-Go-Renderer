// Package objparser loads Wavefront .obj and .mtl files from scratch:
// vertices, texture coordinates, normals, polygonal faces (triangulated),
// materials, and automatic UV generation for meshes without vt data.
package objparser

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"scop/internal/mathgl"
)

// Vertex is one deduplicated vertex as uploaded to the GPU.
type Vertex struct {
	Pos    mathgl.Vec3
	Normal mathgl.Vec3
	UV     mathgl.Vec2
}

// Material holds the .mtl fields the renderer consumes.
type Material struct {
	Name       string
	Ka, Kd, Ks mathgl.Vec3
	Ns         float32
	MapKd      string // path resolved relative to the .mtl directory; "" if none
}

// Group is a contiguous index range drawn with a single material.
type Group struct {
	Material    Material
	IndexOffset int
	IndexCount  int
}

// Mesh is the parser output: deduplicated interleaved vertices with
// triangle indices, grouped by material.
type Mesh struct {
	Vertices         []Vertex
	Indices          []uint32
	Groups           []Group
	BBoxMin, BBoxMax mathgl.Vec3
	HasNormals       bool
	HasUVs           bool
}

// Center returns the bounding-box center: the model's rotation pivot.
func (m *Mesh) Center() mathgl.Vec3 {
	return m.BBoxMin.Add(m.BBoxMax).Scale(0.5)
}

// ParseOBJ reads and parses the .obj file at path.
func ParseOBJ(path string) (*Mesh, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("objparser: %w", err)
	}
	dir := path
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		dir = path[:i]
	} else {
		dir = "."
	}
	return parseOBJBytes(data, dir)
}

// builder accumulates parser state for one .obj file.
type builder struct {
	positions []mathgl.Vec3
	uvs       []mathgl.Vec2
	normals   []mathgl.Vec3

	mesh  Mesh
	dedup map[[3]int32]uint32

	poly      []uint32      // corner indices of the face being assembled
	cornerBuf []mathgl.Vec3 // scratch corner positions for triangulation
	warnings  int
}

func warn(line int, format string, args ...any) {
	fmt.Fprintf(os.Stderr, "objparser: line %d: "+format+"\n", append([]any{line}, args...)...)
}

func parseOBJBytes(data []byte, dir string) (*Mesh, error) {
	b := &builder{dedup: make(map[[3]int32]uint32)}

	lineNo := 0
	for len(data) > 0 {
		lineNo++
		var line string
		if i := indexByte(data, '\n'); i >= 0 {
			line = string(data[:i])
			data = data[i+1:]
		} else {
			line = string(data)
			data = nil
		}
		if err := b.parseLine(strings.TrimSpace(line), lineNo, dir); err != nil {
			return nil, err
		}
	}

	if len(b.mesh.Indices) == 0 {
		return nil, fmt.Errorf("objparser: no renderable faces found")
	}
	b.finishGroups()
	b.computeBBox()
	return &b.mesh, nil
}

func indexByte(s []byte, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func (b *builder) parseLine(line string, lineNo int, dir string) error {
	if line == "" || line[0] == '#' {
		return nil
	}
	fields := strings.Fields(line)
	switch fields[0] {
	case "v":
		if v, ok := parseVec3(fields[1:]); ok {
			b.positions = append(b.positions, v)
		} else {
			b.warnings++
			warn(lineNo, "malformed vertex %q", line)
		}
	case "vt":
		if v, ok := parseVec2(fields[1:]); ok {
			b.uvs = append(b.uvs, v)
		} else {
			b.warnings++
			warn(lineNo, "malformed texcoord %q", line)
		}
	case "vn":
		if v, ok := parseVec3(fields[1:]); ok {
			b.normals = append(b.normals, v)
		} else {
			b.warnings++
			warn(lineNo, "malformed normal %q", line)
		}
	case "f":
		return b.parseFace(fields[1:], lineNo)
	case "o", "g", "s", "mtllib", "usemtl":
		// object/group/smoothing names are irrelevant to rendering;
		// material directives are handled by the mtl support layer.
	default:
		b.warnings++
		warn(lineNo, "unknown directive %q", fields[0])
	}
	return nil
}

func parseVec3(fields []string) (mathgl.Vec3, bool) {
	if len(fields) < 3 {
		return mathgl.Vec3{}, false
	}
	var v mathgl.Vec3
	for i := 0; i < 3; i++ {
		f, err := strconv.ParseFloat(fields[i], 32)
		if err != nil {
			return mathgl.Vec3{}, false
		}
		v[i] = float32(f)
	}
	return v, true
}

func parseVec2(fields []string) (mathgl.Vec2, bool) {
	if len(fields) < 2 {
		return mathgl.Vec2{}, false
	}
	var v mathgl.Vec2
	for i := 0; i < 2; i++ {
		f, err := strconv.ParseFloat(fields[i], 32)
		if err != nil {
			return mathgl.Vec2{}, false
		}
		v[i] = float32(f)
	}
	return v, true
}

// parseFace resolves corner references, deduplicates vertices and appends
// the triangulated polygon to the index buffer.
func (b *builder) parseFace(corners []string, lineNo int) error {
	if len(corners) < 3 {
		b.warnings++
		warn(lineNo, "face with %d corners skipped", len(corners))
		return nil
	}
	b.poly = b.poly[:0]
	for _, c := range corners {
		vi, ti, ni, err := b.resolveCorner(c, lineNo)
		if err != nil {
			return err
		}
		b.poly = append(b.poly, b.vertexIndex(vi, ti, ni))
	}
	b.emitPolygon(b.poly)
	return nil
}

// resolveCorner parses "v", "v/vt", "v//vn" or "v/vt/vn" into 0-based
// indices; -1 marks an absent component. Negative indices are relative to
// the counts parsed so far, per the .obj spec.
func (b *builder) resolveCorner(tok string, lineNo int) (vi, ti, ni int32, err error) {
	ti, ni = -1, -1
	part := tok
	rest := ""
	if i := strings.IndexByte(tok, '/'); i >= 0 {
		part, rest = tok[:i], tok[i+1:]
	}
	vi, err = resolveIndex(part, len(b.positions), lineNo, "vertex")
	if err != nil {
		return
	}
	if rest != "" {
		part = rest
		rest = ""
		if i := strings.IndexByte(part, '/'); i >= 0 {
			part, rest = part[:i], part[i+1:]
		}
		if part != "" {
			ti, err = resolveIndex(part, len(b.uvs), lineNo, "texcoord")
			if err != nil {
				return
			}
		}
		if rest != "" {
			ni, err = resolveIndex(rest, len(b.normals), lineNo, "normal")
			if err != nil {
				return
			}
		}
	}
	return
}

func resolveIndex(s string, count, lineNo int, kind string) (int32, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("objparser: line %d: bad %s index %q", lineNo, kind, s)
	}
	switch {
	case n > 0 && n <= count:
		return int32(n - 1), nil
	case n < 0 && -n <= count:
		return int32(count + n), nil
	default:
		return 0, fmt.Errorf("objparser: line %d: %s index %d out of range (have %d)", lineNo, kind, n, count)
	}
}

// vertexIndex returns the mesh index for the (v,vt,vn) triple, adding a new
// deduplicated vertex on first sight.
func (b *builder) vertexIndex(vi, ti, ni int32) uint32 {
	key := [3]int32{vi, ti, ni}
	if idx, ok := b.dedup[key]; ok {
		return idx
	}
	v := Vertex{Pos: b.positions[vi]}
	if ti >= 0 {
		v.UV = b.uvs[ti]
		b.mesh.HasUVs = true
	}
	if ni >= 0 {
		v.Normal = b.normals[ni]
		b.mesh.HasNormals = true
	}
	idx := uint32(len(b.mesh.Vertices))
	b.mesh.Vertices = append(b.mesh.Vertices, v)
	b.dedup[key] = idx
	return idx
}

// emitPolygon triangulates the polygon (mesh-space corner indices) and
// appends the resulting triangles.
func (b *builder) emitPolygon(poly []uint32) {
	if len(poly) == 3 {
		b.mesh.Indices = append(b.mesh.Indices, poly[0], poly[1], poly[2])
		return
	}
	b.cornerBuf = b.cornerBuf[:0]
	for _, idx := range poly {
		b.cornerBuf = append(b.cornerBuf, b.mesh.Vertices[idx].Pos)
	}
	b.mesh.Indices = triangulate(b.cornerBuf, poly, b.mesh.Indices)
}

// finishGroups wraps all indices in a single default-material group; the
// mtl layer replaces this with real per-material groups.
func (b *builder) finishGroups() {
	if len(b.mesh.Groups) == 0 {
		b.mesh.Groups = []Group{{
			Material:    DefaultMaterial(),
			IndexOffset: 0,
			IndexCount:  len(b.mesh.Indices),
		}}
	}
}

// DefaultMaterial is used when an .obj has no usable .mtl reference.
func DefaultMaterial() Material {
	return Material{
		Name: "default",
		Kd:   mathgl.Vec3{0.64, 0.64, 0.64},
		Ks:   mathgl.Vec3{0.5, 0.5, 0.5},
		Ns:   32,
	}
}

func (b *builder) computeBBox() {
	if len(b.mesh.Vertices) == 0 {
		return
	}
	min := b.mesh.Vertices[0].Pos
	max := min
	for _, v := range b.mesh.Vertices[1:] {
		for i := 0; i < 3; i++ {
			if v.Pos[i] < min[i] {
				min[i] = v.Pos[i]
			}
			if v.Pos[i] > max[i] {
				max[i] = v.Pos[i]
			}
		}
	}
	b.mesh.BBoxMin, b.mesh.BBoxMax = min, max
}
