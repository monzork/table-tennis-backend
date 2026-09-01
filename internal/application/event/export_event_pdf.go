package event

import (
	"context"

	"table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/pdf"
)

type ExportTournamentPdfUseCase struct {
	tournamentRepo tournamentDomain.Repository
	divisionRepo   division.Repository
	pdfGenerator   pdf.Generator
}

func NewExportTournamentPdfUseCase(tournamentRepo tournamentDomain.Repository, divisionRepo division.Repository, pdfGenerator pdf.Generator) *ExportTournamentPdfUseCase {
	return &ExportTournamentPdfUseCase{
		tournamentRepo: tournamentRepo,
		divisionRepo:   divisionRepo,
		pdfGenerator:   pdfGenerator,
	}
}

func (uc *ExportTournamentPdfUseCase) Execute(ctx context.Context, tournamentIDStr string, lang string) ([]byte, error) {
	t, err := uc.tournamentRepo.GetByID(ctx, tournamentIDStr)
	if err != nil {
		return nil, err
	}
	divs, _ := uc.divisionRepo.GetAll(ctx)
	t.ParticipantSnapshots, _ = uc.tournamentRepo.GetParticipantSnapshots(ctx, tournamentIDStr)
	return uc.pdfGenerator.GenerateTournamentReport(t, divs, lang)
}
