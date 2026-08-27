package main

import (
	"context"

	accountApp "table-tennis-backend/internal/application/account"
	"table-tennis-backend/internal/application/dashboard"
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
	oauthinfra "table-tennis-backend/internal/infrastructure/oauth"
	pdfinfra "table-tennis-backend/internal/infrastructure/pdf"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
	qrinfra "table-tennis-backend/internal/infrastructure/qrcode"
	"table-tennis-backend/internal/infrastructure/security"
	storageinfra "table-tennis-backend/internal/infrastructure/storage"
	svgchartinfra "table-tennis-backend/internal/infrastructure/svgchart"
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
	DashboardHandler    *handler.DashboardHandler
	AccountHandler      *handler.AccountHandler
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

	createTournamentUC := event.NewCreateTournamentUseCase(eventRepo, playerRepo)
	getTournamentByIDUC := event.NewGetTournamentByIDUseCase(eventRepo)
	updateTournamentUC := event.NewUpdateTournamentUseCase(eventRepo, playerRepo)
	deleteTournamentUC := event.NewDeleteTournamentUseCase(eventRepo)
	matchRepo := bun.NewMatchRepository(bun.DB, playerRepo)
	finishTournamentUC := event.NewFinishTournamentUseCase(eventRepo, matchRepo, playerRepo)
	recalculateTournamentEloUC := event.NewRecalculateTournamentEloUseCase(eventRepo, matchRepo, playerRepo)
	exportTournamentUC := event.NewExportTournamentReportUseCase(eventRepo)
	pdfGenerator := pdfinfra.NewGoFpdfGenerator()
	exportTournamentPdfUC := event.NewExportTournamentPdfUseCase(eventRepo, divisionRepo, pdfGenerator)
	movePlayerUC := event.NewMovePlayerUseCase(eventRepo)
	createTeamUC := event.NewCreateTeamUseCase(eventRepo)
	deleteTeamUC := event.NewDeleteTeamUseCase(eventRepo)
	assignPlayerToTeamUC := event.NewAssignPlayerToTeamUseCase(eventRepo)
	removePlayerFromTeamUC := event.NewRemovePlayerFromTeamUseCase(eventRepo)
	regenerateSeedsUC := event.NewRegenerateGroupSeedsUseCase(eventRepo, matchRepo)
	dispatcher.Subscribe(tournaments.PlayerEnrolledEventName, func(ctx context.Context, e tournaments.Tournament) error {
		if pe, ok := e.(tournaments.PlayerEnrolledEvent); ok {
			_ = regenerateSeedsUC.Execute(ctx, pe.EventID)
		}
		return nil
	})
	updateParticipantEloUC := event.NewUpdateParticipantEloBeforeUseCase(eventRepo, regenerateSeedsUC)
	addGroupUC := event.NewAddGroupUseCase(eventRepo)
	chartGenerator := svgchartinfra.NewSVGGenerator()
	getPlayerStatsUC := player.NewGetPlayerTournamentStatsUseCase(playerRepo, eventRepo, tournamentRepo)
	getEloTrendUC := player.NewGetPlayerEloTrendUseCase(chartGenerator)
	playerHandler := handler.NewPlayerHandler(playerUC, updatePlayerUC, deletePlayerUC, getPlayerByIDUC, searchPlayerUC, searchPlayerSelectionUC, importPlayerUC, enrollPlayerUC, getTournamentsUC, getPlayerStatsUC, getEloTrendUC)
	if cfg.SupabaseURL != "" && cfg.SupabaseKey != "" {
		playerHandler = playerHandler.WithUploader(storageinfra.NewSupabaseStorage(cfg.SupabaseURL, cfg.SupabaseKey, cfg.SupabaseBucket))
	}

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
	exportEventPdfUC := event.NewExportEventPdfUseCase(tournamentRepo, eventRepo, divisionRepo, pdfGenerator)
	createEventUC := tournament.NewCreateEventUseCase(tournamentRepo, eventRepo, playerRepo, divisionRepo)
	getEventByIDUC := tournament.NewGetEventByIDUseCase(tournamentRepo)
	getAllEventsUC := tournament.NewGetAllEventsUseCase(tournamentRepo)
	deleteEventUC := tournament.NewDeleteEventUseCase(tournamentRepo)
	updateEventUC := tournament.NewUpdateEventUseCase(tournamentRepo)
	getBoardUC := tournament.NewGetBoardDataUseCase(tournamentRepo, divisionRepo)
	autoAssignTablesUC := match.NewAutoAssignTablesUseCase(matchRepo, tournamentRepo)
	eventHandler := handler.NewTournamentHandler(createEventUC, updateEventUC, getEventByIDUC, getAllEventsUC, deleteEventUC, divisionUC, leaderboardUC, exportEventPdfUC, getBoardUC, autoAssignTablesUC)

	GetMatchesUC := match.NewGetMatchesUseCase(matchRepo)

	createMatchUC := match.NewCreateMatchUseCase(matchRepo, playerRepo, eventRepo, divisionRepo)
	finishMatchUC := match.NewFinishMatchUseCase()
	updateScoreUC := match.NewUpdateMatchScoreUseCase(matchRepo, eventRepo)
	teamMatchUC := match.NewTeamMatchOrchestratorUseCase(matchRepo)
	startMatchUC := match.NewStartMatchUseCase(matchRepo, eventRepo, tournamentRepo, createMatchUC)

	notificationRepo := bun.NewPushSubscriptionRepository(bun.DB)
	broadcastNotificationUC := notification.NewBroadcastPushNotificationUseCase(notificationRepo, cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey)

	matchHandler := handler.NewMatchHandler(createMatchUC, finishMatchUC, updateScoreUC, playerRepo, matchRepo, eventRepo, tournamentRepo, finishTournamentUC, broadcastNotificationUC, teamMatchUC, startMatchUC, divisionRepo)

	distUC := leaderboard.NewGetDivisionDistributionUseCase(chartGenerator)
	leaderboardHandler := handler.NewLeaderboardHandler(leaderboardUC, divisionUC, distUC)
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

	dashboardRepo := bun.NewDashboardRepository(bun.DB)
	getDashboardViewUC := dashboard.NewGetPublicDashboardViewUseCase(dashboardRepo, chartGenerator)
	dashboardHandler := handler.NewDashboardHandler(getDashboardViewUC)

	// ── Guardian accounts (Google OAuth) + player self-service score
	// confirmation — entirely separate area from /admin, see
	// internal/interfaces/http/middleware/account_auth.go.
	accountRepo := bun.NewAccountRepository(bun.DB)
	googleClient := oauthinfra.NewGoogleClient(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)

	loginWithGoogleUC := accountApp.NewLoginWithGoogleUseCase(accountRepo)
	getAccountByIDUC := accountApp.NewGetAccountByIDUseCase(accountRepo)
	updateAccountUC := accountApp.NewUpdateAccountUseCase(accountRepo)
	createChildPlayerUC := accountApp.NewCreateChildPlayerUseCase(playerRepo)
	getLinkedPlayersUC := accountApp.NewGetLinkedPlayersUseCase(playerRepo)
	getPlayerPendingMatchesUC := player.NewGetPlayerPendingMatchesUseCase(eventRepo, divisionRepo)
	getGuardianPendingMatchesUC := accountApp.NewGetGuardianPendingMatchesUseCase(getLinkedPlayersUC, getPlayerPendingMatchesUC)
	assignPlayerToAccountUC := accountApp.NewAssignPlayerToAccountUseCase(playerRepo, accountRepo)
	claimPlayerUC := accountApp.NewClaimPlayerUseCase(playerRepo)
	searchClaimableUC := accountApp.NewSearchClaimablePlayersUseCase(playerRepo)
	getPendingClaimsUC := accountApp.NewGetPendingPlayerClaimsUseCase(playerRepo, accountRepo)
	approveClaimUC := accountApp.NewApprovePlayerClaimUseCase(playerRepo)
	rejectClaimUC := accountApp.NewRejectPlayerClaimUseCase(playerRepo)

	proposeMatchScoreUC := match.NewProposeMatchScoreUseCase(matchRepo, eventRepo, playerRepo)
	confirmMatchScoreUC := match.NewConfirmMatchScoreUseCase(matchRepo, eventRepo, updateScoreUC)
	rejectMatchScoreProposalUC := match.NewRejectMatchScoreProposalUseCase(matchRepo)
	matchHandler.WithConfirmMatchScoreUseCase(confirmMatchScoreUC)

	accountHandler := handler.NewAccountHandler(
		store,
		googleClient,
		loginWithGoogleUC,
		getAccountByIDUC,
		updateAccountUC,
		createChildPlayerUC,
		getLinkedPlayersUC,
		getGuardianPendingMatchesUC,
		getPlayerByIDUC,
		updatePlayerUC,
		getPlayerPendingMatchesUC,
		proposeMatchScoreUC,
		confirmMatchScoreUC,
		rejectMatchScoreProposalUC,
	)
	playerHandler.WithAssignPlayerToAccountUseCase(assignPlayerToAccountUC)
	playerHandler.WithGetPlayerRankUseCase(player.NewGetPlayerRankUseCase(playerRepo))
	accountHandler.WithGetPlayerStatsUseCase(getPlayerStatsUC)
	accountHandler.WithClaimUseCases(claimPlayerUC, searchClaimableUC)
	adminHandler.WithClaimReviewUseCases(getPendingClaimsUC, approveClaimUC, rejectClaimUC)

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
		DashboardHandler:    dashboardHandler,
		AccountHandler:      accountHandler,
	}
}
