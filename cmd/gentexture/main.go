// Command gentexture writes the default texture shipped with scop
// (models/default_texture.png): a procedural 512x512 pattern — a diagonal
// rainbow gradient with a contrasting grid and circles so orientation,
// tiling and stretching are visible on any model. The original assignment
// texture was never supplied, so this generated one stands in; any PNG or
// JPEG can be substituted at runtime with -texture.
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
)

const size = 512

func main() {
	out := "models/default_texture.png"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			// Diagonal rainbow base.
			hue := math.Mod(float64(x+y)/float64(2*size)*360+200, 360)
			r, g, b := hsv(hue, 0.55, 0.85)

			// Grid every 64 px, brighter major lines every 128 px.
			if x%64 == 0 || y%64 == 0 {
				r, g, b = dim(r, g, b, 0.55)
			}
			if x%128 == 0 || y%128 == 0 {
				r, g, b = dim(r, g, b, 0.7)
			}

			// Concentric circles from the center reveal distortion.
			dx, dy := float64(x-size/2), float64(y-size/2)
			d := math.Sqrt(dx*dx + dy*dy)
			if math.Mod(d, 96) < 3 {
				r, g, b = 0.95, 0.95, 0.95
			}

			img.SetRGBA(x, y, color.RGBA{
				uint8(r * 255), uint8(g * 255), uint8(b * 255), 255,
			})
		}
	}

	f, err := os.Create(out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s", out)
}

func hsv(h, s, v float64) (float64, float64, float64) {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return r + m, g + m, b + m
}

func dim(r, g, b, f float64) (float64, float64, float64) {
	return r * f, g * f, b * f
}
