package renderer

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// shotBuf is reused across screenshots so pressing F12 repeatedly does not
// re-allocate the framebuffer copy.
var shotBuf []byte

// Screenshot reads the current framebuffer and writes it to a timestamped
// PNG (scop_20060102_150405.png) in the working directory.
func Screenshot(fbW, fbH int) (string, error) {
	n := fbW * fbH * 4
	if cap(shotBuf) < n {
		shotBuf = make([]byte, n)
	}
	buf := shotBuf[:n]

	gl.PixelStorei(gl.PACK_ALIGNMENT, 1)
	gl.ReadPixels(0, 0, int32(fbW), int32(fbH), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(buf))

	// GL rows run bottom-up; image rows run top-down.
	img := image.NewNRGBA(image.Rect(0, 0, fbW, fbH))
	rowLen := fbW * 4
	for y := 0; y < fbH; y++ {
		src := buf[(fbH-1-y)*rowLen : (fbH-y)*rowLen]
		copy(img.Pix[y*img.Stride:y*img.Stride+rowLen], src)
	}
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255 // discard alpha: the framebuffer is opaque
	}

	name := fmt.Sprintf("scop_%s.png", time.Now().Format("20060102_150405"))
	f, err := os.Create(name)
	if err != nil {
		return "", fmt.Errorf("screenshot: %w", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return "", fmt.Errorf("screenshot: %w", err)
	}
	return name, nil
}
