package models

import "github.com/uptrace/bun"

type User struct {
	bun.BaseModel `bun:"table:users"`
	ID            int64  `bun"id,pk,autoincrement"`
	Name          string `bun:"name,notnull"`
}
