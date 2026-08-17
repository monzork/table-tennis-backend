package handler

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	accountApp "table-tennis-backend/internal/application/account"
	matchApp "table-tennis-backend/internal/application/match"
	playerApp "table-tennis-backend/internal/application/player"
	"table-tennis-backend/internal/infrastructure/oauth"
	"table-tennis-backend/internal/interfaces/http/i18n"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

// AccountHandler serves the guardian-account area (/account/*), entirely
// separate from /admin: its own session keys (account_authenticated,
// account_id), its own middleware (AccountProtected), no shared template
// chrome with the admin panel.
type AccountHandler struct {
	store        *session.Store
	googleClient *oauth.GoogleClient

	loginUC                     *accountApp.LoginWithGoogleUseCase
	getAccountByIDUC            *accountApp.GetAccountByIDUseCase
	updateAccountUC             *accountApp.UpdateAccountUseCase
	createChildUC               *accountApp.CreateChildPlayerUseCase
	getLinkedPlayersUC          *accountApp.GetLinkedPlayersUseCase
	getGuardianPendingMatchesUC *accountApp.GetGuardianPendingMatchesUseCase

	getPlayerByIDUC         *playerApp.GetPlayerByIDUseCase
	updatePlayerUC          *playerApp.UpdatePlayerUseCase
	getPlayerPendingMatches *playerApp.GetPlayerPendingMatchesUseCase

	proposeScoreUC *matchApp.ProposeMatchScoreUseCase
	confirmScoreUC *matchApp.ConfirmMatchScoreUseCase
	rejectScoreUC  *matchApp.RejectMatchScoreProposalUseCase
}

func NewAccountHandler(
	store *session.Store,
	googleClient *oauth.GoogleClient,
	loginUC *accountApp.LoginWithGoogleUseCase,
	getAccountByIDUC *accountApp.GetAccountByIDUseCase,
	updateAccountUC *accountApp.UpdateAccountUseCase,
	createChildUC *accountApp.CreateChildPlayerUseCase,
	getLinkedPlayersUC *accountApp.GetLinkedPlayersUseCase,
	getGuardianPendingMatchesUC *accountApp.GetGuardianPendingMatchesUseCase,
	getPlayerByIDUC *playerApp.GetPlayerByIDUseCase,
	updatePlayerUC *playerApp.UpdatePlayerUseCase,
	getPlayerPendingMatches *playerApp.GetPlayerPendingMatchesUseCase,
	proposeScoreUC *matchApp.ProposeMatchScoreUseCase,
	confirmScoreUC *matchApp.ConfirmMatchScoreUseCase,
	rejectScoreUC *matchApp.RejectMatchScoreProposalUseCase,
) *AccountHandler {
	return &AccountHandler{
		store:                       store,
		googleClient:                googleClient,
		loginUC:                     loginUC,
		getAccountByIDUC:            getAccountByIDUC,
		updateAccountUC:             updateAccountUC,
		createChildUC:               createChildUC,
		getLinkedPlayersUC:          getLinkedPlayersUC,
		getGuardianPendingMatchesUC: getGuardianPendingMatchesUC,
		getPlayerByIDUC:             getPlayerByIDUC,
		updatePlayerUC:              updatePlayerUC,
		getPlayerPendingMatches:     getPlayerPendingMatches,
		proposeScoreUC:              proposeScoreUC,
		confirmScoreUC:              confirmScoreUC,
		rejectScoreUC:               rejectScoreUC,
	}
}

// ── Auth ─────────────────────────────────────────────────────────────────

