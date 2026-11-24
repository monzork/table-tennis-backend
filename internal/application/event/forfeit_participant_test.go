package event

import (
	"context"
	"testing"

	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
)

type forfeitMatchRepo struct {
	*mockMatchRepo
	matches map[string]*tournamentDomain.Match
}

func (r *forfeitMatchRepo) FindOrCreateMatch(_ context.Context, _, p1, p2, _, _ string) (string, error) {
	id := p1 + "-" + p2
	if _, ok := r.matches[id]; !ok {
		r.matches[id] = &tournamentDomain.Match{ID: id, Status: "scheduled",
			TeamA: []*playerDomain.Player{{ID: p1}}, TeamB: []*playerDomain.Player{{ID: p2}}}
	}
	return id, nil
}

func (r *forfeitMatchRepo) GetByID(_ context.Context, id string) (*tournamentDomain.Match, error) {
	return r.matches[id], nil
}

type fakeWalkover struct{ calls []string }

func (f *fakeWalkover) ExecuteWalkover(_ context.Context, matchID, _, _, side string) error {
	f.calls = append(f.calls, matchID+":"+side)
	return nil
}

func TestForfeitParticipant_GroupAndKnockout(t *testing.T) {
	p1, p2, p3 := &playerDomain.Player{ID: "p1"}, &playerDomain.Player{ID: "p2"}, &playerDomain.Player{ID: "p3"}
	ev := &tournamentDomain.Event{
		ID: "e1", Type: "singles",
		Groups: []tournamentDomain.Group{{ID: "g", Players: []*playerDomain.Player{p1, p2, p3}}},
		Matches: []tournamentDomain.Match{
			{ID: "ko", Stage: "r16", Status: "scheduled", TeamA: []*playerDomain.Player{p2}, TeamB: []*playerDomain.Player{p3}},
			{ID: "ko2", Stage: "r16", Status: "scheduled", TeamA: []*playerDomain.Player{p3}, TeamB: []*playerDomain.Player{p1}},
		},
	}
	mr := &forfeitMatchRepo{mockMatchRepo: &mockMatchRepo{}, matches: map[string]*tournamentDomain.Match{
		"ko2": {ID: "ko2", Status: "scheduled", TeamA: []*playerDomain.Player{p3}, TeamB: []*playerDomain.Player{p1}},
		// already decided group match is skipped
		"p1-p3": {ID: "p1-p3", Status: "finished", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p3}},
	}}
	w := &fakeWalkover{}
	uc := NewForfeitParticipantUseCase(&mockRepo{events: map[string]*tournamentDomain.Event{"e1": ev}}, mr, w)

	if err := uc.Execute(context.Background(), "e1", "p1"); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, c := range w.calls {
		got[c] = true
	}
	if len(got) != 2 || !got["p1-p2:A"] || !got["ko2:B"] {
		t.Errorf("unexpected walkovers: %v", w.calls)
	}

	ev.Status = "finished"
	if err := uc.Execute(context.Background(), "e1", "p1"); err == nil {
		t.Error("expected error for finished event")
	}
	if err := uc.Execute(context.Background(), "missing", "p1"); err == nil {
		t.Error("expected error for unknown event")
	}
}

func TestUnassign(t *testing.T) {
	p1, p2 := &playerDomain.Player{ID: "p1"}, &playerDomain.Player{ID: "p2"}
	ev := &tournamentDomain.Event{ID: "e1", Groups: []tournamentDomain.Group{
		{ID: "g1", Players: []*playerDomain.Player{p1}}, {ID: "g2", Players: []*playerDomain.Player{p2}}}}
	uc := NewMovePlayerUseCase(&mockRepo{events: map[string]*tournamentDomain.Event{"e1": ev}})
	if err := uc.Unassign(context.Background(), "e1", "p1"); err != nil {
		t.Fatal(err)
	}
	if len(ev.Groups[0].Players) != 0 || len(ev.Groups[1].Players) != 1 {
		t.Errorf("only p1 should be removed: %+v", ev.Groups)
	}
	if uc.Unassign(context.Background(), "e1", "p1") == nil {
		t.Error("expected error when player not in a group")
	}
	if uc.Unassign(context.Background(), "nope", "p1") == nil {
		t.Error("expected error for unknown event")
	}
}
