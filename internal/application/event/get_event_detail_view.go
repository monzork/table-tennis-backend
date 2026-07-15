package event

import (
	"context"
	"sort"
	"strings"
	"sync"

	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/domain/bracket"
	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
)

type EventDetailView struct {
	Event                 *tournamentDomain.Event
	Players               any
	Divisions             []*divisionDomain.Division
	BracketViewModel      *bracket.Bracket
	AvailableParticipants []*player.Player
	ParticipantRows       []ParticipantRow
	Officials             []tournamentDomain.ParticipantSnapshot
	PlayerPins            map[string]string
}

type ParticipantRow struct {
	Player    *player.Player
	Seed      int
	GroupName string
	DivName   string
	Pin       string
	Elo       int16
}

type GetEventDetailViewUseCase struct {
	getByID       *GetTournamentByIDUseCase
	leaderboardUC *leaderboard.GetLeaderboardUseCase
	divisionUC    *division.DivisionUseCase
}

func NewGetEventDetailViewUseCase(getByID *GetTournamentByIDUseCase, leaderboardUC *leaderboard.GetLeaderboardUseCase, divisionUC *division.DivisionUseCase) *GetEventDetailViewUseCase {
	return &GetEventDetailViewUseCase{
		getByID:       getByID,
		leaderboardUC: leaderboardUC,
		divisionUC:    divisionUC,
	}
}

func (uc *GetEventDetailViewUseCase) Execute(ctx context.Context, tournamentID string, statusFilter, playerSearch string) (*EventDetailView, error) {
	type result struct {
		event     *tournamentDomain.Event
		err       error
		players   any
		divisions []*divisionDomain.Division
		snapshots []tournamentDomain.ParticipantSnapshot
		officials []tournamentDomain.ParticipantSnapshot
	}
	var res result
	var wg sync.WaitGroup
	wg.Add(5)

	go func() {
		defer wg.Done()
		res.event, res.err = uc.getByID.Execute(ctx, tournamentID)
	}()
	go func() {
		defer wg.Done()
		res.players, _ = uc.leaderboardUC.ExecuteSingles(ctx)
	}()
	go func() {
		defer wg.Done()
		res.divisions, _ = uc.divisionUC.GetAll(ctx)
	}()
	go func() {
		defer wg.Done()
		res.snapshots, _ = uc.getByID.GetSnapshots(ctx, tournamentID)
	}()
	go func() {
		defer wg.Done()
		res.officials, _ = uc.getByID.GetOfficials(ctx, tournamentID)
	}()
	wg.Wait()

	if res.err != nil {
		return nil, res.err
	}

	t := res.event
	playerSearch = strings.ToLower(playerSearch)

	if statusFilter != "all" || playerSearch != "" {
		var filtered []tournamentDomain.Match
		for _, m := range t.Matches {
			matchStatus := statusFilter == "all" || m.Status == statusFilter
			matchPlayer := true

			if playerSearch != "" {
				matchPlayer = false
				for _, p := range m.TeamA {
					if strings.Contains(strings.ToLower(p.FirstName), playerSearch) || strings.Contains(strings.ToLower(p.LastName), playerSearch) {
						matchPlayer = true
						break
					}
				}
				if !matchPlayer {
					for _, p := range m.TeamB {
						if strings.Contains(strings.ToLower(p.FirstName), playerSearch) || strings.Contains(strings.ToLower(p.LastName), playerSearch) {
							matchPlayer = true
							break
						}
					}
				}
			}

			if matchStatus && matchPlayer {
				filtered = append(filtered, m)
			}
		}
		t.Matches = filtered
	}

	vm := bracket.BuildBracket(t, res.divisions, nil)

	var availableParticipants []*player.Player
	assignedMap := make(map[string]bool)
	for _, team := range t.Teams {
		for _, p := range team.Players {
			assignedMap[p.ID] = true
		}
	}
	for _, p := range t.Participants {
		if !assignedMap[p.ID] {
			availableParticipants = append(availableParticipants, p)
		}
	}

	playerPins := make(map[string]string)
	for _, snap := range res.snapshots {
		playerPins[snap.PlayerID] = snap.Pin
	}

	playerGroupMap := make(map[string]string)
	for _, g := range t.Groups {
		gDisplayName := g.Name
		if idx := strings.Index(g.Name, " - "); idx != -1 {
			gDisplayName = g.Name[idx+3:]
		}
		for _, p := range g.Players {
			playerGroupMap[p.ID] = gDisplayName
		}
	}

	playerDivMap := make(map[string]string)
	for _, p := range t.Participants {
		elo := p.SinglesElo
		if t.Type == "doubles" || t.Type == "mixed_doubles" {
			elo = p.DoublesElo
		}
		found := false
		for _, d := range res.divisions {
			if d.MinElo == 0 && d.MaxElo == nil {
				continue
			}
			if (d.Category == "both" || d.Category == t.Type) && elo >= d.MinElo && (d.MaxElo == nil || elo <= *d.MaxElo) {
				playerDivMap[p.ID] = d.Name
				found = true
				break
			}
		}
		if !found && !t.SkipElo {
			playerDivMap[p.ID] = "Open Bracket"
		}
	}

	sortedParts := make([]*player.Player, len(t.Participants))
	copy(sortedParts, t.Participants)
	sort.Slice(sortedParts, func(i, j int) bool {
		if t.Type == "doubles" || t.Type == "mixed_doubles" {
			return sortedParts[i].DoublesElo > sortedParts[j].DoublesElo
		}
		return sortedParts[i].SinglesElo > sortedParts[j].SinglesElo
	})
	seedMap := make(map[string]int, len(sortedParts))
	for i, p := range sortedParts {
		seedMap[p.ID] = i + 1
	}

	rows := make([]ParticipantRow, len(sortedParts))
	for i, p := range sortedParts {
		elo := p.SinglesElo
		if t.Type == "doubles" || t.Type == "mixed_doubles" {
			elo = p.DoublesElo
		}
		rows[i] = ParticipantRow{
			Player:    p,
			Seed:      seedMap[p.ID],
			GroupName: playerGroupMap[p.ID],
			DivName:   playerDivMap[p.ID],
			Pin:       playerPins[p.ID],
			Elo:       elo,
		}
	}

	return &EventDetailView{
		Event:                 t,
		Players:               res.players,
		Divisions:             res.divisions,
		BracketViewModel:      vm,
		AvailableParticipants: availableParticipants,
		ParticipantRows:       rows,
		Officials:             res.officials,
		PlayerPins:            playerPins,
	}, nil
}
