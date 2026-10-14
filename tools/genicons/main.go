// Command genicons draws the Manus AI app icons (web PWA + Android launcher)
// with the standard library only, so the repository needs no binary assets
// checked in by hand.
//
// Usage:
//
//	go run ./tools/genicons
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

const supersample = 4

type target struct {
	path   string
	size   int
	mask   bool // true = maskable (extra safe-zone padding)
	radius float64
}

func main() {
	targets := []target{
		{path: "web/static/icons/icon-192.png", size: 192, radius: 0.22},
		{path: "web/static/icons/icon-512.png", size: 512, radius: 0.22},
		{path: "web/static/icons/maskable-512.png", size: 512, radius: 0.0, mask: true},
		{path: "web/static/icons/apple-touch-icon.png", size: 180, radius: 0.0},
		{path: "web/static/favicon.png", size: 32, radius: 0.22},
		{path: "android/app/src/main/res/mipmap-mdpi/ic_launcher.png", size: 48, radius: 0.22},
		{path: "android/app/src/main/res/mipmap-hdpi/ic_launcher.png", size: 72, radius: 0.22},
		{path: "android/app/src/main/res/mipmap-xhdpi/ic_launcher.png", size: 96, radius: 0.22},
		{path: "android/app/src/main/res/mipmap-xxhdpi/ic_launcher.png", size: 144, radius: 0.22},
		{path: "android/app/src/main/res/mipmap-xxxhdpi/ic_launcher.png", size: 192, radius: 0.22},
		{path: "android/app/src/main/res/mipmap-mdpi/ic_launcher_round.png", size: 48, radius: 0.5},
		{path: "android/app/src/main/res/mipmap-hdpi/ic_launcher_round.png", size: 72, radius: 0.5},
		{path: "android/app/src/main/res/mipmap-xhdpi/ic_launcher_round.png", size: 96, radius: 0.5},
		{path: "android/app/src/main/res/mipmap-xxhdpi/ic_launcher_round.png", size: 144, radius: 0.5},
		{path: "android/app/src/main/res/mipmap-xxxhdpi/ic_launcher_round.png", size: 192, radius: 0.5},
	}

	for _, item := range targets {
		if err := writeIcon(item); err != nil {
			fmt.Fprintf(os.Stderr, "genicons: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s (%dx%d)\n", item.path, item.size, item.size)
	}
}

func writeIcon(item target) error {
	img := render(item.size, item.mask, item.radius)
	if err := os.MkdirAll(filepath.Dir(item.path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(item.path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	return png.Encode(file, img)
}

// render draws the icon at the requested size using 4x supersampling.
func render(size int, maskable bool, radiusRatio float64) *image.RGBA {
	big := size * supersample
	canvas := image.NewRGBA(image.Rect(0, 0, big, big))

	center := float64(big) / 2
	corner := radiusRatio * float64(big) / 2
	glyph := float64(big) * 0.30 // glyph radius
	ring := float64(big) * 0.022
	if maskable {
		glyph = float64(big) * 0.22
	}

	for y := 0; y < big; y++ {
		for x := 0; x < big; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5

			// Background: rounded square (or full square for maskable icons).
			if !maskable && !insideRoundedRect(fx, fy, float64(big), corner) {
				continue
			}

			base := gradient(fx, fy, float64(big))
			canvas.Set(x, y, base)

			// Glyph: white ring plus a play triangle.
			distance := math.Hypot(fx-center, fy-center)
			if math.Abs(distance-glyph) < ring {
				canvas.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
				continue
			}
			if insideTriangle(fx, fy, center, center, glyph*0.62) {
				canvas.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
			}
		}
	}

	return downsample(canvas, size)
}

// gradient blends two brand colours diagonally across the icon.
func gradient(x, y, size float64) color.RGBA {
	ratio := (x + y) / (2 * size)
	from := [3]float64{0x5b, 0x6c, 0xff}
	to := [3]float64{0xa0, 0x6b, 0xff}
	return color.RGBA{
		R: uint8(from[0] + (to[0]-from[0])*ratio),
		G: uint8(from[1] + (to[1]-from[1])*ratio),
		B: uint8(from[2] + (to[2]-from[2])*ratio),
		A: 255,
	}
}

func insideRoundedRect(x, y, size, radius float64) bool {
	if radius <= 0 {
		return true
	}
	cx := math.Min(math.Max(x, radius), size-radius)
	cy := math.Min(math.Max(y, radius), size-radius)
	return math.Hypot(x-cx, y-cy) <= radius
}

// insideTriangle reports whether the point is inside a right-pointing
// triangle centred on (cx, cy).
func insideTriangle(x, y, cx, cy, size float64) bool {
	ax, ay := cx-size*0.75, cy-size
	bx, by := cx-size*0.75, cy+size
	dx, dy := cx+size*0.85, cy

	d1 := (x-bx)*(ay-by) - (ax-bx)*(y-by)
	d2 := (x-dx)*(by-dy) - (bx-dx)*(y-dy)
	d3 := (x-ax)*(dy-ay) - (dx-ax)*(y-ay)

	hasNegative := d1 < 0 || d2 < 0 || d3 < 0
	hasPositive := d1 > 0 || d2 > 0 || d3 > 0
	return !(hasNegative && hasPositive)
}

// downsample averages each supersample block into one output pixel.
func downsample(src *image.RGBA, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	block := supersample * supersample

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var r, g, b, a, count int
			for dy := 0; dy < supersample; dy++ {
				for dx := 0; dx < supersample; dx++ {
					pixel := src.RGBAAt(x*supersample+dx, y*supersample+dy)
					r += int(pixel.R)
					g += int(pixel.G)
					b += int(pixel.B)
					a += int(pixel.A)
					count++
				}
			}
			if count == 0 || count != block {
				continue
			}
			dst.Set(x, y, color.RGBA{
				R: uint8(r / count),
				G: uint8(g / count),
				B: uint8(b / count),
				A: uint8(a / count),
			})
		}
	}
	return dst
}
