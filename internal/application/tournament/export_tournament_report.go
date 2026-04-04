package tournament

import (
	"context"
	"fmt"
	"encoding/csv"
	"bytes"
	
	"github.com/google/uuid"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
)

type ExportTournamentReportUseCase struct {
	tournamentRepo *bun.TournamentRepository
}

func NewExportTournamentReportUseCase(tournamentRepo *bun.TournamentRepository) *ExportTournamentReportUseCase {
	return &ExportTournamentReportUseCase{tournamentRepo: tournamentRepo}
}

func (uc *ExportTournamentReportUseCase) Execute(ctx context.Context, tournamentIDStr string) ([]byte, error) {
	id, err := uuid.Parse(tournamentIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid tournament id")
	}

	t, err := uc.tournamentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// We query the tournament_participants directly to get the historical Elo before/after
	type ParticipantSnapshot struct {
		PlayerID         uuid.UUID
		EloBeforeSingles *int16
		EloAfterSingles  *int16
		EloBeforeDoubles *int16
		EloAfterDoubles  *int16
	}

	var snapshots []ParticipantSnapshot
	err = uc.tournamentRepo.DB().NewSelect().
		Table("tournament_participants").
		Column("player_id", "elo_before_singles", "elo_after_singles", "elo_before_doubles", "elo_after_doubles").
		Where("tournament_id = ?", id).
		Scan(ctx, &snapshots)
	if err != nil {
		return nil, err
	}

	snapshotMap := make(map[uuid.UUID]ParticipantSnapshot)
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
				dSingles = fmt.Sprintf("%+d", *snap.EloAfterSingles - *snap.EloBeforeSingles)
			}
		}

		if snap.EloBeforeDoubles != nil {
			bDoubles = fmt.Sprintf("%d", *snap.EloBeforeDoubles)
			if snap.EloAfterDoubles != nil {
				aDoubles = fmt.Sprintf("%d", *snap.EloAfterDoubles)
				dDoubles = fmt.Sprintf("%+d", *snap.EloAfterDoubles - *snap.EloBeforeDoubles)
			}
		}

		_ = writer.Write([]string{
			p.FullName(),
			p.Gender,
			p.Country,
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
