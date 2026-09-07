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
	SaveMultiple(ctx context.Context, players []*Player) error
	// UpdateElo writes only the singles_elo/doubles_elo columns for each
	// given player, leaving every other field untouched — unlike
	// Save/SaveMultiple, which rewrite the whole row and so require a
	// fully-hydrated Player or they'll blank out fields the caller didn't
	// set.
	UpdateElo(ctx context.Context, players []*Player) error
	// UpdateEloForCategory writes only the two Elo columns for the given age
	// category ("open" writes singles_elo/doubles_elo, same columns as
	// UpdateElo; "u11".."u19" write that bracket's own pair) for each given
	// player, leaving every other field -- including every other age
	// category's Elo -- untouched.
	UpdateEloForCategory(ctx context.Context, ageCategory string, players []*Player) error
	// UpdateInactivity writes singles_elo/doubles_elo/missed_federated_tournaments/inactive
	// for each given player -- the inactivity-decay pass mutates Elo and the
	// tracking counters together in one pass.
	UpdateInactivity(ctx context.Context, players []*Player) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, query string) ([]*Player, error)
	SearchForSelection(ctx context.Context, query, gender string) ([]*Player, error)
	GetAll(ctx context.Context) ([]*Player, error)
	GetAllSingles(ctx context.Context) ([]*Player, error)
	GetAllDoubles(ctx context.Context) ([]*Player, error)
	GetSinglesByGender(ctx context.Context, gender string) ([]*Player, error)
	GetDoublesByGender(ctx context.Context, gender string) ([]*Player, error)
	GetByGuardianAccountID(ctx context.Context, accountID string) ([]*Player, error)
}

var ErrInvalidName = errors.New("first and last name required")

type Player struct {
	ID             string
	FirstName      string
	SecondName     string
	LastName       string
	SecondLastName string
	Birthdate      time.Time
	Gender         string
	SinglesElo     int16
	DoublesElo     int16
	Country        string
	Department     string
	WhatsAppNumber string
	NationalID     string
	IDFrontPath    string
	IDBackPath     string
	// GuardianAccountID links this player to the Account (parent/guardian)
	// responsible for it. Nil for every admin-created/self-registered player
	// that hasn't been linked by an admin — see NewGuardianChildPlayer and
	// the admin-assisted linking flow.
	GuardianAccountID *string
	// ClaimedByAccountID is set when an Account requests to be linked to this
	// player (self-claim) and cleared once an admin approves (promoting it to
	// GuardianAccountID) or rejects it. Nil means no claim is pending.
	ClaimedByAccountID *string
	// MissedFederatedTournaments counts consecutive federation-endorsed
	// tournaments concluded without this player enrolling in any of their
	// events. Reset to 0 the moment the player enrolls in one again. See
	// application/tournament.ApplyInactivityDecayUseCase.
	MissedFederatedTournaments int16
	// Inactive is set once MissedFederatedTournaments reaches the configured
	// threshold, and cleared again as soon as the player re-enrolls.
	Inactive bool
	// FloorSingles/FloorDoubles are the elo targets inactivity decay is
	// eroding this player's rating towards (see inactivity.BandFloor),
	// fixed the moment each rating first goes inactive so it doesn't keep
	// sliding down band by band on every later missed tournament. Nil
	// means no decay floor is currently in effect for that rating; both
	// are cleared the moment the player re-enrolls.
	FloorSingles *int16
	FloorDoubles *int16
	// LostToInactivitySingles/Doubles are how many Elo points this rating
	// has shed to inactivity decay since the current streak began -- reset
	// to 0 alongside MissedFederatedTournaments/Inactive/Floor* the moment
	// the player enrolls in a tournament again.
	LostToInactivitySingles int16
	LostToInactivityDoubles int16
	// SinglesEloU11/DoublesEloU11 .. SinglesEloU19/DoublesEloU19 are this
	// player's ratings within each youth age bracket, entirely separate
	// pools from SinglesElo/DoublesElo (which remain the Open/adult rating).
	// See EloFor/UpdateEloFor for the age-category-aware accessors, and
	// event.IsAgeEligible for who may play in which bracket.
	SinglesEloU11 int16
	DoublesEloU11 int16
	SinglesEloU13 int16
	DoublesEloU13 int16
	SinglesEloU15 int16
	DoublesEloU15 int16
	SinglesEloU19 int16
	DoublesEloU19 int16
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
		SinglesEloU11:  1000,
		DoublesEloU11:  1000,
		SinglesEloU13:  1000,
		DoublesEloU13:  1000,
		SinglesEloU15:  1000,
		DoublesEloU15:  1000,
		SinglesEloU19:  1000,
		DoublesEloU19:  1000,
		Country:        country,
		Department:     department,
		WhatsAppNumber: "",
		NationalID:     nationalID,
	}, nil
}

