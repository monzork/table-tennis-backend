package event

import (
	"context"
	"errors"
	"strings"
	"testing"

	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
	singleTournamentDomain "table-tennis-backend/internal/domain/tournament"
)

func TestExportTournamentPdfUseCase_Execute(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1"}
		divRepo := &mockDivisionRepo{divisions: []*divisionDomain.Division{{ID: "d1"}}}
		pdfGen := &mockPdfGenerator{tournamentReportBytes: []byte("pdf-data")}
		uc := NewExportTournamentPdfUseCase(repo, divRepo, pdfGen)

		data, err := uc.Execute(context.Background(), "t1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if string(data) != "pdf-data" {
			t.Errorf("expected pdf-data, got %s", data)
		}
	})

	t.Run("tournament repo error propagates", func(t *testing.T) {
		repo := newMockRepo()
		repo.getErr = errors.New("db error")
		uc := NewExportTournamentPdfUseCase(repo, &mockDivisionRepo{}, &mockPdfGenerator{})

		_, err := uc.Execute(context.Background(), "missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("division error is ignored, still generates report", func(t *testing.T) {
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1"}
		divRepo := &mockDivisionRepo{getAllErr: errors.New("boom")}
		pdfGen := &mockPdfGenerator{tournamentReportBytes: []byte("ok")}
		uc := NewExportTournamentPdfUseCase(repo, divRepo, pdfGen)

		data, err := uc.Execute(context.Background(), "t1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if string(data) != "ok" {
			t.Errorf("expected ok, got %s", data)
		}
	})
}

func TestExportEventPdfUseCase_Execute(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newMockSingleTournamentRepo()
		repo.tournaments["e1"] = &singleTournamentDomain.Tournament{ID: "e1"}
		divRepo := &mockDivisionRepo{}
		pdfGen := &mockPdfGenerator{eventReportBytes: []byte("event-pdf")}
		uc := NewExportEventPdfUseCase(repo, divRepo, pdfGen)

		data, err := uc.Execute(context.Background(), "e1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if string(data) != "event-pdf" {
			t.Errorf("expected event-pdf, got %s", data)
		}
	})

	t.Run("event repo error propagates", func(t *testing.T) {
		repo := newMockSingleTournamentRepo()
		repo.getByIDDeepErr = errors.New("db error")
		uc := NewExportEventPdfUseCase(repo, &mockDivisionRepo{}, &mockPdfGenerator{})

		_, err := uc.Execute(context.Background(), "missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("division repo error propagates", func(t *testing.T) {
		repo := newMockSingleTournamentRepo()
		repo.tournaments["e1"] = &singleTournamentDomain.Tournament{ID: "e1"}
		divRepo := &mockDivisionRepo{getAllErr: errors.New("boom")}
		uc := NewExportEventPdfUseCase(repo, divRepo, &mockPdfGenerator{})

		_, err := uc.Execute(context.Background(), "e1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestExportTournamentReportUseCase_Execute(t *testing.T) {
	eloBefore := int16(1000)
	eloAfter := int16(1050)
	dEloBefore := int16(900)

	t.Run("success writes CSV with elo deltas and skips unknown snapshots", func(t *testing.T) {
		repo := newMockRepo()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A", Gender: "F", Country: "NI", Department: "Managua"}
		p2 := &playerDomain.Player{ID: "p2", FirstName: "Bob", LastName: "B", Gender: "M"} // no snapshot -> skipped
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", Participants: []*playerDomain.Player{p1, p2}}
		repo.snapshots = []tournamentDomain.ParticipantSnapshot{
			{PlayerID: "p1", EloBeforeSingles: &eloBefore, EloAfterSingles: &eloAfter, EloBeforeDoubles: &dEloBefore},
		}
		uc := NewExportTournamentReportUseCase(repo)

		data, err := uc.Execute(context.Background(), "t1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		csv := string(data)
		if !containsAll(csv, []string{"Alice A", "1000", "1050", "+50"}) {
			t.Errorf("expected csv to contain elo delta info, got %s", csv)
		}
		if containsAll(csv, []string{"Bob B"}) {
			t.Errorf("expected Bob to be skipped (no snapshot), got %s", csv)
		}
	})

	t.Run("tournament repo error propagates", func(t *testing.T) {
		repo := newMockRepo()
		repo.getErr = errors.New("db error")
		uc := NewExportTournamentReportUseCase(repo)

		_, err := uc.Execute(context.Background(), "missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("snapshots error propagates", func(t *testing.T) {
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1"}
		repo.snapshotsErr = errors.New("boom")
		uc := NewExportTournamentReportUseCase(repo)

		_, err := uc.Execute(context.Background(), "t1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func containsAll(s string, subs []string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
