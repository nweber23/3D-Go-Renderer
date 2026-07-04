package renderer

// A hand-drawn 8x8 pixel font covering exactly the characters the FPS
// overlay prints. Each glyph is eight rows of eight cells; '#' marks a lit
// pixel. Drawn as source-code ASCII art so the data is reviewable.
var glyphArt = map[byte][8]string{
	'0': {
		"  ####  ",
		" ##  ## ",
		" ## ### ",
		" ###### ",
		" ### ## ",
		" ##  ## ",
		"  ####  ",
		"        ",
	},
	'1': {
		"   ##   ",
		"  ###   ",
		"   ##   ",
		"   ##   ",
		"   ##   ",
		"   ##   ",
		" ###### ",
		"        ",
	},
	'2': {
		"  ####  ",
		" ##  ## ",
		"     ## ",
		"   ###  ",
		"  ##    ",
		" ##     ",
		" ###### ",
		"        ",
	},
	'3': {
		"  ####  ",
		" ##  ## ",
		"     ## ",
		"   ###  ",
		"     ## ",
		" ##  ## ",
		"  ####  ",
		"        ",
	},
	'4': {
		"    ##  ",
		"   ###  ",
		"  # ##  ",
		" #  ##  ",
		" ###### ",
		"    ##  ",
		"    ##  ",
		"        ",
	},
	'5': {
		" ###### ",
		" ##     ",
		" #####  ",
		"     ## ",
		"     ## ",
		" ##  ## ",
		"  ####  ",
		"        ",
	},
	'6': {
		"  ####  ",
		" ##     ",
		" #####  ",
		" ##  ## ",
		" ##  ## ",
		" ##  ## ",
		"  ####  ",
		"        ",
	},
	'7': {
		" ###### ",
		"     ## ",
		"    ##  ",
		"   ##   ",
		"  ##    ",
		"  ##    ",
		"  ##    ",
		"        ",
	},
	'8': {
		"  ####  ",
		" ##  ## ",
		" ##  ## ",
		"  ####  ",
		" ##  ## ",
		" ##  ## ",
		"  ####  ",
		"        ",
	},
	'9': {
		"  ####  ",
		" ##  ## ",
		" ##  ## ",
		"  ##### ",
		"     ## ",
		"    ##  ",
		"  ###   ",
		"        ",
	},
	'F': {
		" ###### ",
		" ##     ",
		" ##     ",
		" #####  ",
		" ##     ",
		" ##     ",
		" ##     ",
		"        ",
	},
	'P': {
		" #####  ",
		" ##  ## ",
		" ##  ## ",
		" #####  ",
		" ##     ",
		" ##     ",
		" ##     ",
		"        ",
	},
	'S': {
		"  ##### ",
		" ##     ",
		" ##     ",
		"  ####  ",
		"     ## ",
		"     ## ",
		" #####  ",
		"        ",
	},
	'm': {
		"        ",
		"        ",
		" ## # # ",
		" # # # #",
		" # # # #",
		" # # # #",
		" # # # #",
		"        ",
	},
	's': {
		"        ",
		"        ",
		"  ##### ",
		" ##     ",
		"  ####  ",
		"     ## ",
		" #####  ",
		"        ",
	},
	'.': {
		"        ",
		"        ",
		"        ",
		"        ",
		"        ",
		"  ##    ",
		"  ##    ",
		"        ",
	},
	'/': {
		"      ##",
		"     ## ",
		"    ##  ",
		"   ##   ",
		"  ##    ",
		" ##     ",
		"##      ",
		"        ",
	},
	' ': {
		"        ",
		"        ",
		"        ",
		"        ",
		"        ",
		"        ",
		"        ",
		"        ",
	},
}

// glyphOrder fixes each character's column in the atlas texture.
var glyphOrder = []byte("0123456789FPSms./ ")

// glyphColumn maps a character to its atlas column; unknown characters
// render as a space.
func glyphColumn(c byte) int {
	for i, g := range glyphOrder {
		if g == c {
			return i
		}
	}
	return len(glyphOrder) - 1
}

// buildFontAtlas rasterizes the glyphs into a single-row alpha bitmap
// (8*len(glyphOrder) x 8, one byte per pixel).
func buildFontAtlas() []byte {
	w := 8 * len(glyphOrder)
	pix := make([]byte, w*8)
	for col, c := range glyphOrder {
		art := glyphArt[c]
		for y := 0; y < 8; y++ {
			row := art[y]
			for x := 0; x < 8 && x < len(row); x++ {
				if row[x] == '#' {
					pix[y*w+col*8+x] = 255
				}
			}
		}
	}
	return pix
}
