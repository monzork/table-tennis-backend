package player

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	GetById(ctx context.Context, id uuid.UUID) (*Player, error)
	Save(ctx context.Context, p *Player) error
}

var ErrInvalidName = errors.New("first and last name required")

type Player struct {
	ID             uuid.UUID
	FirstName      string
	LastName       string
	Birthdate      time.Time
	Gender         string
	SinglesElo     int16
	DoublesElo     int16
	Country        string
	Department     string
	WhatsAppNumber string
	Pin            string
}

func NewPlayer(firstName, lastName string, birthdate time.Time, gender, country, department string) (*Player, error) {
	if firstName == "" || lastName == "" {
		return nil, ErrInvalidName
	}
	if gender == "" {
		gender = "M"
	}
	return &Player{
		ID:             uuid.New(),
		FirstName:      firstName,
		LastName:       lastName,
		Birthdate:      birthdate,
		Gender:         gender,
		SinglesElo:     1000,
		DoublesElo:     1000,
		Country:        country,
		Department:     department,
		WhatsAppNumber: "",
		Pin:            "1234",
	}, nil
}

func (p *Player) UpdateSinglesElo(newElo int16) {
	if newElo >= 0 {
		p.SinglesElo = newElo
	}
}

func (p *Player) UpdateDoublesElo(newElo int16) {
	if newElo >= 0 {
		p.DoublesElo = newElo
	}
}

func (p *Player) FullName() string {
	return p.FirstName + " " + p.LastName
}
