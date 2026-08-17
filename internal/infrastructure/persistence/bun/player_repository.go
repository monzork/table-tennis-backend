package bun

import (
	"context"
	"fmt"
	"strings"
	"table-tennis-backend/internal/domain/player"
	"unicode"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// accentPairs maps accented characters (both cases) to their unaccented,
// lowercase equivalent. Used only as a SQLite fallback (local dev without
// DATABASE_URL) — PostgreSQL uses its native unaccent() extension instead
// (migration 048), which handles this in the DB rather than in Go.
var accentPairs = []struct{ from, to string }{
	{"á", "a"}, {"Á", "a"}, {"à", "a"}, {"À", "a"}, {"ä", "a"}, {"Ä", "a"}, {"â", "a"}, {"Â", "a"}, {"ã", "a"}, {"Ã", "a"},
	{"é", "e"}, {"É", "e"}, {"è", "e"}, {"È", "e"}, {"ë", "e"}, {"Ë", "e"}, {"ê", "e"}, {"Ê", "e"},
	{"í", "i"}, {"Í", "i"}, {"ì", "i"}, {"Ì", "i"}, {"ï", "i"}, {"Ï", "i"}, {"î", "i"}, {"Î", "i"},
	{"ó", "o"}, {"Ó", "o"}, {"ò", "o"}, {"Ò", "o"}, {"ö", "o"}, {"Ö", "o"}, {"ô", "o"}, {"Ô", "o"}, {"õ", "o"}, {"Õ", "o"},
	{"ú", "u"}, {"Ú", "u"}, {"ù", "u"}, {"Ù", "u"}, {"ü", "u"}, {"Ü", "u"}, {"û", "u"}, {"Û", "u"},
	{"ñ", "n"}, {"Ñ", "n"}, {"ç", "c"}, {"Ç", "c"},
}

// unaccentExprSQLite builds a SQL expression that folds case and strips the
// diacritics in accentPairs from col via nested REPLACE()+LOWER(), for
// dialects (SQLite) without a native unaccent() function.
func unaccentExprSQLite(col string) string {
	expr := col
	for _, p := range accentPairs {
		expr = fmt.Sprintf("REPLACE(%s, '%s', '%s')", expr, p.from, p.to)
	}
	return "LOWER(" + expr + ")"
}

// stripAccents removes diacritics from s (e.g. "José" -> "Jose") so search
// terms can be compared against unaccentExprSQLite'd columns.
func stripAccents(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, err := transform.String(t, s)
	if err != nil {
		return s
	}
	return result
}

// applyNameSearch AND-filters q so every whitespace-separated term in query
// must match (accent- and case-insensitively) at least one of the player's
// name columns. On PostgreSQL this is delegated entirely to the DB via the
// unaccent() extension; SQLite (local dev only) falls back to Go-side
// diacritic stripping since it has no such builtin.
func (r *PlayerRepository) applyNameSearch(q *bun.SelectQuery, query string) *bun.SelectQuery {
	if query == "" {
		return q
	}
	cols := []string{"first_name", "second_name", "last_name", "second_last_name"}
	isPG := r.db.Dialect().Name() == dialect.PG

	parts := make([]string, len(cols))
	for i, c := range cols {
		if isPG {
			parts[i] = fmt.Sprintf("unaccent(LOWER(%s)) LIKE unaccent(?)", c)
		} else {
			parts[i] = fmt.Sprintf("%s LIKE ?", unaccentExprSQLite(c))
		}
	}
	condition := "(" + strings.Join(parts, " OR ") + ")"

	terms := strings.Fields(strings.ToLower(query))
	for _, term := range terms {
		pattern := term
		if !isPG {
			pattern = stripAccents(term)
		}
		pattern = "%" + pattern + "%"
		q = q.Where(condition, pattern, pattern, pattern, pattern)
	}
	return q
}

type PlayerRepository struct {
	db *bun.DB
}

func NewPlayerRepository(db *bun.DB) *PlayerRepository {
	return &PlayerRepository{db: db}
}

func (r *PlayerRepository) Save(ctx context.Context, p *player.Player) error {
	id, err := uuid.Parse(p.ID)
	if err != nil {
		return err
	}
	model := &PlayerModel{
		ID:             id,
		FirstName:      p.FirstName,
		SecondName:     p.SecondName,
		LastName:       p.LastName,
		SecondLastName: p.SecondLastName,
		Birthdate:      p.Birthdate,
		Gender:         p.Gender,
		SinglesElo:     p.SinglesElo,
		DoublesElo:     p.DoublesElo,
		Country:        p.Country,
		Department:     p.Department,
		WhatsAppNumber: p.WhatsAppNumber,
		NationalID:     p.NationalID,
	}

	_, err = ExtractDB(ctx, r.db).NewInsert().Model(model).
		On("CONFLICT (id) DO UPDATE").
		Set("first_name = EXCLUDED.first_name, second_name = EXCLUDED.second_name, last_name = EXCLUDED.last_name, second_last_name = EXCLUDED.second_last_name, gender = EXCLUDED.gender, singles_elo = EXCLUDED.singles_elo, doubles_elo = EXCLUDED.doubles_elo, country = EXCLUDED.country, whatsapp_number = EXCLUDED.whatsapp_number, department = EXCLUDED.department, national_id = EXCLUDED.national_id").
		Exec(ctx)

	return err
}

func (r *PlayerRepository) GetAllSingles(ctx context.Context) ([]*player.Player, error) {
	var models []PlayerModel
	err := ExtractDB(ctx, r.db).NewSelect().Model(&models).OrderBy("singles_elo", bun.OrderDesc).Scan(ctx)

	if err != nil {
		return nil, err
	}
	return r.mapModelsToDomain(models), nil
}

func (r *PlayerRepository) GetAllDoubles(ctx context.Context) ([]*player.Player, error) {
	var models []PlayerModel
	err := ExtractDB(ctx, r.db).NewSelect().Model(&models).OrderBy("doubles_elo", bun.OrderDesc).Scan(ctx)

	if err != nil {
		return nil, err
	}
	return r.mapModelsToDomain(models), nil
}

func (r *PlayerRepository) GetAll(ctx context.Context) ([]*player.Player, error) {
	return r.GetAllSingles(ctx)
}

func (r *PlayerRepository) GetSinglesByGender(ctx context.Context, gender string) ([]*player.Player, error) {
	var models []PlayerModel
	q := ExtractDB(ctx, r.db).NewSelect().Model(&models).OrderBy("singles_elo", bun.OrderDesc)
	if gender != "" {
		q = q.Where("gender = ?", gender)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, err
	}
	return r.mapModelsToDomain(models), nil
}

func (r *PlayerRepository) GetDoublesByGender(ctx context.Context, gender string) ([]*player.Player, error) {
	var models []PlayerModel
	q := ExtractDB(ctx, r.db).NewSelect().Model(&models).OrderBy("doubles_elo", bun.OrderDesc)
	if gender != "" {
		q = q.Where("gender = ?", gender)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, err
	}
	return r.mapModelsToDomain(models), nil
}

func (r *PlayerRepository) mapModelsToDomain(models []PlayerModel) []*player.Player {
	players := make([]*player.Player, len(models))
	for i, m := range models {
		players[i] = &player.Player{
			ID:             m.ID.String(),
			FirstName:      m.FirstName,
			SecondName:     m.SecondName,
			LastName:       m.LastName,
			SecondLastName: m.SecondLastName,
			Birthdate:      m.Birthdate,
			Gender:         m.Gender,
			SinglesElo:     m.SinglesElo,
			DoublesElo:     m.DoublesElo,
			Country:        m.Country,
			Department:     m.Department,
			WhatsAppNumber: m.WhatsAppNumber,
			NationalID:     m.NationalID,
		}
	}
	return players
}

func (r *PlayerRepository) GetById(ctx context.Context, id string) (*player.Player, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	var model PlayerModel
	err = ExtractDB(ctx, r.db).NewSelect().Model(&model).Where("id = ?", uid).Scan(ctx)

	if err != nil {
		return nil, err
	}

	return &player.Player{
		ID:             model.ID.String(),
		FirstName:      model.FirstName,
		SecondName:     model.SecondName,
		LastName:       model.LastName,
		SecondLastName: model.SecondLastName,
		Birthdate:      model.Birthdate,
		Gender:         model.Gender,
		SinglesElo:     model.SinglesElo,
		DoublesElo:     model.DoublesElo,
		Country:        model.Country,
		Department:     model.Department,
		WhatsAppNumber: model.WhatsAppNumber,
		NationalID:     model.NationalID,
	}, nil
}

func (r *PlayerRepository) GetByIDs(ctx context.Context, ids []string) ([]*player.Player, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	uids := make([]uuid.UUID, 0, len(ids))
	for _, idStr := range ids {
		if uid, err := uuid.Parse(idStr); err == nil {
			uids = append(uids, uid)
		}
	}
	if len(uids) == 0 {
		return nil, nil
	}
	var models []PlayerModel
	err := ExtractDB(ctx, r.db).NewSelect().Model(&models).Where("id IN (?)", bun.List(uids)).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return r.mapModelsToDomain(models), nil
}

func (r *PlayerRepository) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	_, err = ExtractDB(ctx, r.db).NewDelete().Model((*PlayerModel)(nil)).Where("id = ?", uid).Exec(ctx)
	return err
}

