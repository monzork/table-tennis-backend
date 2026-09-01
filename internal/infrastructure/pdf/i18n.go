package pdf

// esTranslations maps each report string (written in English in the
// generator code, matching the rest of the app's canonical i18n key
// convention) to its Spanish rendering -- the report's original,
// long-standing default. Strings with no entry here (e.g. numbers, already
// bilingual abbreviations like "Sets") render unchanged in both languages.
var esTranslations = map[string]string{
	"TABLE TENNIS TOURNAMENT - ": "TORNEO TENIS DE MESA - ",
	"TABLE TENNIS EVENT - ":      "EVENTO TENIS DE MESA - ",

	"FINAL PLACEMENTS":                     "POSICIONES FINALES",
	"REGISTERED PLAYERS - %s (%d PLAYERS)": "LISTA DE INSCRITOS - %s (%d JUGADORES)",
	"GROUP STANDINGS TABLES":               "TABLAS DE POSICIONES DE GRUPOS",
	"TOURNAMENT STATISTICS":                "ESTADÍSTICAS DEL TORNEO",
	"PLAYER STATISTICS":                    "ESTADÍSTICAS DE JUGADORES",

	"  1st Place (Champion): ": "  1er Lugar (Campeón):",
	"  2nd Place: ":            "  2do Lugar:",
	"  3rd Place: ":            "  3er Lugar:",

	"NAME":  "NOMBRE",
	"Day":   "Día",
	"Time":  "Hora",
	"Table": "Mesa",
	"Match": "Part.",

	"Points": "Puntos",

	"Total Matches: ":     "Total Partidos: ",
	"Total Sets: ":        "Total Sets: ",
	"Total Points: ":      "Total Puntos: ",
	"Avg. Points/Match: ": "Prom. Puntos/Partido: ",
	"Avg. Sets/Match: ":   "Prom. Sets/Partido: ",
	"Clean Sweeps: ":      "Barridas: ",
	"Deciding Sets: ":     "Sets Decisivos: ",
	"Avg. Starting Elo: ": "Prom. Elo Inicial: ",
	"Metrics by Division": "Métricas por División",
	"Division":            "División",
	"Matches Played":      "Partidos Jugados",
	"Avg. Points":         "Prom. Puntos",

	"Player": "Jugador",
	"Pld.":   "Jug.",

	"Open Division":   "División Abierta",
	"All Against All": "Todos contra Todos",
	"Unclassified":    "Sin Clasificar",
	"Open Bracket":    "Bracket Abierto",
	"Open":            "Abierto",

	" - Table %d": " - Mesa %d",
	"W/O":         "NSP",

	"REPORT SUMMARY":     "RESUMEN DEL REPORTE",
	"Date Generated: ":   "Fecha de Generación: ",
	"Total Sub-Events: ": "Total de Sub-Eventos: ",
	"%d Events":          "%d Eventos",

	"ID CARD - ": "CÉDULA DE IDENTIDAD - ",
	"ID No: ":    "Cédula No: ",
	"Front":      "Frente",
	"Back":       "Reverso",
}

// L renders a report string in the requested lang ("es" for Spanish; any
// other value, including "en" or empty, renders the English source text
// unchanged). en is always the literal fallback: an untranslated string
// still reads correctly in English rather than showing a raw lookup key.
func L(lang, en string) string {
	if lang == "es" {
		if v, ok := esTranslations[en]; ok {
			return v
		}
	}
	return en
}
