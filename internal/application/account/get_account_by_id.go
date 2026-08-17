package account

import (
	"context"

	"table-tennis-backend/internal/domain/account"
)

type GetAccountByIDUseCase struct {
	accountRepo account.Repository
}

func NewGetAccountByIDUseCase(accountRepo account.Repository) *GetAccountByIDUseCase {
	return &GetAccountByIDUseCase{accountRepo: accountRepo}
}

func (uc *GetAccountByIDUseCase) Execute(ctx context.Context, id string) (*account.Account, error) {
	return uc.accountRepo.GetByID(ctx, id)
}