func (h *AccountHandler) ShowLogin(c *fiber.Ctx) error {
	sess, err := h.store.Get(c)
	if err == nil {
		if auth := sess.Get("account_authenticated"); auth != nil && auth.(bool) {
			return c.Redirect("/account")
		}
	}
	lang := getLang(c)
	return c.Render("account/login", merge(tMap(lang), fiber.Map{
		"Title":     i18n.T(lang, "account.login.title"),
		"CSRFToken": c.Locals("CSRFToken"),
	}), "layouts/public")
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GoogleLogin builds a CSRF state, stashes it in the session, and redirects
// to Google's consent screen.
func (h *AccountHandler) GoogleLogin(c *fiber.Ctx) error {
	sess, err := h.store.Get(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "session error")
	}

	state, err := randomState()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate state")
	}
	sess.Set("oauth_state", state)
	if err := sess.Save(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to save session")
	}

	return c.Redirect(h.googleClient.AuthCodeURL(state))
}

// GoogleCallback verifies state, exchanges the code, and establishes the
// account session — same HTMX-aware-redirect pattern as AuthHandler.Login.
func (h *AccountHandler) GoogleCallback(c *fiber.Ctx) error {
	lang := getLang(c)
	sess, err := h.store.Get(c)
	if err != nil {
		return c.Render("account/login", merge(tMap(lang), fiber.Map{"Error": i18n.T(lang, "account.login.err_session")}), "layouts/public")
	}

	expectedState, _ := sess.Get("oauth_state").(string)
	state := c.Query("state")
	if expectedState == "" || state != expectedState {
		return c.Render("account/login", merge(tMap(lang), fiber.Map{"Error": i18n.T(lang, "account.login.err_state")}), "layouts/public")
	}
	sess.Delete("oauth_state")

	code := c.Query("code")
	if code == "" {
		return c.Render("account/login", merge(tMap(lang), fiber.Map{"Error": i18n.T(lang, "account.login.err_oauth")}), "layouts/public")
	}

	info, err := h.googleClient.Exchange(c.Context(), code)
	if err != nil {
		return c.Render("account/login", merge(tMap(lang), fiber.Map{"Error": i18n.T(lang, "account.login.err_oauth")}), "layouts/public")
	}

	acc, err := h.loginUC.Execute(c.Context(), accountApp.LoginWithGoogleCommand{
		GoogleSub:  info.Sub,
		Email:      info.Email,
		Name:       info.Name,
		PictureURL: info.Picture,
	})
	if err != nil {
		return c.Render("account/login", merge(tMap(lang), fiber.Map{"Error": i18n.T(lang, "account.login.err_oauth")}), "layouts/public")
	}

	sess.Set("account_authenticated", true)
	sess.Set("account_id", acc.ID)
	if err := sess.Save(); err != nil {
		return c.Render("account/login", merge(tMap(lang), fiber.Map{"Error": i18n.T(lang, "account.login.err_session")}), "layouts/public")
	}

	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", "/account")
		return c.SendStatus(200)
	}
	return c.Redirect("/account")
}

func (h *AccountHandler) Logout(c *fiber.Ctx) error {
	sess, err := h.store.Get(c)
	if err == nil {
		sess.Destroy()
	}
	return c.Redirect("/account/login")
}

// ── Dashboard ────────────────────────────────────────────────────────────

func (h *AccountHandler) accountID(c *fiber.Ctx) string {
	id, _ := c.Locals("AccountID").(string)
	return id
}

func (h *AccountHandler) Dashboard(c *fiber.Ctx) error {
	lang := getLang(c)
	accountID := h.accountID(c)

	players, err := h.getLinkedPlayersUC.Execute(c.Context(), accountID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	pending, err := h.getGuardianPendingMatchesUC.Execute(c.Context(), accountID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Render("account/dashboard", merge(tMap(lang), fiber.Map{
		"Title":   i18n.T(lang, "account.dashboard.title"),
		"Players": players,
		"Pending": pending,
	}), "layouts/public")
}

func (h *AccountHandler) PendingMatches(c *fiber.Ctx) error {
	lang := getLang(c)
	pending, err := h.getGuardianPendingMatchesUC.Execute(c.Context(), h.accountID(c))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Render("account/pending-matches", merge(tMap(lang), fiber.Map{
		"Title":   i18n.T(lang, "account.pending.title"),
		"Pending": pending,
	}), "layouts/public")
}

// ── My info ──────────────────────────────────────────────────────────────

func (h *AccountHandler) ShowMyInfo(c *fiber.Ctx) error {
	lang := getLang(c)
	acc, err := h.getAccountByIDUC.Execute(c.Context(), h.accountID(c))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "account not found")
	}
	return c.Render("account/my-info", merge(tMap(lang), fiber.Map{
		"Title":   i18n.T(lang, "account.my_info.title"),
		"Account": acc,
	}), "layouts/public")
}

