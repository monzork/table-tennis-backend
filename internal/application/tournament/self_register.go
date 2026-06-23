package tournament

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	playerDomain "table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
)

// SelfRegisterUseCase allows an existing player (identified by name + country)
// to self-register into a tournament that has RegistrationOpen == true.
type SelfRegisterUseCase struct {
	tournamentRepo tournamentDomain.Repository
	playerRepo     playerDomain.Repository
}

func NewSelfRegisterUseCase(
	tournamentRepo tournamentDomain.Repository,
	playerRepo playerDomain.Repository,
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
// The player is looked up by name match + optional country filter.
func (uc *SelfRegisterUseCase) Execute(
	ctx context.Context,
	tournamentIDStr string,
	firstName string,
	lastName string,
	country string,
	department string,
	whatsAppNumber string,
	birthdateStr string,
	gender string,
) (*tournamentDomain.Tournament, string, error) {
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	if firstName == "" || lastName == "" {
		return nil, "", errors.New("first and last name are required for registration")
	}

	t, err := uc.tournamentRepo.GetByID(ctx, tournamentIDStr)
	if err != nil {
		return nil, "", errors.New("tournament not found")
	}
	if !t.RegistrationOpen || t.Status == "finished" {
		return nil, "", errors.New("tournament is not open for registration")
	}

	// Search for the player by name (case-insensitive) and optional country
	firstLower := strings.ToLower(firstName)
	lastLower := strings.ToLower(lastName)
	countryUpper := strings.ToUpper(strings.TrimSpace(country))

	players, err := uc.playerRepo.GetAll(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("could not search players: %w", err)
	}

	var matched *playerDomain.Player
	for _, p := range players {
		pFirst := strings.ToLower(p.FirstName)
		pLast := strings.ToLower(p.LastName)
		pCountry := strings.ToUpper(p.Country)
		if pFirst == firstLower && pLast == lastLower && (countryUpper == "" || pCountry == countryUpper) {
			matched = p
			break
		}
	}
	if matched == nil {
		if gender == "" {
			gender = "M"
		}

		birthdate := time.Now()
		if birthdateStr != "" {
			if parsed, err := time.Parse("2006-01-02", birthdateStr); err == nil {
				birthdate = parsed
			}
		}

		newPlayer, err := playerDomain.NewPlayer(uuid.NewString(), firstName, lastName, birthdate, gender, countryUpper, strings.TrimSpace(department))
		if err != nil {
			return nil, "", fmt.Errorf("failed to create new player: %w", err)
		}
		newPlayer.WhatsAppNumber = strings.TrimSpace(whatsAppNumber)
		newPlayer.UpdateSinglesElo(500)
		newPlayer.UpdateDoublesElo(500)

		if err := uc.playerRepo.Save(ctx, newPlayer); err != nil {
			return nil, "", fmt.Errorf("failed to save new player: %w", err)
		}
		matched = newPlayer
	}

	// Check if already a participant
	for _, p := range t.Participants {
		if p.ID == matched.ID {
			return nil, "", errors.New("player is already registered in this tournament")
		}
	}

	// Persist the new participant via updating the aggregate root
	t.Participants = append(t.Participants, matched)
	if err := uc.tournamentRepo.Update(ctx, t); err != nil {
		return nil, "", fmt.Errorf("failed to register player: %w", err)
	}

	return t, matched.FirstName + " " + matched.LastName, nil
}
