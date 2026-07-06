package main

import (
	"context"

	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/event"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/application/match"
	"table-tennis-backend/internal/application/player"
	"table-tennis-backend/internal/application/tournament"
	adminDomain "table-tennis-backend/internal/domain/admin"
	"table-tennis-backend/internal/domain/events"
	"table-tennis-backend/internal/domain/idgen"
	pdfinfra "table-tennis-backend/internal/infrastructure/pdf"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
	qrinfra "table-tennis-backend/internal/infrastructure/qrcode"
	"table-tennis-backend/internal/infrastructure/security"
	"table-tennis-backend/internal/interfaces/http/handler"

	"github.com/gofiber/fiber/v2/middleware/session"
)

type Container struct {
	PlayerHandler      *handler.PlayerHandler
	TournamentHandler  *handler.TournamentHandler
	EventHandler       *handler.EventHandler
	MatchHandler       *handler.MatchHandler
	LeaderboardHandler *handler.LeaderboardHandler
	DivisionHandler    *handler.DivisionHandler
	PublicHandler      *handler.PublicHandler
	QRHandler          *handler.QRHandler
	AuthHandler        *handler.AuthHandler
	AdminHandler       *handler.AdminHandler
}

func NewContainer(store *session.Store, cfg Config) *Container {
	playerRepo := bun.NewPlayerRepository(bun.DB)
	playerUC := player.NewRegisterPlayerUseCase(playerRepo)
	updatePlayerUC := player.NewUpdatePlayerUseCase(playerRepo)
	deletePlayerUC := player.NewDeletePlayerUseCase(playerRepo)
	importPlayerUC := player.NewImportPlayersUseCase(playerRepo)
	getPlayerByIDUC := player.NewGetPlayerByIDUseCase(playerRepo)
	searchPlayerUC := player.NewSearchPlayersUseCase(playerRepo)
	searchPlayerSelectionUC := player.NewSearchPlayersForSelectionUseCase(playerRepo)
	tournamentRepo := bun.NewTournamentRepository(bun.DB)

	dispatcher := events.NewInMemoryDispatcher()
	enrollPlayerUC := tournament.NewEnrollPlayerUseCase(tournamentRepo, dispatcher)
	getTournamentsUC := tournament.NewGetTournamentsUseCase(tournamentRepo)


	leaderboardUC := leaderboard.NewGetLeaderboardUseCase(playerRepo)

	divisionRepo := bun.NewDivisionRepository(bun.DB)
	divisionUC := division.NewDivisionUseCase(divisionRepo)

	createTournamentUC := tournament.NewCreateTournamentUseCase(tournamentRepo, playerRepo, divisionRepo)
	getTournamentByIDUC := tournament.NewGetTournamentByIDUseCase(tournamentRepo, divisionRepo)
	updateTournamentUC := tournament.NewUpdateTournamentUseCase(tournamentRepo, playerRepo, divisionRepo)
	deleteTournamentUC := tournament.NewDeleteTournamentUseCase(tournamentRepo)
	matchRepo := bun.NewMatchRepository(bun.DB, playerRepo)
	finishTournamentUC := tournament.NewFinishTournamentUseCase(tournamentRepo, matchRepo, playerRepo)
	exportTournamentUC := tournament.NewExportTournamentReportUseCase(tournamentRepo)
	pdfGenerator := pdfinfra.NewGoFpdfGenerator()
	exportTournamentPdfUC := tournament.NewExportTournamentPdfUseCase(tournamentRepo, divisionRepo, pdfGenerator)
	movePlayerUC := tournament.NewMovePlayerUseCase(tournamentRepo)
	createTeamUC := tournament.NewCreateTeamUseCase(tournamentRepo)
	deleteTeamUC := tournament.NewDeleteTeamUseCase(tournamentRepo)
	assignPlayerToTeamUC := tournament.NewAssignPlayerToTeamUseCase(tournamentRepo)
	removePlayerFromTeamUC := tournament.NewRemovePlayerFromTeamUseCase(tournamentRepo)
	regenerateSeedsUC := tournament.NewRegenerateGroupSeedsUseCase(tournamentRepo, matchRepo, divisionRepo)
	dispatcher.Subscribe(events.PlayerEnrolledEventName, func(ctx context.Context, e events.Event) error {
		if pe, ok := e.(events.PlayerEnrolledEvent); ok {
			_ = regenerateSeedsUC.Execute(ctx, pe.TournamentID)
		}
		return nil
	})
	updateParticipantEloUC := tournament.NewUpdateParticipantEloBeforeUseCase(tournamentRepo, regenerateSeedsUC)
	playerHandler := handler.NewPlayerHandler(playerUC, updatePlayerUC, deletePlayerUC, getPlayerByIDUC, searchPlayerUC, searchPlayerSelectionUC, importPlayerUC, enrollPlayerUC, getTournamentsUC)

	tournamentHandler := handler.NewTournamentHandler(
		createTournamentUC,
		getTournamentByIDUC,
		updateTournamentUC,
		deleteTournamentUC,
		leaderboardUC,
		divisionUC,
		finishTournamentUC,
		exportTournamentUC,
		exportTournamentPdfUC,
		movePlayerUC,
		createTeamUC,
		deleteTeamUC,
		assignPlayerToTeamUC,
		removePlayerFromTeamUC,
		getTournamentsUC,
		tournament.NewGetOccupiedTablesUseCase(matchRepo),
		regenerateSeedsUC,
		updateParticipantEloUC,
		tournament.NewRemoveParticipantUseCase(tournamentRepo),
	)
	eventRepo := bun.NewEventRepository(bun.DB, tournamentRepo)
	exportEventPdfUC := tournament.NewExportEventPdfUseCase(eventRepo, divisionRepo, pdfGenerator)
	createEventUC := event.NewCreateEventUseCase(eventRepo, tournamentRepo, playerRepo, divisionRepo)
	getEventByIDUC := event.NewGetEventByIDUseCase(eventRepo)
	getAllEventsUC := event.NewGetAllEventsUseCase(eventRepo)
	deleteEventUC := event.NewDeleteEventUseCase(eventRepo)
	updateEventUC := event.NewUpdateEventUseCase(eventRepo)
	eventHandler := handler.NewEventHandler(createEventUC, updateEventUC, getEventByIDUC, getAllEventsUC, deleteEventUC, divisionUC, leaderboardUC, exportEventPdfUC)

	GetMatchesUC := match.NewGetMatchesUseCase(matchRepo)

	createMatchUC := match.NewCreateMatchUseCase(matchRepo, playerRepo, tournamentRepo, divisionRepo)
	finishMatchUC := match.NewFinishMatchUseCase()
	updateScoreUC := match.NewUpdateMatchScoreUseCase(matchRepo, tournamentRepo)
	matchHandler := handler.NewMatchHandler(createMatchUC, finishMatchUC, updateScoreUC, playerRepo, matchRepo, tournamentRepo, finishTournamentUC)

	leaderboardHandler := handler.NewLeaderboardHandler(leaderboardUC, divisionUC)
	divisionHandler := handler.NewDivisionHandler(divisionUC)
	selfRegisterUC := tournament.NewSelfRegisterUseCase(tournamentRepo, playerRepo)
	publicHandler := handler.NewPublicHandler(playerUC, selfRegisterUC)

	qrGenerator := qrinfra.NewGoQRCodeGenerator()
	qrHandler := handler.NewQRHandler(qrGenerator)

	adminRepo := bun.NewAdminRepository(bun.DB)

	hasher := security.NewBcryptHasher()

	// Seed default admin if DB empty
	count, _ := adminRepo.Count(context.Background())
	if count == 0 {
		user := cfg.AdminUsername
		pass := cfg.AdminPassword
		hashed, err := hasher.Hash(pass)
		if err == nil {
			if a, err := adminDomain.NewAdmin(idgen.Generate(), user, hashed); err == nil {
				adminRepo.Save(context.Background(), a)
			}
		}
	}

	authHandler := handler.NewAuthHandler(store, adminRepo, hasher)
	adminHandler := handler.NewAdminHandler(playerUC, createTournamentUC, createMatchUC, GetMatchesUC, leaderboardUC, getTournamentsUC, divisionUC, getAllEventsUC)

	return &Container{
		PlayerHandler:      playerHandler,
		TournamentHandler:  tournamentHandler,
		EventHandler:       eventHandler,
		MatchHandler:       matchHandler,
		LeaderboardHandler: leaderboardHandler,
		DivisionHandler:    divisionHandler,
		PublicHandler:      publicHandler,
		QRHandler:          qrHandler,
		AuthHandler:        authHandler,
		AdminHandler:       adminHandler,
	}
}
