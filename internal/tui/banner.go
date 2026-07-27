package tui

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	_ "image/jpeg"
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
)

// goatJPEG is generated art (Nano Banana), committed here rather than read
// from a personal path — the prototype it came from
// (proto/start-screen-banner) used an absolute Downloads path, which only
// works on one machine; embedding is what makes it ship in the real binary.
//
//go:embed assets/goat.jpg
var goatJPEG []byte

var (
	goatIconOnce sync.Once
	goatIconArt  []string
)

// goatIcon is the picker's banner icon: half-block terminal pixel art,
// rendered once and cached — this runs on every return to the picker, and
// the source image never changes at runtime.
func goatIcon() []string {
	goatIconOnce.Do(func() {
		goatIconArt = renderPixelArt(goatJPEG, 18, 13)
	})
	return goatIconArt
}

// renderPixelArt renders an image as terminal pixel art: each character row
// samples two source pixel-rows (a "top" and a "bottom"), drawn with the
// half-block character so a monospace terminal — whose cells are roughly
// twice as tall as wide — ends up with roughly square pixels rather than
// vertically squashed ones.
//
// The source is a JPEG, which carries no alpha channel, unlike the PNG the
// prototype was built against — a near-white pixel is treated as background
// instead. Skipping it (rather than painting it white) is what keeps the
// icon from carrying a visible box around it on a dark terminal.
func renderPixelArt(data []byte, cols, rows int) []string {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return []string{fmt.Sprintf("(could not decode embedded image: %s)", err)}
	}

	b := img.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	pixelRows := rows * 2

	const whiteThreshold = 0xf000 // ~94% of 0xffff per channel

	// sample reports a pixel-grid cell's colour, and whether it's "ink"
	// (not near-white background) worth drawing at all.
	sample := func(col, prow int) (hex string, ink bool) {
		x0, x1 := b.Min.X+col*srcW/cols, b.Min.X+(col+1)*srcW/cols
		y0, y1 := b.Min.Y+prow*srcH/pixelRows, b.Min.Y+(prow+1)*srcH/pixelRows
		if x1 <= x0 {
			x1 = x0 + 1
		}
		if y1 <= y0 {
			y1 = y0 + 1
		}
		var rs, gs, bs, n uint64
		for y := y0; y < y1 && y < b.Max.Y; y++ {
			for x := x0; x < x1 && x < b.Max.X; x++ {
				r, g, bl, _ := img.At(x, y).RGBA()
				rs, gs, bs = rs+uint64(r), gs+uint64(g), bs+uint64(bl)
				n++
			}
		}
		if n == 0 {
			return "", false
		}
		r, g, bl := rs/n, gs/n, bs/n
		if r >= whiteThreshold && g >= whiteThreshold && bl >= whiteThreshold {
			return "", false
		}
		return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, bl>>8), true
	}

	out := make([]string, rows)
	for r := range rows {
		var line strings.Builder
		for c := range cols {
			topHex, topInk := sample(c, r*2)
			botHex, botInk := sample(c, r*2+1)
			switch {
			case topInk && botInk:
				style := lipgloss.NewStyle().Foreground(lipgloss.Color(topHex)).Background(lipgloss.Color(botHex))
				line.WriteString(style.Render("▀"))
			case topInk:
				line.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(topHex)).Render("▀"))
			case botInk:
				line.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(botHex)).Render("▄"))
			default:
				line.WriteString(" ")
			}
		}
		out[r] = line.String()
	}
	return out
}

// padVisible pads by *visible* width (lipgloss.Width, which discounts ANSI
// escape codes) rather than rune count — icon lines carry real colour codes,
// so naive rune-length padding would misjudge how much space is left and
// throw the wordmark column next to it out of alignment.
func padVisible(s string, width int) string {
	if w := lipgloss.Width(s); w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return s
}
