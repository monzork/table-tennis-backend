package account_test

import (
	"context"
	"errors"

	"table-tennis-backend/internal/domain/account"
	tournamentEvent "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
)

// ─── fake account.Repository ────────────────────────────────────────────────

type fakeAccountRepo struct {
	byID     map[string]*account.Account
	byGoogle map[string]*account.Account
	byEmail  map[string]*account.Account
	saveErr  error
	getIDErr error
}

func newFakeAccountRepo() *fakeAccountRepo {
	return &fakeAccountRepo{
		byID:     make(map[string]*account.Account),
		byGoogle: make(map[string]*account.Account),
		byEmail:  make(map[string]*account.Account),
	}
}

func (f *fakeAccountRepo) GetByID(ctx context.Context, id string) (*account.Account, error) {
	if f.getIDErr != nil {
		return nil, f.getIDErr
	}
	a, ok := f.byID[id]
	if !ok {
		return nil, errors.New("account not found")
	}
	return a, nil
}

func (f *fakeAccountRepo) GetByGoogleSub(ctx context.Context, sub string) (*account.Account, error) {
	return f.byGoogle[sub], nil
}

func (f *fakeAccountRepo) GetByEmail(ctx context.Context, email string) (*account.Account, error) {
	return f.byEmail[email], nil
}

func (f *fakeAccountRepo) Save(ctx context.Context, a *account.Account) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.byID[a.ID] = a
	f.byGoogle[a.GoogleSub] = a
	f.byEmail[a.Email] = a
	return nil
}

// ─── fake player.Repository ─────────────────────────────────────────────────

type fakePlayerRepo struct {
	player.Repository // embed nil interface: unused methods panic if called
	players           map[string]*player.Player
	saveErr           error
}

func newFakePlayerRepo() *fakePlayerRepo {
	return &fakePlayerRepo{players: make(map[string]*player.Player)}
}

func (f *fakePlayerRepo) GetById(ctx context.Context, id string) (*player.Player, error) {
	p, ok := f.players[id]
	if !ok {
		return nil, errors.New("player not found")
	}
	return p, nil
}

func (f *fakePlayerRepo) Save(ctx context.Context, p *player.Player) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.players[p.ID] = p
	return nil
}

func (f *fakePlayerRepo) GetByGuardianAccountID(ctx context.Context, accountID string) ([]*player.Player, error) {
	var result []*player.Player
	for _, p := range f.players {
		if p.GuardianAccountID != nil && *p.GuardianAccountID == accountID {
			result = append(result, p)
		}
	}
	return result, nil
}

// ─── fake event.Repository (for GetGuardianPendingMatchesUseCase via
// playerApp.GetPlayerPendingMatchesUseCase) ─────────────────────────────────

type fakeEventRepo struct {
	tournamentEvent.Repository
	eventsByPlayer map[string][]*tournamentEvent.Event
	err            error
}

func newFakeEventRepo() *fakeEventRepo {
	return &fakeEventRepo{eventsByPlayer: make(map[string][]*tournamentEvent.Event)}
}

func (f *fakeEventRepo) GetByParticipantID(ctx context.Context, playerID string) ([]*tournamentEvent.Event, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.eventsByPlayer[playerID], nil
}
