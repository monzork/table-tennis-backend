package main

import (
	"context"

	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/event"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/application/match"
	"table-tennis-backend/internal/application/notification"
	"table-tennis-backend/internal/application/player"
	"table-tennis-backend/internal/application/tournament"
	adminDomain "table-tennis-backend/internal/domain/admin"
	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/domain/tournaments"
	pdfinfra "table-tennis-backend/internal/infrastructure/pdf"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
	qrinfra "table-tennis-backend/internal/infrastructure/qrcode"
	"table-tennis-backend/internal/infrastructure/security"
	"table-tennis-backend/internal/interfaces/http/handler"

	"github.com/gofiber/fiber/v2/middleware/session"
)

type Container struct {
	PlayerHandler       *handler.PlayerHandler
	EventHandler        *handler.EventHandler
	TournamentHandler   *handler.TournamentHandler
	MatchHandler        *handler.MatchHandler
	LeaderboardHandler  *handler.LeaderboardHandler
	DivisionHandler     *handler.DivisionHandler
	PublicHandler       *handler.PublicHandler
	QRHandler           *handler.QRHandler
	AuthHandler         *handler.AuthHandler
	AdminHandler        *handler.AdminHandler
	NotificationHandler *handler.NotificationHandler
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
	eventRepo := bun.NewEventRepository(bun.DB)
	tournamentRepo := bun.NewTournamentRepository(bun.DB, eventRepo)

	dispatcher := tournaments.NewInMemoryDispatcher()
	enrollPlayerUC := event.NewEnrollPlayerUseCase(eventRepo, dispatcher)
	getTournamentsUC := event.NewGetTournamentsUseCase(eventRepo)

	leaderboardUC := leaderboard.NewGetLeaderboardUseCase(playerRepo)

	divisionRepo := bun.NewDivisionRepository(bun.DB)
	divisionUC := division.NewDivisionUseCase(divisionRepo)

	createTournamentUC := event.NewCreateTournamentUseCase(eventRepo, playerRepo, divisionRepo)
	getTournamentByIDUC := event.NewGetTournamentByIDUseCase(eventRepo, divisionRepo)
	updateTournamentUC := event.NewUpdateTournamentUseCase(eventRepo, playerRepo, divisionRepo)
	deleteTournamentUC := event.NewDeleteTournamentUseCase(eventRepo)
	matchRepo := bun.NewMatchRepository(bun.DB, playerRepo)
	finishTournamentUC := event.NewFinishTournamentUseCase(eventRepo, matchRepo, playerRepo)
	recalculateTournamentEloUC := event.NewRecalculateTournamentEloUseCase(eventRepo, playerRepo)
	exportTournamentUC := event.NewExportTournamentReportUseCase(eventRepo)
	pdfGenerator := pdfinfra.NewGoFpdfGenerator()
	exportTournamentPdfUC := event.NewExportTournamentPdfUseCase(eventRepo, divisionRepo, pdfGenerator)
	movePlayerUC := event.NewMovePlayerUseCase(eventRepo)
	createTeamUC := event.NewCreateTeamUseCase(eventRepo)
	deleteTeamUC := event.NewDeleteTeamUseCase(eventRepo)
	assignPlayerToTeamUC := event.NewAssignPlayerToTeamUseCase(eventRepo)
	removePlayerFromTeamUC := event.NewRemovePlayerFromTeamUseCase(eventRepo)
	regenerateSeedsUC := event.NewRegenerateGroupSeedsUseCase(eventRepo, matchRepo, divisionRepo)
	dispatcher.Subscribe(tournaments.PlayerEnrolledEventName, func(ctx context.Context, e tournaments.Tournament) error {
		if pe, ok := e.(tournaments.PlayerEnrolledEvent); ok {
			_ = regenerateSeedsUC.Execute(ctx, pe.TournamentID)
		}
		return nil
	})
	updateParticipantEloUC := event.NewUpdateParticipantEloBeforeUseCase(eventRepo, regenerateSeedsUC)
	addGroupUC := event.NewAddGroupUseCase(eventRepo)
	playerHandler := handler.NewPlayerHandler(playerUC, updatePlayerUC, deletePlayerUC, getPlayerByIDUC, searchPlayerUC, searchPlayerSelectionUC, importPlayerUC, enrollPlayerUC, getTournamentsUC)

	tournamentHandler := handler.NewEventHandler(
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
		event.NewGetOccupiedTablesUseCase(matchRepo),
		regenerateSeedsUC,
		updateParticipantEloUC,
		event.NewRemoveParticipantUseCase(eventRepo),
		event.NewSaveKnockoutSeedsUseCase(eventRepo, divisionRepo),
		event.NewToggleSeedingLockUseCase(eventRepo),
		addGroupUC,
		recalculateTournamentEloUC,
		event.NewStartKnockoutStageUseCase(eventRepo, matchRepo, divisionRepo),
		event.NewGetEventDetailViewUseCase(getTournamentByIDUC, leaderboardUC, divisionUC),
		event.NewGetPublicEventDetailViewUseCase(getTournamentByIDUC, leaderboardUC, divisionUC),
		event.NewGetPublicTVDashboardViewUseCase(getTournamentByIDUC, leaderboardUC, divisionUC),
		event.NewGetBoardViewUseCase(getTournamentByIDUC, divisionUC),
		event.NewGetEditFormViewUseCase(getTournamentByIDUC, leaderboardUC, divisionUC),
	)
	exportEventPdfUC := event.NewExportEventPdfUseCase(tournamentRepo, divisionRepo, pdfGenerator)
	createEventUC := tournament.NewCreateEventUseCase(tournamentRepo, eventRepo, playerRepo, divisionRepo)
	getEventByIDUC := tournament.NewGetEventByIDUseCase(tournamentRepo)
	getAllEventsUC := tournament.NewGetAllEventsUseCase(tournamentRepo)
	deleteEventUC := tournament.NewDeleteEventUseCase(tournamentRepo)
	updateEventUC := tournament.NewUpdateEventUseCase(tournamentRepo)
	eventHandler := handler.NewTournamentHandler(createEventUC, updateEventUC, getEventByIDUC, getAllEventsUC, deleteEventUC, divisionUC, leaderboardUC, exportEventPdfUC)

	GetMatchesUC := match.NewGetMatchesUseCase(matchRepo)

	createMatchUC := match.NewCreateMatchUseCase(matchRepo, playerRepo, eventRepo, divisionRepo)
	finishMatchUC := match.NewFinishMatchUseCase()
	updateScoreUC := match.NewUpdateMatchScoreUseCase(matchRepo, eventRepo)
	teamMatchUC := match.NewTeamMatchOrchestratorUseCase(matchRepo)
	startMatchUC := match.NewStartMatchUseCase(matchRepo, eventRepo, tournamentRepo, createMatchUC)

	notificationRepo := bun.NewPushSubscriptionRepository(bun.DB)
	broadcastNotificationUC := notification.NewBroadcastPushNotificationUseCase(notificationRepo, cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey)

	matchHandler := handler.NewMatchHandler(createMatchUC, finishMatchUC, updateScoreUC, playerRepo, matchRepo, eventRepo, tournamentRepo, finishTournamentUC, broadcastNotificationUC, teamMatchUC, startMatchUC)

	leaderboardHandler := handler.NewLeaderboardHandler(leaderboardUC, divisionUC)
	divisionHandler := handler.NewDivisionHandler(divisionUC)
	selfRegisterUC := event.NewSelfRegisterUseCase(eventRepo, playerRepo)
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

	notificationHandler := handler.NewNotificationHandler(notificationRepo, cfg.VAPIDPublicKey, broadcastNotificationUC)
	return &Container{
		PlayerHandler:       playerHandler,
		EventHandler:        tournamentHandler,
		TournamentHandler:   eventHandler,
		MatchHandler:        matchHandler,
		LeaderboardHandler:  leaderboardHandler,
		DivisionHandler:     divisionHandler,
		PublicHandler:       publicHandler,
		QRHandler:           qrHandler,
		AuthHandler:         authHandler,
		AdminHandler:        adminHandler,
		NotificationHandler: notificationHandler,
	}
}
