package account

import (
	"context"

	"table-tennis-backend/internal/domain/account"
)

// UpdateAccountUseCase updates the only Account field an accountholder is
// allowed to edit directly — Name. Email/picture come authoritatively from
// Google on every login.
type UpdateAccountUseCase struct {
	accountRepo account.Repository
}

func NewUpdateAccountUseCase(accountRepo account.Repository) *UpdateAccountUseCase {
	return &UpdateAccountUseCase{accountRepo: accountRepo}
}

func (uc *UpdateAccountUseCase) Execute(ctx context.Context, accountID, name string) (*account.Account, error) {
	a, err := uc.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	a.Name = name
	if err := uc.accountRepo.Save(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}
