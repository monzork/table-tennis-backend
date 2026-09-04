package tournament_test

import (
	"context"
	"testing"

	"table-tennis-backend/internal/application/tournament"
	subTourneyDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/inactivity"
	playerDomain "table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
)

type mockSettingsRepo struct {
	settings *inactivity.Settings
}

func (m *mockSettingsRepo) Get(ctx context.Context) (*inactivity.Settings, error) {
	return m.settings, nil
}

func (m *mockSettingsRepo) Update(ctx context.Context, s *inactivity.Settings) error {
	m.settings = s
	return nil
}

func defaultSettingsRepo() *mockSettingsRepo {
	return &mockSettingsRepo{settings: &inactivity.Settings{TournamentThreshold: 4, EloPenalty: 50}}
}

func TestApplyInactivityDecayUseCase_ExecuteForEvent(t *testing.T) {
	t.Run("nil tournament id is a no-op", func(t *testing.T) {
		uc := tournament.NewApplyInactivityDecayUseCase(newMockEventRepo(), newMockPlayerRepo(), defaultSettingsRepo())
		if err := uc.ExecuteForEvent(context.Background(), nil); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("non-federation-endorsed tournament never increments a non-participant's streak", func(t *testing.T) {
		repo := newMockEventRepo()
		id := "t1"
		repo.events[id] = &tournamentDomain.Tournament{ID: id, FederationEndorsed: false, Events: []*subTourneyDomain.Event{{Status: "finished"}}}
		playerRepo := newMockPlayerRepo()
		p := &playerDomain.Player{ID: "p1", SinglesElo: 1000}
		playerRepo.players["p1"] = p

		uc := tournament.NewApplyInactivityDecayUseCase(repo, playerRepo, defaultSettingsRepo())
		if err := uc.ExecuteForEvent(context.Background(), &id); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if p.MissedFederatedTournaments != 0 {
			t.Errorf("expected no change for a non-endorsed tournament, got %d", p.MissedFederatedTournaments)
		}
	})

	t.Run("non-federation-endorsed tournament resets only its own enrolled players", func(t *testing.T) {
		repo := newMockEventRepo()
		id := "t1"
		floor := int16(1700)
		enrolledElsewhere := &playerDomain.Player{ID: "played-local", SinglesElo: 1859, MissedFederatedTournaments: 12, Inactive: true, FloorSingles: &floor}
		untouched := &playerDomain.Player{ID: "elsewhere", SinglesElo: 1600, MissedFederatedTournaments: 3, Inactive: false}
		repo.events[id] = &tournamentDomain.Tournament{
			ID: id, FederationEndorsed: false,
			Events: []*subTourneyDomain.Event{{Status: "finished", Participants: []*playerDomain.Player{enrolledElsewhere}}},
		}
		playerRepo := newMockPlayerRepo()
		playerRepo.players[enrolledElsewhere.ID] = enrolledElsewhere
		playerRepo.players[untouched.ID] = untouched

		uc := tournament.NewApplyInactivityDecayUseCase(repo, playerRepo, defaultSettingsRepo())
		if err := uc.ExecuteForEvent(context.Background(), &id); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if enrolledElsewhere.MissedFederatedTournaments != 0 || enrolledElsewhere.Inactive || enrolledElsewhere.FloorSingles != nil {
			t.Errorf("expected the local-tournament player's streak/flag/floor reset, got %+v", enrolledElsewhere)
		}
		// Elo itself is never touched by a reset -- only the tracking state.
		if enrolledElsewhere.SinglesElo != 1859 {
			t.Errorf("expected elo untouched by a reset, got %d", enrolledElsewhere.SinglesElo)
		}
		if untouched.MissedFederatedTournaments != 3 || untouched.Inactive {
			t.Errorf("expected a player who didn't play this local tournament left untouched, got %+v", untouched)
		}
	})

	t.Run("local tournament with elo skipped resets nothing: it's not a scored result", func(t *testing.T) {
		repo := newMockEventRepo()
		id := "t1"
		floor := int16(1700)
		p := &playerDomain.Player{ID: "p1", SinglesElo: 1859, MissedFederatedTournaments: 12, Inactive: true, FloorSingles: &floor}
		repo.events[id] = &tournamentDomain.Tournament{
			ID: id, FederationEndorsed: false, SkipElo: true,
			Events: []*subTourneyDomain.Event{{Status: "finished", Participants: []*playerDomain.Player{p}}},
		}
		playerRepo := newMockPlayerRepo()
		playerRepo.players[p.ID] = p

		uc := tournament.NewApplyInactivityDecayUseCase(repo, playerRepo, defaultSettingsRepo())
		if err := uc.ExecuteForEvent(context.Background(), &id); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if p.MissedFederatedTournaments != 12 || !p.Inactive || p.FloorSingles == nil {
			t.Errorf("expected an elo-skipped tournament to reset nothing, got %+v", p)
		}
	})

	t.Run("skipped while any child event is still unfinished", func(t *testing.T) {
		repo := newMockEventRepo()
		id := "t1"
		repo.events[id] = &tournamentDomain.Tournament{
			ID: id, FederationEndorsed: true,
			Events: []*subTourneyDomain.Event{{Status: "finished"}, {Status: "in_progress"}},
		}
		playerRepo := newMockPlayerRepo()
		p := &playerDomain.Player{ID: "p1", SinglesElo: 1000}
		playerRepo.players["p1"] = p

		uc := tournament.NewApplyInactivityDecayUseCase(repo, playerRepo, defaultSettingsRepo())
		if err := uc.ExecuteForEvent(context.Background(), &id); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if p.MissedFederatedTournaments != 0 {
			t.Errorf("expected no change while the tournament hasn't fully concluded, got %d", p.MissedFederatedTournaments)
		}
	})

	t.Run("non-participants accumulate a miss; participants reset", func(t *testing.T) {
		repo := newMockEventRepo()
		id := "t1"
		enrolled := &playerDomain.Player{ID: "enrolled", SinglesElo: 1000, MissedFederatedTournaments: 2, Inactive: true}
		absent := &playerDomain.Player{ID: "absent", SinglesElo: 2000, MissedFederatedTournaments: 3}
		repo.events[id] = &tournamentDomain.Tournament{
			ID: id, FederationEndorsed: true,
			Events: []*subTourneyDomain.Event{{Status: "finished", Participants: []*playerDomain.Player{enrolled}}},
		}
		playerRepo := newMockPlayerRepo()
		playerRepo.players[enrolled.ID] = enrolled
		playerRepo.players[absent.ID] = absent

		uc := tournament.NewApplyInactivityDecayUseCase(repo, playerRepo, defaultSettingsRepo())
		if err := uc.ExecuteForEvent(context.Background(), &id); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if enrolled.MissedFederatedTournaments != 0 || enrolled.Inactive {
			t.Errorf("expected enrolled player's streak reset, got missed=%d inactive=%v", enrolled.MissedFederatedTournaments, enrolled.Inactive)
		}
		// absent crosses from 3 to 4 (the configured threshold): flagged
		// inactive and docked the configured penalty.
		if absent.MissedFederatedTournaments != 4 {
			t.Errorf("expected absent player's streak at 4, got %d", absent.MissedFederatedTournaments)
		}
		if !absent.Inactive {
			t.Errorf("expected absent player flagged inactive at the threshold")
		}
		if absent.SinglesElo != 1950 {
			t.Errorf("expected absent player docked the configured 50 points, got %d", absent.SinglesElo)
		}
		if absent.FloorSingles == nil || *absent.FloorSingles != 1900 {
			t.Errorf("expected absent player's floor fixed at band-relative 1900 (2000's band minus 100), got %+v", absent.FloorSingles)
		}
	})

	t.Run("elo floor is relative to each player's own rating band", func(t *testing.T) {
		// Matches the spec examples: 2101 -> 2000, 2436 -> 2300, 1859 -> 1700.
		cases := []struct {
			startElo  int16
			wantFloor int16
		}{
			{2101, 2000},
			{2436, 2300},
			{1859, 1700},
		}
		for _, tc := range cases {
			repo := newMockEventRepo()
			id := "t1"
			repo.events[id] = &tournamentDomain.Tournament{ID: id, FederationEndorsed: true, Events: []*subTourneyDomain.Event{{Status: "finished"}}}
			p := &playerDomain.Player{ID: "p1", SinglesElo: tc.startElo, MissedFederatedTournaments: 3}
			playerRepo := newMockPlayerRepo()
			playerRepo.players[p.ID] = p

			uc := tournament.NewApplyInactivityDecayUseCase(repo, playerRepo, defaultSettingsRepo())
			if err := uc.ExecuteForEvent(context.Background(), &id); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if p.FloorSingles == nil || *p.FloorSingles != tc.wantFloor {
				t.Errorf("elo %d: expected floor %d, got %+v", tc.startElo, tc.wantFloor, p.FloorSingles)
			}
			if int(p.SinglesElo) != int(tc.startElo)-50 {
				t.Errorf("elo %d: expected the first decay step to just apply the penalty (%d), got %d", tc.startElo, tc.startElo-50, p.SinglesElo)
			}
		}
	})

	t.Run("elo decay never drops a player below their fixed band floor, and it doesn't keep sliding down", func(t *testing.T) {
		repo := newMockEventRepo()
		id := "t1"
		repo.events[id] = &tournamentDomain.Tournament{
			ID: id, FederationEndorsed: true,
			Events: []*subTourneyDomain.Event{{Status: "finished"}},
		}
		// Band 1800-1899 floors at 1700 (matches the 1859 -> 1700 example).
		p := &playerDomain.Player{ID: "p1", SinglesElo: 1859, MissedFederatedTournaments: 3}
		playerRepo := newMockPlayerRepo()
		playerRepo.players[p.ID] = p

		uc := tournament.NewApplyInactivityDecayUseCase(repo, playerRepo, defaultSettingsRepo())

		// Four separate threshold crossings (missed counts 4, 8, 12, 16):
		// 1859 -> 1809 -> 1759 -> 1709 -> clamped to the 1700 floor.
		wantAfterCrossing := []int16{1809, 1759, 1709, 1700}
		for i, want := range wantAfterCrossing {
			p.MissedFederatedTournaments = int16(4*i + 3) // one below the next multiple of 4
			if err := uc.ExecuteForEvent(context.Background(), &id); err != nil {
				t.Fatalf("crossing %d: expected no error, got %v", i+1, err)
			}
			if p.SinglesElo != want {
				t.Errorf("crossing %d: expected elo %d, got %d", i+1, want, p.SinglesElo)
			}
			if p.FloorSingles == nil || *p.FloorSingles != 1700 {
				t.Errorf("crossing %d: expected the floor to stay fixed at 1700, got %+v", i+1, p.FloorSingles)
			}
		}
		// 1859 - 1700 = 159 lost in total across the four crossings above
		// (50 + 50 + 50 + 9, the last one clamped).
		if p.LostToInactivitySingles != 159 {
			t.Errorf("expected 159 total elo lost, got %d", p.LostToInactivitySingles)
		}

		// A fifth crossing costs nothing further: already at the floor.
		p.MissedFederatedTournaments = 19
		if err := uc.ExecuteForEvent(context.Background(), &id); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if p.SinglesElo != 1700 {
			t.Errorf("expected elo to stay at the floor once reached, got %d", p.SinglesElo)
		}
		if p.LostToInactivitySingles != 159 {
			t.Errorf("expected no further loss once at the floor, got %d", p.LostToInactivitySingles)
		}
	})

	t.Run("re-enrolling resets the lost-elo tally alongside the rest of the streak", func(t *testing.T) {
		repo := newMockEventRepo()
		id := "t1"
		enrolled := &playerDomain.Player{
			ID: "p1", SinglesElo: 1809, MissedFederatedTournaments: 4, Inactive: true,
			LostToInactivitySingles: 50,
		}
		floor := int16(1700)
		enrolled.FloorSingles = &floor
		repo.events[id] = &tournamentDomain.Tournament{
			ID: id, FederationEndorsed: true,
			Events: []*subTourneyDomain.Event{{Status: "finished", Participants: []*playerDomain.Player{enrolled}}},
		}
		playerRepo := newMockPlayerRepo()
		playerRepo.players[enrolled.ID] = enrolled

		uc := tournament.NewApplyInactivityDecayUseCase(repo, playerRepo, defaultSettingsRepo())
		if err := uc.ExecuteForEvent(context.Background(), &id); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if enrolled.LostToInactivitySingles != 0 {
			t.Errorf("expected lost-elo tally reset on re-enrollment, got %d", enrolled.LostToInactivitySingles)
		}
	})

	t.Run("settings error propagates", func(t *testing.T) {
		repo := newMockEventRepo()
		id := "t1"
		repo.events[id] = &tournamentDomain.Tournament{ID: id, FederationEndorsed: true, Events: []*subTourneyDomain.Event{{Status: "finished"}}}
		uc := tournament.NewApplyInactivityDecayUseCase(repo, newMockPlayerRepo(), &errSettingsRepo{})
		if err := uc.ExecuteForEvent(context.Background(), &id); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("get tournament error propagates", func(t *testing.T) {
		repo := newMockEventRepo()
		id := "missing"
		uc := tournament.NewApplyInactivityDecayUseCase(repo, newMockPlayerRepo(), defaultSettingsRepo())
		if err := uc.ExecuteForEvent(context.Background(), &id); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

type errSettingsRepo struct{}

func (m *errSettingsRepo) Get(ctx context.Context) (*inactivity.Settings, error) {
	return nil, context.DeadlineExceeded
}
func (m *errSettingsRepo) Update(ctx context.Context, s *inactivity.Settings) error { return nil }
