package event

import (
	"context"

	divisionDomain "table-tennis-backend/internal/domain/division"
	subEventDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/pdf"
	eventDomain "table-tennis-backend/internal/domain/tournament"

	"golang.org/x/sync/errgroup"
)

type ExportEventPdfUseCase struct {
	eventRepo    eventDomain.Repository
	subEventRepo subEventDomain.ParticipantRepository
	divisionRepo divisionDomain.Repository
	pdfGenerator pdf.Generator
}

func NewExportEventPdfUseCase(eventRepo eventDomain.Repository, subEventRepo subEventDomain.ParticipantRepository, divisionRepo divisionDomain.Repository, pdfGenerator pdf.Generator) *ExportEventPdfUseCase {
	return &ExportEventPdfUseCase{
		eventRepo:    eventRepo,
		subEventRepo: subEventRepo,
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

	// Fetch each child event's participant Elo snapshots concurrently, since
	// they're independent per-event reads.
	eg, egCtx := errgroup.WithContext(ctx)
	for _, childEvent := range e.Events {
		eg.Go(func() error {
			childEvent.ParticipantSnapshots, _ = uc.subEventRepo.GetParticipantSnapshots(egCtx, childEvent.ID)
			return nil
		})
	}
	_ = eg.Wait()

	return uc.pdfGenerator.GenerateEventReport(e, divs)
}
