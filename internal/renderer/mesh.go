package renderer

import (
	"github.com/go-gl/gl/v4.1-core/gl"

	"scop/internal/mathgl"
	"scop/internal/objparser"
)

// GLMesh is a mesh uploaded to GPU buffers: one VAO/VBO/EBO for the whole
// model, drawn per material group via index ranges. Buffers are filled once
// at load time and never touched again.
type GLMesh struct {
	vao, vbo, ebo uint32
	Groups        []objparser.Group
	IndexCount    int32
	BBoxMin       mathgl.Vec3
	BBoxMax       mathgl.Vec3
	HasNormals    bool
}

// UploadMesh interleaves the parsed mesh as pos(3)+normal(3)+uv(2) floats
// and uploads vertices and indices with STATIC_DRAW.
func UploadMesh(m *objparser.Mesh) *GLMesh {
	data := make([]float32, 0, len(m.Vertices)*8)
	for _, v := range m.Vertices {
		data = append(data,
			v.Pos[0], v.Pos[1], v.Pos[2],
			v.Normal[0], v.Normal[1], v.Normal[2],
			v.UV[0], v.UV[1],
		)
	}

	g := &GLMesh{
		Groups:     m.Groups,
		IndexCount: int32(len(m.Indices)),
		BBoxMin:    m.BBoxMin,
		BBoxMax:    m.BBoxMax,
		HasNormals: m.HasNormals,
	}
	gl.GenVertexArrays(1, &g.vao)
	gl.GenBuffers(1, &g.vbo)
	gl.GenBuffers(1, &g.ebo)

	gl.BindVertexArray(g.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, g.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(data)*4, gl.Ptr(data), gl.STATIC_DRAW)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, g.ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(m.Indices)*4, gl.Ptr(m.Indices), gl.STATIC_DRAW)

	const stride = 8 * 4
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, stride, 0)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 3, gl.FLOAT, false, stride, 3*4)
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointerWithOffset(2, 2, gl.FLOAT, false, stride, 6*4)
	gl.BindVertexArray(0)
	return g
}

// Bind makes the mesh's VAO current.
func (g *GLMesh) Bind() { gl.BindVertexArray(g.vao) }

// DrawAll issues one indexed draw covering every group.
func (g *GLMesh) DrawAll() {
	gl.DrawElementsWithOffset(gl.TRIANGLES, g.IndexCount, gl.UNSIGNED_INT, 0)
}

// Center returns the bounding-box center: the model's rotation pivot.
func (g *GLMesh) Center() mathgl.Vec3 {
	return g.BBoxMin.Add(g.BBoxMax).Scale(0.5)
}

// MaxExtent returns the largest bounding-box dimension, used to place the
// default camera at a distance that frames any model.
func (g *GLMesh) MaxExtent() float32 {
	e := g.BBoxMax.Sub(g.BBoxMin)
	m := e[0]
	if e[1] > m {
		m = e[1]
	}
	if e[2] > m {
		m = e[2]
	}
	return m
}
