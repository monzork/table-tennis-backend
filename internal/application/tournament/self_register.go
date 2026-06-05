package tournament

import (
	"context"
	"errors"
	"fmt"
	"strings"
	playerDomain "table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
)

// SelfRegisterUseCase allows an existing player (identified by name + country)
// to self-register into a tournament that has RegistrationOpen == true.
type SelfRegisterUseCase struct {
	tournamentRepo *bun.TournamentRepository
	playerRepo     *bun.PlayerRepository
}

func NewSelfRegisterUseCase(
	tournamentRepo *bun.TournamentRepository,
	playerRepo *bun.PlayerRepository,
) *SelfRegisterUseCase {
	return &SelfRegisterUseCase{
		tournamentRepo: tournamentRepo,
		playerRepo:     playerRepo,
	}
}

// GetOpenTournaments returns all tournaments where RegistrationOpen == true.
func (uc *SelfRegisterUseCase) GetOpenTournaments(ctx context.Context) ([]*tournamentDomain.Tournament, error) {
	all, err := uc.tournamentRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	var open []*tournamentDomain.Tournament
	for _, t := range all {
		if t.RegistrationOpen && t.Status != "finished" {
			open = append(open, t)
		}
	}
	return open, nil
}

// Execute attempts to register an existing player into the specified tournament.
// The player is looked up by fuzzy name match + optional country filter.
func (uc *SelfRegisterUseCase) Execute(
	ctx context.Context,
	tournamentIDStr string,
	fullName string,
	country string,
) (*tournamentDomain.Tournament, string, error) {
	t, err := uc.tournamentRepo.GetByIDStr(ctx, tournamentIDStr)
	if err != nil {
		return nil, "", errors.New("tournament not found")
	}
	if !t.RegistrationOpen || t.Status == "finished" {
		return nil, "", errors.New("tournament is not open for registration")
	}

	// Search for the player by name (case-insensitive) and optional country
	nameLower := strings.ToLower(strings.TrimSpace(fullName))
	countryUpper := strings.ToUpper(strings.TrimSpace(country))

	players, err := uc.playerRepo.GetAll(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("could not search players: %w", err)
	}

	var matched *playerDomain.Player
	for _, p := range players {
		pName := strings.ToLower(p.FirstName + " " + p.LastName)
		pCountry := strings.ToUpper(p.Country)
		if strings.Contains(pName, nameLower) && (countryUpper == "" || pCountry == countryUpper) {
			matched = p
			break
		}
	}
	if matched == nil {
		return nil, "", fmt.Errorf("no player found matching '%s' from '%s'", fullName, country)
	}

	// Check if already a participant
	for _, p := range t.Participants {
		if p.ID == matched.ID {
			return nil, "", errors.New("player is already registered in this tournament")
		}
	}

	// Persist the new participant
	if err := uc.tournamentRepo.AddParticipant(ctx, t.ID, matched.ID, matched.SinglesElo, matched.DoublesElo); err != nil {
		return nil, "", fmt.Errorf("failed to register player: %w", err)
	}

	return t, matched.FirstName + " " + matched.LastName, nil
}
