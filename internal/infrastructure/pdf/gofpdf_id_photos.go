package pdf

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log/slog"
	"path/filepath"
	"strings"

	"table-tennis-backend/internal/domain/player"

	xdraw "golang.org/x/image/draw"

	"github.com/go-pdf/fpdf"
)

// maxIDPhotoDimension is the largest width/height (in pixels) an ID photo is
// downscaled to before embedding. Phone-camera photos routinely run
// 3000-4000px / several MB each; a report with many players embedding them
// at full resolution has been observed to push the server over its memory
// limit and get OOM-killed mid-generation. A cédula only needs to be legible
// on a printed/PDF page, not full sensor resolution.
const maxIDPhotoDimension = 1000

// idPhotoJPEGQuality is the re-encode quality for the downscaled photo.
const idPhotoJPEGQuality = 80

// isSupportedIDPhotoFormat reports whether the storage path's extension is
// one decodeIDPhoto can handle, checked before downloading so an
// unsupported format (e.g. .webp — not decodable by the stdlib image
// package) is skipped without wasting a download.
func isSupportedIDPhotoFormat(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".gif":
		return true
	default:
		return false
	}
}

// decodeIDPhoto decodes raw downloaded bytes by the format implied by the
// storage path's extension, returning nil with no error for an unsupported
// format (caller skips it rather than treating it as a hard failure).
func decodeIDPhoto(path string, data []byte) (image.Image, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		img, err := jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		return applyJPEGOrientation(img, jpegOrientation(data)), nil
	case ".png":
		return png.Decode(bytes.NewReader(data))
	case ".gif":
		return gif.Decode(bytes.NewReader(data))
	default:
		return nil, nil
	}
}

// prepareIDPhoto decodes, corrects EXIF rotation (JPEG only), downscales if
// needed, and re-encodes as a modest-quality JPEG — normalizing every
// supported input format to one small buffer so fpdf embeds a consistent,
// print-sized image instead of a multi-megabyte original.
func prepareIDPhoto(path string, data []byte) ([]byte, error) {
	img, err := decodeIDPhoto(path, data)
	if err != nil {
		return nil, err
	}
	if img == nil {
		return nil, nil
	}

	b := img.Bounds()
	if w, h := b.Dx(), b.Dy(); w > maxIDPhotoDimension || h > maxIDPhotoDimension {
		scale := float64(maxIDPhotoDimension) / float64(w)
		if h > w {
			scale = float64(maxIDPhotoDimension) / float64(h)
		}
		newW, newH := int(float64(w)*scale), int(float64(h)*scale)
		if newW < 1 {
			newW = 1
		}
		if newH < 1 {
			newH = 1
		}
		dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
		xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
		img = dst
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: idPhotoJPEGQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// jpegOrientation returns the EXIF Orientation tag (1-8) from raw JPEG
// bytes, or 1 (no rotation) if absent/unparseable. Phone cameras commonly
// save sensor-orientation pixel data as-is and rely on this tag for viewers
// to rotate it upright on display; fpdf embeds pixel data verbatim and
// ignores it, so an ID photo taken in portrait can render sideways in the
// PDF unless the rotation is applied before embedding.
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != 0xFF {
			break
		}
		marker := data[pos+1]
		if marker == 0xD8 || marker == 0xD9 {
			pos += 2
			continue
		}
		if marker == 0xDA { // start of scan: no more markers before compressed data
			break
		}
		segLen := int(data[pos+2])<<8 | int(data[pos+3])
		if marker == 0xE1 && pos+2+segLen <= len(data) {
			if o := exifOrientation(data[pos+4 : pos+2+segLen]); o != 0 {
				return o
			}
		}
		pos += 2 + segLen
	}
	return 1
}

