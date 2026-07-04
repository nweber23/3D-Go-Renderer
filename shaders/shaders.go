// Package shaders embeds the GLSL sources so the binary is self-contained
// while the sources stay editable as plain files.
package shaders

import _ "embed"

//go:embed basic.vert
var BasicVert string

//go:embed basic.frag
var BasicFrag string

//go:embed scop.vert
var ScopVert string

//go:embed scop.frag
var ScopFrag string
