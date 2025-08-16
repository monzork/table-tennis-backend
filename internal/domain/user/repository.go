package user

import "context"

type Repository interface {
	Create(ctx context.Context, u *User) error
	Login(ctx context.Context, u *User) (*User, error)
}
