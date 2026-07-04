package renderer

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg" // stdlib decoders only: no third-party image libraries
	_ "image/png"
	"os"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// LoadTexture decodes a PNG or JPEG with the standard library, flips it to
// OpenGL's bottom-up convention and uploads it with mipmaps and trilinear
// filtering. Returns the GL texture name.
func LoadTexture(path string) (uint32, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("texture: %w", err)
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return 0, fmt.Errorf("texture %s: %w", path, err)
	}

	b := src.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Src)
	flipVertical(rgba)

	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8,
		int32(rgba.Rect.Dx()), int32(rgba.Rect.Dy()), 0,
		gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba.Pix))
	gl.GenerateMipmap(gl.TEXTURE_2D)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	return tex, nil
}

func flipVertical(img *image.RGBA) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	stride := img.Stride
	tmp := make([]byte, w*4)
	for y := 0; y < h/2; y++ {
		top := img.Pix[y*stride : y*stride+w*4]
		bot := img.Pix[(h-1-y)*stride : (h-1-y)*stride+w*4]
		copy(tmp, top)
		copy(top, bot)
		copy(bot, tmp)
	}
}
