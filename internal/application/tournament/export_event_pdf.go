package tournament

import (
	"context"

	eventDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/pdf"
)

type ExportEventPdfUseCase struct {
	eventRepo    eventDomain.Repository
	pdfGenerator pdf.Generator
}

func NewExportEventPdfUseCase(eventRepo eventDomain.Repository, pdfGenerator pdf.Generator) *ExportEventPdfUseCase {
	return &ExportEventPdfUseCase{
		eventRepo:    eventRepo,
		pdfGenerator: pdfGenerator,
	}
}

func (uc *ExportEventPdfUseCase) Execute(ctx context.Context, eventID string) ([]byte, error) {
	e, err := uc.eventRepo.GetByIDDeep(ctx, eventID)
	if err != nil {
		return nil, err
	}
	return uc.pdfGenerator.GenerateEventReport(e)
}
