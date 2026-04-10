package player

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	playerDomain "table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
)

// ImportResult summarises the outcome of a bulk import.
type ImportResult struct {
	Imported int
	Skipped  int
	Errors   []string
}

// ImportPlayersUseCase reads a CSV or Excel file and bulk-inserts players.
type ImportPlayersUseCase struct {
	playerRepo *bun.PlayerRepository
}

func NewImportPlayersUseCase(repo *bun.PlayerRepository) *ImportPlayersUseCase {
	return &ImportPlayersUseCase{playerRepo: repo}
}

// Execute reads records from r (CSV or XLSX determined by filename extension)
// and persists each player.  Expected columns (header row required):
//
//	first_name | last_name | birthdate (YYYY-MM-DD) | gender (M/F) | country
//
// Optional columns: singles_elo, doubles_elo
func (uc *ImportPlayersUseCase) Execute(ctx context.Context, filename string, r io.Reader) (*ImportResult, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".csv":
		return uc.importCSV(ctx, r)
	case ".xlsx", ".xls":
		return uc.importExcel(ctx, filename, r)
	default:
		return nil, fmt.Errorf("unsupported file type: %s (accepted: .csv, .xlsx)", ext)
	}
}

// ─── CSV ─────────────────────────────────────────────────────────────────────

func (uc *ImportPlayersUseCase) importCSV(ctx context.Context, r io.Reader) (*ImportResult, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}
	colIdx := buildColIndex(headers)

	result := &ImportResult{}
	rowNum := 1
	for {
		rowNum++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: parse error: %v", rowNum, err))
			result.Skipped++
			continue
		}
		if err := uc.insertRow(ctx, record, colIdx, rowNum, result); err != nil {
			// insertRow already appended the error
		}
	}
	return result, nil
}

// ─── Excel ───────────────────────────────────────────────────────────────────

func (uc *ImportPlayersUseCase) importExcel(ctx context.Context, filename string, r io.Reader) (*ImportResult, error) {
	// excelize needs a ReadSeeker; buffer into memory
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	f, err := excelize.OpenReader(strings.NewReader(string(data)))
	if err != nil {
		// Try again with raw bytes via a temp approach
		f2, err2 := excelize.OpenReader(newBytesReader(data))
		if err2 != nil {
			return nil, fmt.Errorf("failed to parse Excel file: %w", err)
		}
		f = f2
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("Excel file has no sheets")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet rows: %w", err)
	}
	if len(rows) < 2 {
		return &ImportResult{}, nil // empty or header-only
	}

	colIdx := buildColIndex(rows[0])
	result := &ImportResult{}
	for i, row := range rows[1:] {
		uc.insertRow(ctx, row, colIdx, i+2, result)
	}
	return result, nil
}

// ─── Shared helpers ──────────────────────────────────────────────────────────

// buildColIndex maps lowercase normalised header names to their column index.
func buildColIndex(headers []string) map[string]int {
	idx := make(map[string]int, len(headers))
	for i, h := range headers {
		idx[normalise(h)] = i
	}
	return idx
}

func normalise(s string) string {
	return strings.ToLower(strings.TrimSpace(strings.ReplaceAll(s, " ", "_")))
}

func cell(row []string, col map[string]int, key string) string {
	i, ok := col[key]
	if !ok || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func (uc *ImportPlayersUseCase) insertRow(ctx context.Context, row []string, colIdx map[string]int, rowNum int, result *ImportResult) error {
	firstName := cell(row, colIdx, "first_name")
	lastName := cell(row, colIdx, "last_name")
	birthStr := cell(row, colIdx, "birthdate")
	gender := strings.ToUpper(cell(row, colIdx, "gender"))
	country := strings.ToUpper(cell(row, colIdx, "country"))
	singlesEloStr := cell(row, colIdx, "singles_elo")
	doublesEloStr := cell(row, colIdx, "doubles_elo")

	if firstName == "" || lastName == "" {
		result.Errors = append(result.Errors, fmt.Sprintf("row %d: missing first_name or last_name — skipped", rowNum))
		result.Skipped++
		return nil
	}

	if gender != "M" && gender != "F" {
		gender = "M" // default
	}

	var birthdate time.Time
	if birthStr != "" {
		for _, layout := range []string{"2006-01-02", "02/01/2006", "01/02/2006", "2006/01/02"} {
			if t, err := time.Parse(layout, birthStr); err == nil {
				birthdate = t
				break
			}
		}
	}
	if birthdate.IsZero() {
		birthdate = time.Now()
	}

	p, err := playerDomain.NewPlayer(firstName, lastName, birthdate, gender, country)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("row %d: %v", rowNum, err))
		result.Skipped++
		return err
	}

	// Apply optional Elo values if provided
	if v, err := strconv.Atoi(singlesEloStr); err == nil && v > 0 {
		p.UpdateSinglesElo(int16(v))
	}
	if v, err := strconv.Atoi(doublesEloStr); err == nil && v > 0 {
		p.UpdateDoublesElo(int16(v))
	}

	if err := uc.playerRepo.Save(ctx, p); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("row %d: db error: %v", rowNum, err))
		result.Skipped++
		return err
	}

	result.Imported++
	return nil
}

// newBytesReader wraps a []byte as an io.Reader (used for excelize).
type bytesReader struct{ *strings.Reader }

func newBytesReader(b []byte) io.Reader { return strings.NewReader(string(b)) }
