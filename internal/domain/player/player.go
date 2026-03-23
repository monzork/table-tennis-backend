package player

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidName = errors.New("first and last name required")

type Player struct {
	ID         uuid.UUID
	FirstName  string
	LastName   string
	Birthdate  time.Time
	Gender     string
	SinglesElo int16
	DoublesElo int16
	Country    string
}

func NewPlayer(firstName, lastName string, birthdate time.Time, gender, country string) (*Player, error) {
	if firstName == "" || lastName == "" {
		return nil, ErrInvalidName
	}
	if gender == "" {
		gender = "M"
	}
	return &Player{
		ID:         uuid.New(),
		FirstName:  firstName,
		LastName:   lastName,
		Birthdate:  birthdate,
		Gender:     gender,
		SinglesElo: 1000,
		DoublesElo: 1000,
		Country:    country,
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
