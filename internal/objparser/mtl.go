package objparser

import (
	"fmt"
	"strconv"
	"strings"
)

// parseMTL parses a .mtl file. Texture paths (map_Kd) are resolved relative
// to dir. Unknown directives are ignored: only Ka/Kd/Ks/Ns/map_Kd matter to
// the renderer.
func parseMTL(data []byte, dir string) (map[string]Material, error) {
	mats := make(map[string]Material)
	var cur *Material

	flush := func() {
		if cur != nil {
			mats[cur.Name] = *cur
		}
	}

	lineNo := 0
	for _, raw := range strings.Split(string(data), "\n") {
		lineNo++
		line := strings.TrimSpace(raw)
		if line == "" || line[0] == '#' {
			continue
		}
		fields := strings.Fields(line)
		if fields[0] == "newmtl" {
			if len(fields) < 2 {
				warn(lineNo, "newmtl without a name")
				continue
			}
			flush()
			m := DefaultMaterial()
			m.Name = fields[1]
			m.MapKd = ""
			cur = &m
			continue
		}
		if cur == nil {
			continue // directives before the first newmtl
		}
		switch fields[0] {
		case "Ka":
			if v, ok := parseVec3(fields[1:]); ok {
				cur.Ka = v
			} else {
				warn(lineNo, "malformed Ka %q", line)
			}
		case "Kd":
			if v, ok := parseVec3(fields[1:]); ok {
				cur.Kd = v
			} else {
				warn(lineNo, "malformed Kd %q", line)
			}
		case "Ks":
			if v, ok := parseVec3(fields[1:]); ok {
				cur.Ks = v
			} else {
				warn(lineNo, "malformed Ks %q", line)
			}
		case "Ns":
			if f, err := strconv.ParseFloat(fields[1], 32); err == nil {
				cur.Ns = float32(f)
			} else {
				warn(lineNo, "malformed Ns %q", line)
			}
		case "map_Kd":
			if len(fields) >= 2 {
				// The texture file is the last field; options are unsupported.
				cur.MapKd = dir + "/" + fields[len(fields)-1]
			}
		}
	}
	flush()
	if len(mats) == 0 {
		return nil, fmt.Errorf("objparser: mtl defines no materials")
	}
	return mats, nil
}
