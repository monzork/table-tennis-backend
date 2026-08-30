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
	if t.Format != "groups_elimination" && t.Format != "round_robin" && t.Format != "elimination" && t.Format != "single_division_multiple_brackets" {
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
				ID:      idgen.Generate(),
				EventID: t.ID,
				Name:    "All Against All",
				Players: units, // Everyone in one single group
			},
		}
		return nil
	}

	if t.Format == "elimination" {
		t.Groups = []Group{
			{
				ID:      idgen.Generate(),
				EventID: t.ID,
				Name:    "Bracket Draw",
				Players: units,
			},
		}
		return nil
	}

	var numGroups int
	if t.GroupCount > 0 {
		numGroups = t.GroupCount
		if numGroups > n {
			numGroups = n
		}
	} else {
		// WTT standard: groups of 3 or 4.
		// Let's aim for groups of 4 if possible, otherwise 3.
		numGroups = n / 4
		if n%4 != 0 {
			numGroups++
		}
	}

	t.Groups = make([]Group, numGroups)
	for i := 0; i < numGroups; i++ {
		t.Groups[i] = Group{
			ID:      idgen.Generate(),
			EventID: t.ID,
			Name:    fmt.Sprintf("Group %c", 'A'+i),
			Players: []*player.Player{},
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