// NewGuardianChildPlayer creates a Player explicitly linked to a guardian
// Account, for the account-holder-adds-their-child flow. This is a separate
// constructor from NewPlayer (whose signature and every existing call site
// stay untouched) so every admin-created/self-registered player keeps
// GuardianAccountID == nil unless an admin later links it explicitly.
func NewGuardianChildPlayer(id, guardianAccountID, firstName, lastName string, birthdate time.Time, gender, country, department string) (*Player, error) {
	if firstName == "" || lastName == "" {
		return nil, ErrInvalidName
	}
	if gender == "" {
		gender = "M"
	}
	gID := guardianAccountID
	return &Player{
		ID:                id,
		FirstName:         firstName,
		LastName:          lastName,
		Birthdate:         birthdate,
		Gender:            gender,
		SinglesElo:        1000,
		DoublesElo:        1000,
		SinglesEloU11:     1000,
		DoublesEloU11:     1000,
		SinglesEloU13:     1000,
		DoublesEloU13:     1000,
		SinglesEloU15:     1000,
		DoublesEloU15:     1000,
		SinglesEloU19:     1000,
		DoublesEloU19:     1000,
		Country:           country,
		Department:        department,
		WhatsAppNumber:    "",
		GuardianAccountID: &gID,
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

// EloFor returns this player's rating for the given age category ("",
// "open", "u11", "u13", "u15", or "u19") and rank type ("singles" or
// anything else, which is treated as "doubles" -- matching the convention
// already used by e.g. Match.MatchType throughout this codebase). An
// unrecognized or empty age category falls back to the Open rating
// (SinglesElo/DoublesElo), so every existing caller that never deals with
// age categories keeps working unchanged.
func (p *Player) EloFor(ageCategory, rankType string) int16 {
	singles := rankType != "doubles"
	switch ageCategory {
	case "u11":
		if singles {
			return p.SinglesEloU11
		}
		return p.DoublesEloU11
	case "u13":
		if singles {
			return p.SinglesEloU13
		}
		return p.DoublesEloU13
	case "u15":
		if singles {
			return p.SinglesEloU15
		}
		return p.DoublesEloU15
	case "u19":
		if singles {
			return p.SinglesEloU19
		}
		return p.DoublesEloU19
	default: // "", "open", or anything unrecognized
		if singles {
			return p.SinglesElo
		}
		return p.DoublesElo
	}
}

// UpdateEloFor sets this player's rating for the given age category and
// rank type, mirroring UpdateSinglesElo/UpdateDoublesElo's non-negative
// guard. See EloFor for the age-category/rank-type conventions.
func (p *Player) UpdateEloFor(ageCategory, rankType string, newElo int16) {
	if newElo < 0 {
		return
	}
	singles := rankType != "doubles"
	switch ageCategory {
	case "u11":
		if singles {
			p.SinglesEloU11 = newElo
		} else {
			p.DoublesEloU11 = newElo
		}
	case "u13":
		if singles {
			p.SinglesEloU13 = newElo
		} else {
			p.DoublesEloU13 = newElo
		}
	case "u15":
		if singles {
			p.SinglesEloU15 = newElo
		} else {
			p.DoublesEloU15 = newElo
		}
	case "u19":
		if singles {
			p.SinglesEloU19 = newElo
		} else {
			p.DoublesEloU19 = newElo
		}
	default:
		if singles {
			p.SinglesElo = newElo
		} else {
			p.DoublesElo = newElo
		}
	}
}

func (p *Player) FullName() string {
	return p.FirstName + " " + p.LastName
}

func (p *Player) FirstNameWithSecond() string {
	if p.SecondName != "" {
		return p.FirstName + " " + p.SecondName
	}
	return p.FirstName
}

func (p *Player) LastNameWithSecond() string {
	if p.SecondLastName != "" {
		return p.LastName + " " + p.SecondLastName
	}
	return p.LastName
}
