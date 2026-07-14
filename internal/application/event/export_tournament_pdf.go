package event

import (
	"context"

	divisionDomain "table-tennis-backend/internal/domain/division"
	eventDomain "table-tennis-backend/internal/domain/tournament"
	"table-tennis-backend/internal/domain/pdf"
)

type ExportEventPdfUseCase struct {
	eventRepo    eventDomain.Repository
	divisionRepo divisionDomain.Repository
	pdfGenerator pdf.Generator
}

func NewExportEventPdfUseCase(eventRepo eventDomain.Repository, divisionRepo divisionDomain.Repository, pdfGenerator pdf.Generator) *ExportEventPdfUseCase {
	return &ExportEventPdfUseCase{
		eventRepo:    eventRepo,
		divisionRepo: divisionRepo,
		pdfGenerator: pdfGenerator,
	}
}

func (uc *ExportEventPdfUseCase) Execute(ctx context.Context, eventID string) ([]byte, error) {
	e, err := uc.eventRepo.GetByIDDeep(ctx, eventID)
	if err != nil {
		return nil, err
	}
	divs, err := uc.divisionRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	return uc.pdfGenerator.GenerateEventReport(e, divs)
}
