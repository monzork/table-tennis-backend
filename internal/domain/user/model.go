package user

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users"`
	ID            uuid.UUID `bun:"id,pk,type:uuid"`
	Username      string    `bun:"username,notnull"`
	Password      string    `bun:"password,notnull"`
	Created_at    time.Time `bun:"created_at,notnull,default:current_timestamp"`
	Updated_at    time.Time `bun:"updated_at,notnull,default:current_timestamp"`
}
