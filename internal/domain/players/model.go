package players

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Players struct {
	bun.BaseModel `bun:"table:players"`
	ID            uuid.UUID `bun:"id,pk,type:uuid"`
	Name          string    `bun:"name,notnull"`
	Sex           string    `bun:"sex,notnull"`
	Country       string    `bun:"country,notnull"`
	City          string    `bun:"city,notnull"`
	Birthdate     string    `bun:"birthdate,notnull"`
	Elo           *int16    `bun:"elo,default:1000"`
	Created_at    time.Time `bun:"created_at,notnull,default:current_timestamp,skipupdate"`
	Updated_at    time.Time `bun:"updated_at,notnull,default:current_timestamp"`
}
