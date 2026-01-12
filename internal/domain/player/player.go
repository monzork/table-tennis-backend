package player

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidName = errors.New("first and last name required")

type Player struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
	Birthdate time.Time
	Elo       int16
	Country   string
}

func NewPlayer(firstName, lastName string, birthdate time.Time, country string) (*Player, error) {
	if firstName == "" || lastName == "" {
		return nil, ErrInvalidName
	}
	return &Player{
		ID:        uuid.New(),
		FirstName: firstName,
		LastName:  lastName,
		Birthdate: birthdate,
		Elo:       1000,
		Country:   country,
	}, nil
}

func (p *Player) UpdateElo(newElo int16) {
	if newElo >= 0 {
		p.Elo = newElo
	}

}

func (p *Player) FullName() string {
	return p.FirstName + " " + p.LastName
}
