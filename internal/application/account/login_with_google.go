package account

import (
	"context"

	"table-tennis-backend/internal/domain/account"
	"table-tennis-backend/internal/domain/idgen"
)

// LoginWithGoogleCommand carries the Google identity resolved by the OAuth
// callback (see internal/infrastructure/oauth.GoogleUserInfo).
type LoginWithGoogleCommand struct {
	GoogleSub  string
	Email      string
	Name       string
	PictureURL string
}

// LoginWithGoogleUseCase gets-or-creates an Account by Google sub, so repeat
// logins are idempotent, and keeps the profile fields (name/picture, which
// come authoritatively from Google) fresh on every login.
type LoginWithGoogleUseCase struct {
	accountRepo account.Repository
}

func NewLoginWithGoogleUseCase(accountRepo account.Repository) *LoginWithGoogleUseCase {
	return &LoginWithGoogleUseCase{accountRepo: accountRepo}
}

func (uc *LoginWithGoogleUseCase) Execute(ctx context.Context, cmd LoginWithGoogleCommand) (*account.Account, error) {
	existing, err := uc.accountRepo.GetByGoogleSub(ctx, cmd.GoogleSub)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		existing.Email = cmd.Email
		existing.Name = cmd.Name
		existing.PictureURL = cmd.PictureURL
		if err := uc.accountRepo.Save(ctx, existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	a, err := account.NewAccount(idgen.Generate(), cmd.GoogleSub, cmd.Email, cmd.Name, cmd.PictureURL)
	if err != nil {
		return nil, err
	}
	if err := uc.accountRepo.Save(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}
