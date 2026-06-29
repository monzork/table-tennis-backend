package tournament

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"table-tennis-backend/internal/domain/idgen"
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
	secondName string,
	lastName string,
	secondLastName string,
	country string,
	department string,
	whatsAppNumber string,
	birthdateStr string,
	gender string,
	nationalID string,
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
	secondLower := strings.ToLower(strings.TrimSpace(secondName))
	lastLower := strings.ToLower(lastName)
	secondLastLower := strings.ToLower(strings.TrimSpace(secondLastName))
	countryUpper := strings.ToUpper(strings.TrimSpace(country))

	players, err := uc.playerRepo.GetAll(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("could not search players: %w", err)
	}

	var matched *playerDomain.Player
	for _, p := range players {
		pFirst := strings.ToLower(p.FirstName)
		pSecond := strings.ToLower(p.SecondName)
		pLast := strings.ToLower(p.LastName)
		pSecondLast := strings.ToLower(p.SecondLastName)
		pCountry := strings.ToUpper(p.Country)
		if pFirst == firstLower && pSecond == secondLower && pLast == lastLower && pSecondLast == secondLastLower && (countryUpper == "" || pCountry == countryUpper) {
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

		newPlayer, err := playerDomain.NewPlayer(idgen.Generate(), firstName, lastName, birthdate, gender, countryUpper, strings.TrimSpace(department), strings.TrimSpace(nationalID))
		if err != nil {
			return nil, "", fmt.Errorf("failed to create new player: %w", err)
		}
		newPlayer.SecondName = strings.TrimSpace(secondName)
		newPlayer.SecondLastName = strings.TrimSpace(secondLastName)
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
