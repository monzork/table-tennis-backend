package pdf

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

// buildExifAPP1 builds a minimal JPEG APP1 (Exif) segment carrying a single
// Orientation (0x0112) tag, little-endian TIFF, for splicing into a JPEG
// right after its SOI marker.
func buildExifAPP1(orientation uint16) []byte {
	tiff := []byte{
		'I', 'I', 0x2A, 0x00, // byte order + TIFF magic
		0x08, 0x00, 0x00, 0x00, // IFD0 offset = 8
		0x01, 0x00, // 1 entry
		0x12, 0x01, // tag 0x0112 (Orientation)
		0x03, 0x00, // type SHORT
		0x01, 0x00, 0x00, 0x00, // count 1
		byte(orientation), byte(orientation >> 8), 0x00, 0x00, // value
		0x00, 0x00, 0x00, 0x00, // next IFD offset
	}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	segLen := len(payload) + 2
	seg := []byte{0xFF, 0xE1, byte(segLen >> 8), byte(segLen)}
	return append(seg, payload...)
}

func withExifOrientation(jpegData []byte, orientation uint16) []byte {
	app1 := buildExifAPP1(orientation)
	// jpegData starts with SOI (FF D8); splice APP1 right after it.
	out := make([]byte, 0, len(jpegData)+len(app1))
	out = append(out, jpegData[:2]...)
	out = append(out, app1...)
	out = append(out, jpegData[2:]...)
	return out
}

func encodeTestJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Distinct colors per corner so rotation can be verified.
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})   // top-left: red
	img.Set(w-1, 0, color.RGBA{0, 255, 0, 255}) // top-right: green
	img.Set(0, h-1, color.RGBA{0, 0, 255, 255}) // bottom-left: blue
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("failed to encode fixture JPEG: %v", err)
	}
	return buf.Bytes()
}

func TestJpegOrientation(t *testing.T) {
	base := encodeTestJPEG(t, 4, 4)

	t.Run("no EXIF segment defaults to 1", func(t *testing.T) {
		if o := jpegOrientation(base); o != 1 {
			t.Errorf("expected 1, got %d", o)
		}
	})

	t.Run("reads Orientation tag from APP1", func(t *testing.T) {
		withOrientation := withExifOrientation(base, 6)
		if o := jpegOrientation(withOrientation); o != 6 {
			t.Errorf("expected 6, got %d", o)
		}
	})

	t.Run("garbage bytes default to 1", func(t *testing.T) {
		if o := jpegOrientation([]byte{0x00, 0x01, 0x02}); o != 1 {
			t.Errorf("expected 1, got %d", o)
		}
	})

	t.Run("too short defaults to 1", func(t *testing.T) {
		if o := jpegOrientation([]byte{0xFF}); o != 1 {
			t.Errorf("expected 1, got %d", o)
		}
	})
}

func TestApplyJPEGOrientation(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	red := color.RGBA{255, 0, 0, 255}
	img.Set(0, 0, red) // top-left

	t.Run("orientation 1 returns image unchanged", func(t *testing.T) {
		out := applyJPEGOrientation(img, 1)
		if out != image.Image(img) {
			t.Error("expected the same image instance for orientation 1")
		}
	})

	t.Run("orientation 6 (90 CW) swaps dimensions", func(t *testing.T) {
		out := applyJPEGOrientation(img, 6)
		b := out.Bounds()
		if b.Dx() != 2 || b.Dy() != 3 {
			t.Errorf("expected 2x3 after 90CW rotation of 3x2, got %dx%d", b.Dx(), b.Dy())
		}
		// Top-left red pixel should now be at the top-right.
		r, g, bl, _ := out.At(1, 0).RGBA()
		if r>>8 != 255 || g>>8 != 0 || bl>>8 != 0 {
			t.Errorf("expected red pixel at (1,0) after rotation, got rgb(%d,%d,%d)", r>>8, g>>8, bl>>8)
		}
	})

	t.Run("orientation 3 (180) keeps dimensions", func(t *testing.T) {
		out := applyJPEGOrientation(img, 3)
		b := out.Bounds()
		if b.Dx() != 3 || b.Dy() != 2 {
			t.Errorf("expected 3x2 after 180 rotation, got %dx%d", b.Dx(), b.Dy())
		}
	})
}

func TestPrepareIDPhoto(t *testing.T) {
	t.Run("downscales an oversized image", func(t *testing.T) {
		big := encodeTestJPEG(t, 2000, 1500)
		out, err := prepareIDPhoto("id_front/big.jpg", big)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		img, err := jpeg.Decode(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("failed to decode output: %v", err)
		}
		b := img.Bounds()
		if b.Dx() > maxIDPhotoDimension || b.Dy() > maxIDPhotoDimension {
			t.Errorf("expected downscaled to <=%d, got %dx%d", maxIDPhotoDimension, b.Dx(), b.Dy())
		}
		if len(out) >= len(big) {
			t.Errorf("expected downscaled output smaller than input (%d bytes), got %d bytes", len(big), len(out))
		}
	})

	t.Run("small image left at original size", func(t *testing.T) {
		small := encodeTestJPEG(t, 100, 80)
		out, err := prepareIDPhoto("id_front/small.jpg", small)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		img, err := jpeg.Decode(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("failed to decode output: %v", err)
		}
		b := img.Bounds()
		if b.Dx() != 100 || b.Dy() != 80 {
			t.Errorf("expected 100x80 unchanged, got %dx%d", b.Dx(), b.Dy())
		}
	})

	t.Run("unsupported format returns nil, nil", func(t *testing.T) {
		out, err := prepareIDPhoto("id_front/photo.webp", []byte("not-a-real-webp"))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if out != nil {
			t.Errorf("expected nil for unsupported format, got %d bytes", len(out))
		}
	})

	t.Run("corrupt data returns error", func(t *testing.T) {
		_, err := prepareIDPhoto("id_front/photo.jpg", []byte("not-a-real-jpeg"))
		if err == nil {
			t.Fatal("expected an error for corrupt JPEG data")
		}
	})
}

func TestIsSupportedIDPhotoFormat(t *testing.T) {
	cases := map[string]bool{
		"id_front/a.jpg":  true,
		"id_front/a.JPEG": true,
		"id_front/a.png":  true,
		"id_front/a.gif":  true,
		"id_front/a.webp": false,
		"id_front/a":      false,
	}
	for path, want := range cases {
		if got := isSupportedIDPhotoFormat(path); got != want {
			t.Errorf("isSupportedIDPhotoFormat(%q) = %v, want %v", path, got, want)
		}
	}
}
