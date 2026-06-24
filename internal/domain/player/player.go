package player

import (
	"context"
	"errors"
	"time"
)

type Repository interface {
	GetById(ctx context.Context, id string) (*Player, error)
	GetByIDs(ctx context.Context, ids []string) ([]*Player, error)
	Save(ctx context.Context, p *Player) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, query string) ([]*Player, error)
	GetAll(ctx context.Context) ([]*Player, error)
	GetAllSingles(ctx context.Context) ([]*Player, error)
	GetAllDoubles(ctx context.Context) ([]*Player, error)
	GetSinglesByGender(ctx context.Context, gender string) ([]*Player, error)
	GetDoublesByGender(ctx context.Context, gender string) ([]*Player, error)
}

var ErrInvalidName = errors.New("first and last name required")

type Player struct {
	ID             string
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
	NationalID     string
}

func NewPlayer(id, firstName, lastName string, birthdate time.Time, gender, country, department, nationalID string) (*Player, error) {
	if firstName == "" || lastName == "" {
		return nil, ErrInvalidName
	}
	if gender == "" {
		gender = "M"
	}
	return &Player{
		ID:             id,
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
		NationalID:     nationalID,
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
