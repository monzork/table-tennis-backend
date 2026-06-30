package tournament

import (
	"context"

	"table-tennis-backend/internal/domain/pdf"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
)

type ExportTournamentPdfUseCase struct {
	tournamentRepo tournamentDomain.Repository
	pdfGenerator   pdf.Generator
}

func NewExportTournamentPdfUseCase(tournamentRepo tournamentDomain.Repository, pdfGenerator pdf.Generator) *ExportTournamentPdfUseCase {
	return &ExportTournamentPdfUseCase{
		tournamentRepo: tournamentRepo,
		pdfGenerator:   pdfGenerator,
	}
}

func (uc *ExportTournamentPdfUseCase) Execute(ctx context.Context, tournamentIDStr string) ([]byte, error) {
	t, err := uc.tournamentRepo.GetByID(ctx, tournamentIDStr)
	if err != nil {
		return nil, err
	}
	return uc.pdfGenerator.GenerateTournamentReport(t)
}
