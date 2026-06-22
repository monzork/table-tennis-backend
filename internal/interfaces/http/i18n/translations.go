package i18n

// Translations holds all UI strings keyed by locale then message key.
var Translations = map[string]map[string]string{
	"en": {
		// Nav
		"nav.singles":     "Singles",
		"nav.doubles":     "Doubles",
		"nav.tournaments": "Tournaments",
		"nav.register":    "Register",

		// Rankings page
		"rankings.title":            "Rankings",
		"rankings.subtitle":         "Official world rankings by category.",
		"rankings.mens_singles":     "♂ Men's Singles",
		"rankings.womens_singles":   "♀ Women's Singles",
		"rankings.mens_doubles":     "♂ Men's Doubles",
		"rankings.womens_doubles":   "♀ Women's Doubles",
		"rankings.mixed_doubles":    "⚤ Mixed Doubles",
		"rankings.search_label":     "Search Player",
		"rankings.search_ph":        "Name, Country, or Dept...",
		"rankings.filter_label":     "Filter Division",
		"rankings.all_divisions":    "All Divisions",
		"rankings.sort_label":       "Sort Order",
		"rankings.sort_pts_desc":    "Points (High to Low)",
		"rankings.sort_pts_asc":     "Points (Low to High)",
		"rankings.sort_name_asc":    "Name (A-Z)",
		"rankings.col_rank":         "Rank",
		"rankings.col_player":       "Player",
		"rankings.col_assoc":        "Assoc.",
		"rankings.col_points":       "Points",

		// Register page (player)
		"register.title":          "Join the Club.",
		"register.subtitle":       "Official Athlete Registration",
		"register.first_name":     "First Name",
		"register.last_name":      "Last Name",
		"register.birthdate":      "Birthdate",
		"register.gender":         "Gender",
		"register.gender_male":    "Male",
		"register.gender_female":  "Female",
		"register.country":        "Country / ISO Code",
		"register.country_ph":     "e.g. MEX or USA",
		"register.whatsapp":       "WhatsApp Number (Optional)",
		"register.whatsapp_ph":    "+52 1 123 456 7890",
		"register.whatsapp_hint":  "Used for tournament notifications and registration.",
		"register.btn":            "Verify & Register",
		"register.back_rankings":  "← Back to Rankings",

		// Register success
		"register_success.view_rankings": "View Rankings",

		// Tournament registration page
		"tourney_reg.title":           "Tournament Registration",
		"tourney_reg.subtitle":        "Register for an Open Tournament",
		"tourney_reg.open_label":      "Open for Registration",
		"tourney_reg.starts":          "Starts",
		"tourney_reg.ends":            "Ends",
		"tourney_reg.type":            "Type",
		"tourney_reg.category":        "Category",
		"tourney_reg.participants":    "Participants",
		"tourney_reg.register_btn":    "Register for this Tournament",
		"tourney_reg.no_open":         "There are currently no tournaments open for registration.",
		"tourney_reg.back":            "← Back to Rankings",
		"tourney_reg.already_in":      "You are already registered in this tournament.",
		"tourney_reg.closed":          "This tournament is not open for registration.",
		"tourney_reg.not_found":       "Tournament not found.",
		"tourney_reg.form_title":      "Your Details",
		"tourney_reg.form_subtitle":   "We'll use your name to find your player profile.",
		"tourney_reg.name_label":      "Full Name",
		"tourney_reg.name_ph":         "John Doe",
		"tourney_reg.country_label":   "Country (ISO)",
		"tourney_reg.country_ph":      "MEX",
		"tourney_reg.submit_btn":      "Confirm Registration",
		"tourney_reg.success_title":   "You're In!",
		"tourney_reg.success_msg":     "You have been successfully registered for",
		"tourney_reg.view_bracket":    "View Rankings",

		// Footer
		"footer.copy": "© 2026 Club Rankings System",
	},
	"es": {
		// Nav
		"nav.singles":     "Individuales",
		"nav.doubles":     "Dobles",
		"nav.tournaments": "Torneos",
		"nav.register":    "Registro",

		// Rankings page
		"rankings.title":            "Clasificación",
		"rankings.subtitle":         "Ranking mundial oficial por categoría.",
		"rankings.mens_singles":     "♂ Individuales Varonil",
		"rankings.womens_singles":   "♀ Individuales Femenil",
		"rankings.mens_doubles":     "♂ Dobles Varonil",
		"rankings.womens_doubles":   "♀ Dobles Femenil",
		"rankings.mixed_doubles":    "⚤ Dobles Mixtos",
		"rankings.search_label":     "Buscar Jugador",
		"rankings.search_ph":        "Nombre, País, o Depto...",
		"rankings.filter_label":     "Filtrar División",
		"rankings.all_divisions":    "Todas las Divisiones",
		"rankings.sort_label":       "Orden",
		"rankings.sort_pts_desc":    "Puntos (Mayor a Menor)",
		"rankings.sort_pts_asc":     "Puntos (Menor a Mayor)",
		"rankings.sort_name_asc":    "Nombre (A-Z)",
		"rankings.col_rank":         "Pos.",
		"rankings.col_player":       "Jugador",
		"rankings.col_assoc":        "País",
		"rankings.col_points":       "Puntos",

		// Register page (player)
		"register.title":          "Únete al Club.",
		"register.subtitle":       "Registro Oficial de Atletas",
		"register.first_name":     "Nombre",
		"register.last_name":      "Apellido",
		"register.birthdate":      "Fecha de Nacimiento",
		"register.gender":         "Género",
		"register.gender_male":    "Masculino",
		"register.gender_female":  "Femenino",
		"register.country":        "País / Código ISO",
		"register.country_ph":     "ej. MEX o USA",
		"register.whatsapp":       "Número de WhatsApp (Opcional)",
		"register.whatsapp_ph":    "+52 1 123 456 7890",
		"register.whatsapp_hint":  "Utilizado para notificaciones y registro en torneos.",
		"register.btn":            "Verificar y Registrarse",
		"register.back_rankings":  "← Volver al Ranking",

		// Register success
		"register_success.view_rankings": "Ver Ranking",

		// Tournament registration page
		"tourney_reg.title":           "Registro de Torneo",
		"tourney_reg.subtitle":        "Regístrate en un Torneo Abierto",
		"tourney_reg.open_label":      "Abierto para Registro",
		"tourney_reg.starts":          "Inicio",
		"tourney_reg.ends":            "Fin",
		"tourney_reg.type":            "Tipo",
		"tourney_reg.category":        "Categoría",
		"tourney_reg.participants":    "Participantes",
		"tourney_reg.register_btn":    "Registrarme en este Torneo",
		"tourney_reg.no_open":         "Actualmente no hay torneos abiertos para registro.",
		"tourney_reg.back":            "← Volver al Ranking",
		"tourney_reg.already_in":      "Ya estás registrado en este torneo.",
		"tourney_reg.closed":          "Este torneo no está abierto para registro.",
		"tourney_reg.not_found":       "Torneo no encontrado.",
		"tourney_reg.form_title":      "Tus Datos",
		"tourney_reg.form_subtitle":   "Usaremos tu nombre para encontrar tu perfil de jugador.",
		"tourney_reg.name_label":      "Nombre Completo",
		"tourney_reg.name_ph":         "Juan Pérez",
		"tourney_reg.country_label":   "País (ISO)",
		"tourney_reg.country_ph":      "MEX",
		"tourney_reg.submit_btn":      "Confirmar Registro",
		"tourney_reg.success_title":   "¡Estás Inscrito!",
		"tourney_reg.success_msg":     "Fuiste registrado exitosamente en",
		"tourney_reg.view_bracket":    "Ver Ranking",

		// Footer
		"footer.copy": "© 2026 Sistema de Rankings del Club",
	},
}

// T returns the translated string for the given locale and key.
// Falls back to English, then the key itself.
func T(locale, key string) string {
	if locale == "" {
		locale = "en"
	}
	if msgs, ok := Translations[locale]; ok {
		if val, ok := msgs[key]; ok {
			return val
		}
	}
	// Fallback to English
	if msgs, ok := Translations["en"]; ok {
		if val, ok := msgs[key]; ok {
			return val
		}
	}
	return key
}