// exifOrientation reads the Orientation tag (0x0112) out of a JPEG APP1
// segment's Exif/TIFF payload. Returns 0 if the segment isn't a valid
// Exif/TIFF block or carries no Orientation tag.
func exifOrientation(seg []byte) int {
	if len(seg) < 10 || string(seg[0:6]) != "Exif\x00\x00" {
		return 0
	}
	tiff := seg[6:]
	if len(tiff) < 8 {
		return 0
	}
	var bo binary.ByteOrder
	switch string(tiff[0:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 0
	}
	ifdOffset := bo.Uint32(tiff[4:8])
	if int(ifdOffset)+2 > len(tiff) {
		return 0
	}
	numEntries := int(bo.Uint16(tiff[ifdOffset : ifdOffset+2]))
	for i := 0; i < numEntries; i++ {
		entryOffset := int(ifdOffset) + 2 + i*12
		if entryOffset+12 > len(tiff) {
			break
		}
		tag := bo.Uint16(tiff[entryOffset : entryOffset+2])
		if tag == 0x0112 {
			val := bo.Uint16(tiff[entryOffset+8 : entryOffset+10])
			if val >= 1 && val <= 8 {
				return int(val)
			}
		}
	}
	return 0
}

// applyJPEGOrientation rotates/flips img so it displays upright, per the
// EXIF Orientation convention (1 = already upright, 2-8 = various
// rotate/mirror combinations a camera can write instead of physically
// rotating the pixel data).
func applyJPEGOrientation(img image.Image, orientation int) image.Image {
	if orientation == 1 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	rotated := w != h && (orientation == 5 || orientation == 6 || orientation == 7 || orientation == 8)
	outW, outH := w, h
	if rotated {
		outW, outH = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, outW, outH))

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.At(b.Min.X+x, b.Min.Y+y)
			var dx, dy int
			switch orientation {
			case 2: // mirror horizontal
				dx, dy = w-1-x, y
			case 3: // rotate 180
				dx, dy = w-1-x, h-1-y
			case 4: // mirror vertical
				dx, dy = x, h-1-y
			case 5: // mirror horizontal + rotate 270 CW
				dx, dy = y, x
			case 6: // rotate 90 CW
				dx, dy = h-1-y, x
			case 7: // mirror horizontal + rotate 90 CW
				dx, dy = h-1-y, w-1-x
			case 8: // rotate 270 CW
				dx, dy = y, w-1-x
			default:
				dx, dy = x, y
			}
			dst.Set(dx, dy, c)
		}
	}
	return dst
}

// appendIDPhotos adds one page per player (who has at least one ID photo on
// file) showing their name and the front/back of their cédula de identidad,
// at the end of the report. Players with neither photo are skipped
// entirely, and a single failed/unsupported/undecodable download just skips
// that one image rather than failing the whole report.
func appendIDPhotos(pdf *fpdf.Fpdf, players []*player.Player, downloader PhotoDownloader, tr func(string) string, lang string) {
	if downloader == nil {
		return
	}

	seen := make(map[string]bool)
	var withPhotos []*player.Player
	for _, p := range players {
		if p == nil || seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		if p.IDFrontPath != "" || p.IDBackPath != "" {
			withPhotos = append(withPhotos, p)
		}
	}
	if len(withPhotos) == 0 {
		return
	}

	ctx := context.Background()

	registerPhoto := func(imgName, path string) bool {
		if path == "" {
			return false
		}
		if !isSupportedIDPhotoFormat(path) {
			slog.Warn("skipping ID photo with unsupported format", "path", path)
			return false
		}
		raw, err := downloader.Download(ctx, path)
		if err != nil {
			slog.Warn("failed to download ID photo for report", "path", path, "error", err)
			return false
		}
		jpegData, err := prepareIDPhoto(path, raw)
		if err != nil {
			slog.Warn("failed to decode ID photo for report", "path", path, "error", err)
			return false
		}
		if jpegData == nil {
			slog.Warn("skipping ID photo with unsupported format", "path", path)
			return false
		}
		pdf.RegisterImageOptionsReader(imgName, fpdf.ImageOptions{ImageType: "JPG"}, bytes.NewReader(jpegData))
		return true
	}

	for _, p := range withPhotos {
		pdf.AddPage()

		pdf.SetFont("Arial", "B", 14)
		name := strings.TrimSpace(p.FirstNameWithSecond() + " " + p.LastNameWithSecond())
		pdf.CellFormat(0, 10, tr(strings.ToUpper(L(lang, "ID CARD - ")+name)), "", 1, "L", false, 0, "")
		if p.NationalID != "" {
			pdf.SetFont("Arial", "", 10)
			pdf.CellFormat(0, 8, tr(L(lang, "ID No: ")+p.NationalID), "", 1, "L", false, 0, "")
		}
		pdf.Ln(6)

		w, _ := pdf.GetPageSize()
		printableW := w - 30 // 15mm margins each side
		imgW := printableW
		imgH := 90.0

		frontName := "idfront-" + p.ID
		if registerPhoto(frontName, p.IDFrontPath) {
			pdf.SetFont("Arial", "B", 10)
			pdf.CellFormat(0, 6, tr(L(lang, "Front")), "", 1, "L", false, 0, "")
			pdf.ImageOptions(frontName, pdf.GetX(), pdf.GetY(), imgW, imgH, false, fpdf.ImageOptions{ImageType: "JPG"}, 0, "")
			pdf.SetY(pdf.GetY() + imgH + 8)
		}

		backName := "idback-" + p.ID
		if registerPhoto(backName, p.IDBackPath) {
			pdf.SetFont("Arial", "B", 10)
			pdf.CellFormat(0, 6, tr(L(lang, "Back")), "", 1, "L", false, 0, "")
			pdf.ImageOptions(backName, pdf.GetX(), pdf.GetY(), imgW, imgH, false, fpdf.ImageOptions{ImageType: "JPG"}, 0, "")
			pdf.SetY(pdf.GetY() + imgH + 8)
		}
	}
}