func (r *PlayerRepository) Search(ctx context.Context, query string) ([]*player.Player, error) {
	var models []PlayerModel
	q := r.applyNameSearch(ExtractDB(ctx, r.db).NewSelect().Model(&models).OrderBy("singles_elo", bun.OrderDesc), query)
	if err := q.Scan(ctx); err != nil {
		return nil, err
	}
	return r.mapModelsToDomain(models), nil
}

// SearchForSelection is a lighter-weight variant of Search for the participant
// selection cards UI, which only ever renders name, gender and singles Elo.
func (r *PlayerRepository) SearchForSelection(ctx context.Context, query, gender string) ([]*player.Player, error) {
	var models []PlayerModel
	q := r.applyNameSearch(ExtractDB(ctx, r.db).NewSelect().
		Model(&models).
		Column("id", "first_name", "second_name", "last_name", "second_last_name", "gender", "singles_elo").
		OrderBy("singles_elo", bun.OrderDesc), query)
	if gender != "" {
		q = q.Where("gender = ?", gender)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, err
	}
	return r.mapModelsToDomain(models), nil
}

func (r *PlayerRepository) SaveMultiple(ctx context.Context, players []*player.Player) error {
	if len(players) == 0 {
		return nil
	}
	models := make([]PlayerModel, len(players))
	for i, p := range players {
		id, err := uuid.Parse(p.ID)
		if err != nil {
			return err
		}
		models[i] = PlayerModel{
			ID:             id,
			FirstName:      p.FirstName,
			SecondName:     p.SecondName,
			LastName:       p.LastName,
			SecondLastName: p.SecondLastName,
			Birthdate:      p.Birthdate,
			Gender:         p.Gender,
			SinglesElo:     p.SinglesElo,
			DoublesElo:     p.DoublesElo,
			Country:        p.Country,
			Department:     p.Department,
			WhatsAppNumber: p.WhatsAppNumber,
			NationalID:     p.NationalID,
		}
	}

	_, err := ExtractDB(ctx, r.db).NewInsert().Model(&models).
		On("CONFLICT (id) DO UPDATE").
		Set("first_name = EXCLUDED.first_name, second_name = EXCLUDED.second_name, last_name = EXCLUDED.last_name, second_last_name = EXCLUDED.second_last_name, gender = EXCLUDED.gender, singles_elo = EXCLUDED.singles_elo, doubles_elo = EXCLUDED.doubles_elo, country = EXCLUDED.country, whatsapp_number = EXCLUDED.whatsapp_number, department = EXCLUDED.department, national_id = EXCLUDED.national_id").
		Exec(ctx)
	return err
}
