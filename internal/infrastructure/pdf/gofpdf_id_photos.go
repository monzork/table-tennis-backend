package pdf

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
	"strings"

	"table-tennis-backend/internal/domain/player"

	"github.com/go-pdf/fpdf"
)

// imageTypeFromPath maps a storage object's file extension to the image
// type string go-pdf/fpdf expects ("JPG", "PNG", "GIF" — its only natively
// supported formats). Returns "" for anything else so the caller can skip it
// instead of feeding fpdf a format it can't decode.
func imageTypeFromPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "JPG"
	case ".png":
		return "PNG"
	case ".gif":
		return "GIF"
	default:
		return ""
	}
}

// appendIDPhotos adds one page per player (who has at least one ID photo on
// file) showing their name and the front/back of their cédula de identidad,
// at the end of the report. Players with neither photo are skipped
// entirely, and a single failed/unsupported download just skips that one
// image rather than failing the whole report.
func appendIDPhotos(pdf *fpdf.Fpdf, players []*player.Player, downloader PhotoDownloader, tr func(string) string) {
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
		tp := imageTypeFromPath(path)
		if tp == "" {
			slog.Warn("skipping ID photo with unsupported format", "path", path)
			return false
		}
		data, err := downloader.Download(ctx, path)
		if err != nil {
			slog.Warn("failed to download ID photo for report", "path", path, "error", err)
			return false
		}
		pdf.RegisterImageOptionsReader(imgName, fpdf.ImageOptions{ImageType: tp, ReadDpi: true}, bytes.NewReader(data))
		return true
	}

	for _, p := range withPhotos {
		pdf.AddPage()

		pdf.SetFont("Arial", "B", 14)
		name := strings.TrimSpace(p.FirstNameWithSecond() + " " + p.LastNameWithSecond())
		pdf.CellFormat(0, 10, tr(strings.ToUpper("CÉDULA DE IDENTIDAD - "+name)), "", 1, "L", false, 0, "")
		if p.NationalID != "" {
			pdf.SetFont("Arial", "", 10)
			pdf.CellFormat(0, 8, tr("Cédula No: "+p.NationalID), "", 1, "L", false, 0, "")
		}
		pdf.Ln(6)

		w, _ := pdf.GetPageSize()
		printableW := w - 30 // 15mm margins each side
		imgW := printableW
		imgH := 90.0

		frontName := "idfront-" + p.ID
		if registerPhoto(frontName, p.IDFrontPath) {
			pdf.SetFont("Arial", "B", 10)
			pdf.CellFormat(0, 6, tr("Frente"), "", 1, "L", false, 0, "")
			pdf.ImageOptions(frontName, pdf.GetX(), pdf.GetY(), imgW, imgH, false, fpdf.ImageOptions{ImageType: imageTypeFromPath(p.IDFrontPath)}, 0, "")
			pdf.SetY(pdf.GetY() + imgH + 8)
		}

		backName := "idback-" + p.ID
		if registerPhoto(backName, p.IDBackPath) {
			pdf.SetFont("Arial", "B", 10)
			pdf.CellFormat(0, 6, tr("Reverso"), "", 1, "L", false, 0, "")
			pdf.ImageOptions(backName, pdf.GetX(), pdf.GetY(), imgW, imgH, false, fpdf.ImageOptions{ImageType: imageTypeFromPath(p.IDBackPath)}, 0, "")
			pdf.SetY(pdf.GetY() + imgH + 8)
		}
	}
}
