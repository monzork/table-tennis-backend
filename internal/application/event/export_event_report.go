package event

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	tournamentDomain "table-tennis-backend/internal/domain/event"
)

type ExportTournamentReportUseCase struct {
	tournamentRepo tournamentDomain.Repository
}

func NewExportTournamentReportUseCase(tournamentRepo tournamentDomain.Repository) *ExportTournamentReportUseCase {
	return &ExportTournamentReportUseCase{tournamentRepo: tournamentRepo}
}

func (uc *ExportTournamentReportUseCase) Execute(ctx context.Context, tournamentIDStr string) ([]byte, error) {
	t, err := uc.tournamentRepo.GetByID(ctx, tournamentIDStr)
	if err != nil {
		return nil, err
	}

	snapshots, err := uc.tournamentRepo.GetParticipantSnapshots(ctx, tournamentIDStr)
	if err != nil {
		return nil, err
	}

	snapshotMap := make(map[string]tournamentDomain.ParticipantSnapshot)
	for _, s := range snapshots {
		snapshotMap[s.PlayerID] = s
	}

	// Create CSV
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Header
	header := []string{
		"Player Name",
		"Gender",
		"Country",
		"Department",
		"Elo Before (Singles)",
		"Elo After (Singles)",
		"Elo Delta (Singles)",
		"Elo Before (Doubles)",
		"Elo After (Doubles)",
		"Elo Delta (Doubles)",
	}
	_ = writer.Write(header)

	for _, p := range t.Participants {
		snap, ok := snapshotMap[p.ID]
		if !ok {
			continue
		}

		bSingles := "-"
		aSingles := "-"
		dSingles := "-"
		bDoubles := "-"
		aDoubles := "-"
		dDoubles := "-"

		if snap.EloBeforeSingles != nil {
			bSingles = fmt.Sprintf("%d", *snap.EloBeforeSingles)
			if snap.EloAfterSingles != nil {
				aSingles = fmt.Sprintf("%d", *snap.EloAfterSingles)
				dSingles = fmt.Sprintf("%+d", *snap.EloAfterSingles-*snap.EloBeforeSingles)
			}
		}

		if snap.EloBeforeDoubles != nil {
			bDoubles = fmt.Sprintf("%d", *snap.EloBeforeDoubles)
			if snap.EloAfterDoubles != nil {
				aDoubles = fmt.Sprintf("%d", *snap.EloAfterDoubles)
				dDoubles = fmt.Sprintf("%+d", *snap.EloAfterDoubles-*snap.EloBeforeDoubles)
			}
		}

		_ = writer.Write([]string{
			p.FullName(),
			p.Gender,
			p.Country,
			p.Department,
			bSingles,
			aSingles,
			dSingles,
			bDoubles,
			aDoubles,
			dDoubles,
		})
	}

	writer.Flush()
	return buf.Bytes(), nil
}
