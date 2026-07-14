package event

import (
	"fmt"
	"sort"
	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/domain/player"
)

type SeedingStrategy interface {
	AssignGroups(t *Event) error
}

type OpenBracketSnakeSeeder struct{}

func (s *OpenBracketSnakeSeeder) AssignGroups(t *Event) error {
	if t.Format != "groups_elimination" && t.Format != "round_robin" {
		return nil
	}
	// Determine units to group (players or teams)
	var units []*player.Player
	if t.Type == "teams" || t.Type == "doubles" || t.Type == "mixed_doubles" {
		units = make([]*player.Player, len(t.Teams))
		for i, team := range t.Teams {
			avgElo := team.AverageElo(t.Type)
			units[i] = &player.Player{
				ID:         team.ID,
				FirstName:  team.Name,
				LastName:   " (Team)",
				SinglesElo: avgElo,
				DoublesElo: avgElo,
			}
		}
	} else {
		units = make([]*player.Player, len(t.Participants))
		copy(units, t.Participants)
	}

	n := len(units)
	if n == 0 {
		return nil
	}

	// Sort participants/teams by Elo (descending)
	sort.Slice(units, func(i, j int) bool {
		if t.Type == "doubles" || t.Type == "mixed_doubles" {
			return units[i].DoublesElo > units[j].DoublesElo
		}
		return units[i].SinglesElo > units[j].SinglesElo
	})

	if t.Format == "round_robin" {
		t.Groups = []Group{
			{
				ID:           idgen.Generate(),
				TournamentID: t.ID,
				Name:         "All Against All",
				Players:      units, // Everyone in one single group
			},
		}
		return nil
	}

	// WTT standard: groups of 3 or 4.
	// Let's aim for groups of 4 if possible, otherwise 3.
	numGroups := n / 4
	if n%4 != 0 {
		numGroups++
	}

	t.Groups = make([]Group, numGroups)
	for i := 0; i < numGroups; i++ {
		t.Groups[i] = Group{
			ID:           idgen.Generate(),
			TournamentID: t.ID,
			Name:         fmt.Sprintf("Group %c", 'A'+i),
			Players:      []*player.Player{},
		}
	}

	// Snake seeding
	for i, p := range units {
		groupIndex := i % numGroups
		// In snake seeding:
		// Row 0: 0, 1, 2, 3
		// Row 1: 7, 6, 5, 4
		row := i / numGroups
		if row%2 != 0 {
			groupIndex = numGroups - 1 - groupIndex
		}
		t.Groups[groupIndex].Players = append(t.Groups[groupIndex].Players, p)
	}

	return nil
}

type DivisionSeeding struct {
	ID     string
	Name   string
	MinElo int16
	MaxElo *int16
}

type DivisionSeeder struct {
	Divisions []DivisionSeeding
}

func (s *DivisionSeeder) AssignGroups(t *Event) error {
	if t.Format != "groups_elimination" && t.Format != "round_robin" && t.Format != "elimination" {
		t.Groups = []Group{}
		return nil
	}

	// 1. Group participants by division
	type DivGroup struct {
		DivisionID string
		Name       string
		Players    []*player.Player
	}

	var divGroups []DivGroup

	// Determine units to group (players or teams)
	var units []*player.Player
	if t.Type == "teams" || t.Type == "doubles" || t.Type == "mixed_doubles" {
		units = make([]*player.Player, len(t.Teams))
		for i, team := range t.Teams {
			avgElo := team.AverageElo(t.Type)
			units[i] = &player.Player{
				ID:         team.ID,
				FirstName:  team.Name,
				LastName:   " (Team)",
				SinglesElo: avgElo,
				DoublesElo: avgElo,
			}
		}
	} else {
		units = make([]*player.Player, len(t.Participants))
		copy(units, t.Participants)
	}

	if t.SkipElo || len(s.Divisions) == 0 {
		divGroups = append(divGroups, DivGroup{
			Name:    "Open Bracket",
			Players: units,
		})
	} else {
		assigned := make(map[string]bool)
		for _, d := range s.Divisions {
			var dPlayers []*player.Player
			for _, p := range units {
				if assigned[p.ID] {
					continue
				}
				elo := p.SinglesElo
				if t.Type == "doubles" || t.Type == "mixed_doubles" {
					elo = p.DoublesElo
				}
				if elo >= d.MinElo && (d.MaxElo == nil || elo <= *d.MaxElo) {
					dPlayers = append(dPlayers, p)
					assigned[p.ID] = true
				}
			}
			if len(dPlayers) > 0 {
				divGroups = append(divGroups, DivGroup{
					DivisionID: d.ID,
					Name:       d.Name,
					Players:    dPlayers,
				})
			}
		}

		var unassigned []*player.Player
		for _, p := range units {
			if !assigned[p.ID] {
				unassigned = append(unassigned, p)
			}
		}
		if len(unassigned) > 0 {
			divGroups = append(divGroups, DivGroup{
				Name:    "Unclassified",
				Players: unassigned,
			})
		}
	}

	t.Groups = []Group{}

	for _, dg := range divGroups {
		n := len(dg.Players)
		if n == 0 {
			continue
		}

		sort.Slice(dg.Players, func(i, j int) bool {
			if t.Type == "doubles" || t.Type == "mixed_doubles" {
				return dg.Players[i].DoublesElo > dg.Players[j].DoublesElo
			}
			return dg.Players[i].SinglesElo > dg.Players[j].SinglesElo
		})

		divFormat := t.GetDivisionFormat(dg.DivisionID)

		if divFormat == "round_robin" || divFormat == "elimination" {
			groupName := fmt.Sprintf("%s - Round Robin", dg.Name)
			if divFormat == "elimination" {
				groupName = fmt.Sprintf("%s - Bracket Draw", dg.Name)
			}
			t.Groups = append(t.Groups, Group{
				ID:           idgen.Generate(),
				TournamentID: t.ID,
				Name:         groupName,
				Players:      dg.Players,
			})
			continue
		}

		numGroups := t.GetDivisionGroupCount(dg.DivisionID)
		if numGroups <= 0 {
			numGroups = n / 4
			if n%4 != 0 {
				numGroups++
			}
		}

		divGroupsList := make([]Group, numGroups)
		for i := 0; i < numGroups; i++ {
			divGroupsList[i] = Group{
				ID:           idgen.Generate(),
				TournamentID: t.ID,
				Name:         fmt.Sprintf("%s - Group %c", dg.Name, 'A'+i),
				Players:      []*player.Player{},
			}
		}

		for i, p := range dg.Players {
			groupIndex := i % numGroups
			row := i / numGroups
			if row%2 != 0 {
				groupIndex = numGroups - 1 - groupIndex
			}
			divGroupsList[groupIndex].Players = append(divGroupsList[groupIndex].Players, p)
		}

		t.Groups = append(t.Groups, divGroupsList...)
	}

	return nil
}