func (h *AccountHandler) UpdateMyInfo(c *fiber.Ctx) error {
	var body struct {
		Name string `json:"name" form:"name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	acc, err := h.updateAccountUC.Execute(c.Context(), h.accountID(c), body.Name)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	lang := getLang(c)
	return c.Render("account/my-info", merge(tMap(lang), fiber.Map{
		"Title":   i18n.T(lang, "account.my_info.title"),
		"Account": acc,
		"Saved":   true,
	}), "layouts/public")
}

// ── Child players ────────────────────────────────────────────────────────

func (h *AccountHandler) ShowAddChildForm(c *fiber.Ctx) error {
	lang := getLang(c)
	return c.Render("account/player-form", merge(tMap(lang), fiber.Map{
		"Title": i18n.T(lang, "account.player_form.add_title"),
	}), "layouts/public")
}

func (h *AccountHandler) CreateChild(c *fiber.Ctx) error {
	var body struct {
		FirstName  string `json:"firstName" form:"firstName"`
		LastName   string `json:"lastName" form:"lastName"`
		Birthdate  string `json:"birthdate" form:"birthdate"`
		Gender     string `json:"gender" form:"gender"`
		Country    string `json:"country" form:"country"`
		Department string `json:"department" form:"department"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	bd, _ := time.Parse("2006-01-02", body.Birthdate)

	_, err := h.createChildUC.Execute(c.Context(), accountApp.CreateChildPlayerCommand{
		GuardianAccountID: h.accountID(c),
		FirstName:         body.FirstName,
		LastName:          body.LastName,
		Birthdate:         bd,
		Gender:            body.Gender,
		Country:           body.Country,
		Department:        body.Department,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", "/account")
		return c.SendStatus(200)
	}
	return c.Redirect("/account")
}

func (h *AccountHandler) PlayerDetail(c *fiber.Ctx) error {
	lang := getLang(c)
	playerID := c.Params("id")
	p, err := h.getPlayerByIDUC.Execute(c.Context(), playerID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "player not found")
	}
	if err := accountApp.EnsurePlayerBelongsToAccount(p, h.accountID(c)); err != nil {
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	}

	pending, err := h.getPlayerPendingMatches.Execute(c.Context(), playerID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Render("account/player-detail", merge(tMap(lang), fiber.Map{
		"Title":   p.FullName(),
		"Player":  p,
		"Pending": pending,
	}), "layouts/public")
}

func (h *AccountHandler) EditPlayer(c *fiber.Ctx) error {
	lang := getLang(c)
	playerID := c.Params("id")
	p, err := h.getPlayerByIDUC.Execute(c.Context(), playerID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "player not found")
	}
	if err := accountApp.EnsurePlayerBelongsToAccount(p, h.accountID(c)); err != nil {
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	}

	return c.Render("account/player-form", merge(tMap(lang), fiber.Map{
		"Title":  i18n.T(lang, "account.player_form.edit_title"),
		"Player": p,
	}), "layouts/public")
}

func (h *AccountHandler) UpdatePlayer(c *fiber.Ctx) error {
	playerID := c.Params("id")
	existing, err := h.getPlayerByIDUC.Execute(c.Context(), playerID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "player not found")
	}
	if err := accountApp.EnsurePlayerBelongsToAccount(existing, h.accountID(c)); err != nil {
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	}

	var body struct {
		FirstName  string `json:"firstName" form:"firstName"`
		LastName   string `json:"lastName" form:"lastName"`
		Birthdate  string `json:"birthdate" form:"birthdate"`
		Gender     string `json:"gender" form:"gender"`
		Country    string `json:"country" form:"country"`
		Department string `json:"department" form:"department"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	p, err := h.updatePlayerUC.Execute(c.Context(), playerID, body.FirstName, "", body.LastName, "",
		body.Birthdate, body.Gender, body.Country, body.Department, "", "",
		existing.SinglesElo, existing.DoublesElo)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", "/account/players/"+p.ID)
		return c.SendStatus(200)
	}
	return c.Redirect("/account/players/" + p.ID)
}

// ── Score proposals ──────────────────────────────────────────────────────

// ProposeScore handles both a real match (:matchId is its ID) and a
// "potential" matchup the account UI shows before any Match row exists for
// it (:matchId is the literal "new" sentinel) — see event.BuildBoardCards
// and ProposeMatchScoreUseCase's find-or-create handling.
func (h *AccountHandler) ProposeScore(c *fiber.Ctx) error {
	matchID := c.Params("matchId")
	if matchID == "new" {
		matchID = ""
	}
	var body struct {
		PlayerID   string   `json:"playerId" form:"playerId"`
		EventID    string   `json:"eventId" form:"eventId"`
		OpponentID string   `json:"opponentId" form:"opponentId"`
		Stage      string   `json:"stage" form:"stage"`
		Sets       []string `json:"sets" form:"sets[]"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	// fiber's BodyParser doesn't reliably bind repeated form keys into a
	// slice field (same limitation worked around in match_handler.go) —
	// fall back to reading them directly off the raw form.
	if len(body.Sets) == 0 {
		for _, s := range c.Request().PostArgs().PeekMulti("sets[]") {
			body.Sets = append(body.Sets, string(s))
		}
	}

	sets, err := matchApp.ParseSetScores(body.Sets)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	err = h.proposeScoreUC.Execute(c.Context(), matchApp.ProposeMatchScoreCommand{
		AccountID:          h.accountID(c),
		MatchID:            matchID,
		ProposedByPlayerID: body.PlayerID,
		EventID:            body.EventID,
		OpponentID:         body.OpponentID,
		Stage:              body.Stage,
		Sets:               sets,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", "/account/pending-matches")
		return c.SendStatus(200)
	}
	return c.Redirect("/account/pending-matches")
}

func (h *AccountHandler) ConfirmScore(c *fiber.Ctx) error {
	matchID := c.Params("matchId")
	var body struct {
		PlayerID string `json:"playerId" form:"playerId"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	// Ownership check: the confirming player must belong to this account.
	p, err := h.getPlayerByIDUC.Execute(c.Context(), body.PlayerID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "player not found")
	}
	if err := accountApp.EnsurePlayerBelongsToAccount(p, h.accountID(c)); err != nil {
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	}

	playerID := body.PlayerID
	if err := h.confirmScoreUC.Execute(c.Context(), matchID, &playerID, false); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", "/account/pending-matches")
		return c.SendStatus(200)
	}
	return c.Redirect("/account/pending-matches")
}

func (h *AccountHandler) RejectScore(c *fiber.Ctx) error {
	matchID := c.Params("matchId")
	var body struct {
		PlayerID string `json:"playerId" form:"playerId"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	p, err := h.getPlayerByIDUC.Execute(c.Context(), body.PlayerID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "player not found")
	}
	if err := accountApp.EnsurePlayerBelongsToAccount(p, h.accountID(c)); err != nil {
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	}

	if err := h.rejectScoreUC.Execute(c.Context(), matchID, body.PlayerID); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", "/account/pending-matches")
		return c.SendStatus(200)
	}
	return c.Redirect("/account/pending-matches")
}
