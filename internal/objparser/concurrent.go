package objparser

import (
	"runtime"
	"strconv"
	"strings"
	"sync"

	"scop/internal/mathgl"
)

// The concurrent parser splits the file into line-aligned chunks and lets
// one goroutine per chunk convert the float-heavy v/vt/vn lines into typed
// events. A sequential replay pass then walks the chunks in file order —
// faces, materials and negative-index resolution are position-dependent and
// must observe running counts — so the output is byte-for-byte identical to
// a sequential parse (verified against parseOBJBytesSequential in tests).

const minChunkSize = 64 << 10

// event is one parsed line: a pre-parsed vertex component or face record,
// or a raw line deferred to the sequential pass.
type event struct {
	kind   byte // 'v', 't', 'n', 'f', or 0 for a raw line
	v3     mathgl.Vec3
	v2     mathgl.Vec2
	face   [][3]int32 // raw (v,vt,vn) ints; absentIdx marks a missing part
	line   string
	lineNo int // chunk-local; offset by the chunk's start line at replay
}

// absentIdx marks a face corner component that was not written (e.g. the
// vt slot in "v//vn"). Raw indices are resolved against running counts
// during replay, exactly like the sequential parser.
const absentIdx = int32(-1 << 31)

// parseCornerRaw converts one face corner token into raw integers without
// resolving them. Reports false for malformed tokens so the whole line can
// fall back to the sequential error path (identical messages).
func parseCornerRaw(tok string) ([3]int32, bool) {
	out := [3]int32{0, absentIdx, absentIdx}
	part := 0
	start := 0
	for i := 0; i <= len(tok); i++ {
		if i < len(tok) && tok[i] != '/' {
			continue
		}
		if part > 2 {
			return out, false
		}
		if s := tok[start:i]; s != "" {
			n, err := strconv.Atoi(s)
			if err != nil || n == 0 {
				return out, false
			}
			out[part] = int32(n)
		} else if part == 0 {
			return out, false // the vertex index is mandatory
		}
		part++
		start = i + 1
	}
	return out, true
}

type chunkResult struct {
	events    []event
	lineCount int
}

func parseOBJBytes(data []byte, dir string) (*Mesh, error) {
	chunks := splitLines(data, runtime.GOMAXPROCS(0))
	if len(chunks) <= 1 {
		return parseOBJBytesSequential(data, dir)
	}

	results := make([]chunkResult, len(chunks))
	var wg sync.WaitGroup
	for i, chunk := range chunks {
		wg.Add(1)
		go func(i int, chunk []byte) {
			defer wg.Done()
			results[i] = scanChunk(chunk)
		}(i, chunk)
	}
	wg.Wait()

	// Sequential replay in file order keeps every position-dependent rule
	// (negative indices, usemtl scope, warning line numbers) intact.
	b := newBuilder()
	startLine := 0
	for _, res := range results {
		for i := range res.events {
			ev := &res.events[i]
			if err := b.replay(ev, startLine, dir); err != nil {
				return nil, err
			}
		}
		startLine += res.lineCount
	}
	return b.finish()
}

// parseOBJBytesSequential is the plain line-by-line parser; the concurrent
// path must produce identical output.
func parseOBJBytesSequential(data []byte, dir string) (*Mesh, error) {
	b := newBuilder()
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
	return b.finish()
}

// splitLines cuts data into up to n chunks ending on line boundaries.
func splitLines(data []byte, n int) [][]byte {
	if n < 1 {
		n = 1
	}
	target := len(data) / n
	if target < minChunkSize {
		target = minChunkSize
	}
	var chunks [][]byte
	for len(data) > 0 {
		if len(data) <= target {
			chunks = append(chunks, data)
			break
		}
		cut := target
		for cut < len(data) && data[cut] != '\n' {
			cut++
		}
		if cut < len(data) {
			cut++ // include the newline
		}
		chunks = append(chunks, data[:cut])
		data = data[cut:]
	}
	return chunks
}

// scanChunk parses one chunk's v/vt/vn lines into typed events and defers
// everything else as raw lines.
func scanChunk(data []byte) chunkResult {
	var res chunkResult
	for len(data) > 0 {
		res.lineCount++
		var raw []byte
		if i := indexByte(data, '\n'); i >= 0 {
			raw = data[:i]
			data = data[i+1:]
		} else {
			raw = data
			data = nil
		}
		line := strings.TrimSpace(string(raw))
		if line == "" || line[0] == '#' {
			continue
		}
		fields := strings.Fields(line)
		ev := event{lineNo: res.lineCount, line: line}
		switch fields[0] {
		case "v":
			if v, ok := parseVec3(fields[1:]); ok {
				ev.kind, ev.v3 = 'v', v
			}
		case "vt":
			if v, ok := parseVec2(fields[1:]); ok {
				ev.kind, ev.v2 = 't', v
			}
		case "vn":
			if v, ok := parseVec3(fields[1:]); ok {
				ev.kind, ev.v3 = 'n', v
			}
		case "f":
			if len(fields) >= 4 {
				face := make([][3]int32, 0, len(fields)-1)
				ok := true
				for _, tok := range fields[1:] {
					c, good := parseCornerRaw(tok)
					if !good {
						ok = false
						break
					}
					face = append(face, c)
				}
				if ok {
					ev.kind, ev.face = 'f', face
				}
			}
		}
		res.events = append(res.events, ev)
	}
	return res
}

// replay applies one event to the builder. Pre-parsed vertex data is
// appended directly; raw lines (faces, materials, malformed input) go
// through the normal sequential path.
func (b *builder) replay(ev *event, startLine int, dir string) error {
	switch ev.kind {
	case 'v':
		b.positions = append(b.positions, ev.v3)
	case 't':
		b.uvs = append(b.uvs, ev.v2)
	case 'n':
		b.normals = append(b.normals, ev.v3)
	case 'f':
		return b.emitFaceRaw(ev.face, startLine+ev.lineNo)
	default:
		return b.parseLine(ev.line, startLine+ev.lineNo, dir)
	}
	return nil
}

// emitFaceRaw resolves a pre-tokenized face against the running counts and
// emits it, mirroring parseFace.
func (b *builder) emitFaceRaw(face [][3]int32, lineNo int) error {
	b.poly = b.poly[:0]
	for _, c := range face {
		vi, err := resolveRaw(int(c[0]), len(b.positions), lineNo, "vertex")
		if err != nil {
			return err
		}
		ti, ni := int32(-1), int32(-1)
		if c[1] != absentIdx {
			if ti, err = resolveRaw(int(c[1]), len(b.uvs), lineNo, "texcoord"); err != nil {
				return err
			}
		}
		if c[2] != absentIdx {
			if ni, err = resolveRaw(int(c[2]), len(b.normals), lineNo, "normal"); err != nil {
				return err
			}
		}
		b.poly = append(b.poly, b.vertexIndex(vi, ti, ni))
	}
	b.emitPolygon(b.poly)
	return nil
}
