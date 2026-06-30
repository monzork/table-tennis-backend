package main

import (
	"fmt"
	"os"
	"strings"

	"table-tennis-backend/internal/interfaces/http/handler"
	"table-tennis-backend/internal/interfaces/http/i18n"
	"html/template"

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
	engine.AddFunc("safeHTML", func(s string) template.HTML {
		return template.HTML(s)
	})
	return engine
}
