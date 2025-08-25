package migrations

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/monzork/table-tennis-backend/internal/db"
	"github.com/monzork/table-tennis-backend/internal/domain/user"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	DB := db.Connect()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("1234"), 4)

	if err != nil {
		fmt.Printf("%v", err)
	}

	u := &user.User{
		ID:       uuid.New(),
		Username: "admin",
		Password: string(hashedPassword),
	}

	value := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "users" (
				"id"			TEXT,
				"username"		TEXT NOT NULL,
				"password"		TEXT NOT NULL,
				"created_at"	TEXT DEFAULT CURRENT_TIMESTAMP,
				"updated_at"	TEXT DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY("id") 
	);

				INSERT INTO users (id, username, password)
				VALUES("%s","%s","%s");`, u.ID, u.Username, u.Password)

	db.Migrations.MustRegister(
		func(ctx context.Context, db *bun.DB) error {
			_, err := DB.Exec(value)
			return err
		},
		func(ctx context.Context, db *bun.DB) error {
			_, err := DB.Exec(`DROP TABLE "users"`)
			return err
		},
	)
}
