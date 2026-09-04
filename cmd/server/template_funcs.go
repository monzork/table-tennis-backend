package main

import (
	"fmt"
	"os"
	"strings"

	"html/template"
	"table-tennis-backend/internal/interfaces/http/handler"
	"table-tennis-backend/internal/interfaces/http/i18n"

	"github.com/gofiber/template/html/v2"
)

type CountryInfo struct {
	Code string
	Name string
}

var countriesList = []CountryInfo{
	{"NIC", "Nicaragua"},
	{"ARG", "Argentina"},
	{"BRA", "Brazil"},
	{"CAN", "Canada"},
	{"CHL", "Chile"},
	{"CHN", "China"},
	{"COL", "Colombia"},
	{"CRC", "Costa Rica"},
	{"CUB", "Cuba"},
	{"DOM", "Dominican Republic"},
	{"ECU", "Ecuador"},
	{"SLV", "El Salvador"},
	{"ESP", "Spain"},
	{"FRA", "France"},
	{"GER", "Germany"},
	{"GTM", "Guatemala"},
	{"HON", "Honduras"},
	{"JPN", "Japan"},
	{"KOR", "South Korea"},
	{"MEX", "Mexico"},
	{"PAN", "Panama"},
	{"PER", "Peru"},
	{"PRI", "Puerto Rico"},
	{"SWE", "Sweden"},
	{"TPE", "Chinese Taipei"},
	{"USA", "United States"},
	{"VEN", "Venezuela"},
}

func SetupTemplateEngine() *html.Engine {
	engine := html.New("./internal/interfaces/http/templates", ".html")
	engine.Reload(os.Getenv("DATABASE_URL") == "")
	engine.AddFunc("countries", func() []CountryInfo {
		return countriesList
	})
	engine.AddFunc("add", func(a, b int) int {
		return a + b
	})
	// seq(n) returns [1..n], for {{range seq .BestOf}} to render a
	// tournament-configured number of set-score inputs.
	engine.AddFunc("seq", func(n int) []int {
		if n <= 0 {
			return nil
		}
		s := make([]int, n)
		for i := range s {
			s[i] = i + 1
		}
		return s
	})
	engine.AddFunc("mul", func(a, b int) int {
		return a * b
	})
	engine.AddFunc("div", func(a, b int) int {
		if b == 0 {
			return 0
		}
		return a / b
	})
	engine.AddFunc("dict", func(values ...interface{}) (map[string]interface{}, error) {
		if len(values)%2 != 0 {
			return nil, fmt.Errorf("invalid dict call, must have even number of arguments")
		}
		dict := make(map[string]interface{}, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict keys must be strings")
			}
			dict[key] = values[i+1]
		}
		return dict, nil
	})
	engine.AddFunc("isNicaragua", func(country string) bool {
		c := strings.TrimSpace(strings.ToUpper(country))
		return c == "NIC" || c == "NICARAGUA" || c == "NI"
	})
	engine.AddFunc("nicaraguaDepartments", func() []string {
		return handler.NicaraguaDepartments
	})
	// t(tmap, key) — shorthand for {{index .T "key"}} usable as {{t .T "key"}}
	engine.AddFunc("t", func(tmap map[string]string, key string) string {
		if tmap != nil {
			if v, ok := tmap[key]; ok {
				return v
			}
		}
		if v, ok := i18n.Translations["en"][key]; ok {
			return v
		}
		return key
	})
	engine.AddFunc("cleanPhone", func(phone string) string {
		var b strings.Builder
		for _, ch := range phone {
			if ch >= '0' && ch <= '9' {
				b.WriteRune(ch)
			}
		}
		return b.String()
	})
	engine.AddFunc("safeHTML", func(s string) template.HTML {
		return template.HTML(s)
	})
	engine.AddFunc("eventDivision", handler.EventDivisionName)
	// eloDelta formats a per-match Elo change as a signed whole-number
	// string (e.g. "+18", "-12"), or "" if Elo hasn't been applied to this
	// match yet.
	engine.AddFunc("eloDelta", func(delta *float64) string {
		if delta == nil {
			return ""
		}
		return fmt.Sprintf("%+.0f", *delta)
	})
	engine.AddFunc("rankMovement", handler.RenderRankMovement)
	// inactivityLoss picks the rank-type-appropriate side of a player's
	// cumulative inactivity-decay loss (see player.Player.LostToInactivitySingles/Doubles).
	engine.AddFunc("inactivityLoss", func(rankType string, singles, doubles int16) int16 {
		if rankType == "doubles" {
			return doubles
		}
		return singles
	})
	return engine
}
