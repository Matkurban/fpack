package pack

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

// PNGSize returns the pixel size of a PNG file.
func PNGSize(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		return 0, 0, fmt.Errorf("%s: not a PNG: %w", filepath.Base(path), err)
	}
	return cfg.Width, cfg.Height, nil
}

// ResizePNG writes a size×size copy of src (area-averaged, alpha aware).
// Non-square sources are centered on a transparent square first.
func ResizePNG(src, dst string, size int) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(src), err)
	}
	b := img.Bounds()
	side := b.Dx()
	if b.Dy() > side {
		side = b.Dy()
	}
	sq := image.NewNRGBA(image.Rect(0, 0, side, side))
	draw.Draw(sq, image.Rect((side-b.Dx())/2, (side-b.Dy())/2, side, side), img, b.Min, draw.Src)
	out := image.NewNRGBA(image.Rect(0, 0, size, size))
	scale := float64(side) / float64(size)
	for y := 0; y < size; y++ {
		y0, y1 := float64(y)*scale, float64(y+1)*scale
		for x := 0; x < size; x++ {
			x0, x1 := float64(x)*scale, float64(x+1)*scale
			var r, g, bl, a, wsum float64
			for sy := int(y0); sy < int(ceil(y1)) && sy < side; sy++ {
				wy := overlap(y0, y1, float64(sy))
				for sx := int(x0); sx < int(ceil(x1)) && sx < side; sx++ {
					w := wy * overlap(x0, x1, float64(sx))
					c := sq.NRGBAAt(sx, sy)
					al := float64(c.A) / 255
					r += float64(c.R) * al * w
					g += float64(c.G) * al * w
					bl += float64(c.B) * al * w
					a += al * w
					wsum += w
				}
			}
			if wsum == 0 || a == 0 {
				continue
			}
			out.SetNRGBA(x, y, color.NRGBA{R: u8(r / a), G: u8(g / a), B: u8(bl / a), A: u8(a / wsum * 255)})
		}
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	if err := png.Encode(w, out); err != nil {
		w.Close()
		return err
	}
	return w.Close()
}

func overlap(a, b, p float64) float64 {
	lo, hi := p, p+1
	if a > lo {
		lo = a
	}
	if b < hi {
		hi = b
	}
	if hi <= lo {
		return 0
	}
	return hi - lo
}

func ceil(f float64) float64 {
	i := float64(int(f))
	if i < f {
		return i + 1
	}
	return i
}

func u8(f float64) uint8 {
	if f < 0 {
		return 0
	}
	if f > 255 {
		return 255
	}
	return uint8(f + 0.5)
}
