package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	appEvent "table-tennis-backend/internal/application/event"
	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
	"table-tennis-backend/internal/interfaces/http/handler"
)

func TestEventHandlerDirectly(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	sessionCookie := getSessionCookie(app)
	
	p1, _ := playerDomain.NewPlayer(uuid.New().String(), "Test", "P1", time.Now(), "M", "", "", "")
	p2, _ := playerDomain.NewPlayer(uuid.New().String(), "Test", "P2", time.Now(), "M", "", "", "")
	playerRepo := bunRepo.NewPlayerRepository(db)
	playerRepo.Save(context.Background(), p1)
	playerRepo.Save(context.Background(), p2)

	tourney, _ := tournamentDomain.NewTournament(uuid.New().String(), "Comp Test Event", "singles", "elimination", "open", time.Now(), time.Now(), []tournamentDomain.Rule{}, 2, []*playerDomain.Player{p1, p2}, false)
	tournamentRepo := bunRepo.NewEventRepository(db)
	tournamentRepo.Save(context.Background(), tourney)
	validID := tourney.ID

	teamID := uuid.New()
	tourneyUUID, _ := uuid.Parse(validID)
	db.NewInsert().Model(&bunRepo.TeamModel{ID: teamID, Name: "Test Team", TournamentID: tourneyUUID}).Exec(context.Background())

	tests := []struct {
		name   string
		method string
		route  string
		body   string
	}{
		{"Update with Bad ID", "PUT", "/events/bad-id", "name=BadEvent"},
		{"Finish with Bad ID", "POST", "/admin/events/bad-id/finish", ""},
		{"SaveKnockoutSeeds with Bad ID", "POST", "/admin/events/bad-id/divisions/bad-div/knockout/seeds", "divId=bad-div&playerIds=1,2"},
		{"AddGroup with Bad ID", "POST", "/admin/events/bad-id/groups", "divisionName=Open"},
		{"AssignPlayerToTeam with Bad ID", "POST", "/events/bad-id/teams/bad-team/players", "playerId=bad-player"},
		{"DeleteTeam with Bad ID", "DELETE", "/events/bad-id/teams/bad-team", ""},
		{"RemovePlayerFromTeam with Bad ID", "DELETE", "/events/bad-id/teams/bad-team/players/bad-player", ""},
		{"UpdateParticipantEloBefore with Bad ID", "POST", "/admin/events/bad-id/participants/elo-before", "playerId=bad-player"},
		{"UpdateParticipantEloBefore with Missing Body", "POST", "/admin/events/bad-id/participants/elo-before", ""},
		{"UpdateParticipantEloBefore with Missing PlayerID", "POST", "/admin/events/bad-id/participants/elo-before", "singlesElo=1500"},
		{"AddOfficial with Bad ID", "POST", "/admin/events/bad-id/officials", "playerId=bad-player"},
		{"ShowEditForm with Bad ID", "GET", "/admin/events/bad-id/edit", ""},
		{"CreateTeam with Bad ID", "POST", "/events/bad-id/teams", "name=BadTeam"},
		{"RecalculateElo with Bad ID", "POST", "/admin/events/bad-id/recalculate-elo", ""},
		{"StartKnockout with Bad ID", "POST", "/admin/events/bad-id/divisions/bad-div/start-knockout", ""},
		{"Detail with Bad ID", "GET", "/admin/events/bad-id", ""},
		{"RemoveOfficial with Bad ID", "DELETE", "/admin/events/bad-id/officials/bad-player", ""},
		{"RemoveParticipant with Bad ID", "DELETE", "/admin/events/bad-id/participants/bad-player", ""},
		{"Delete with Bad ID", "DELETE", "/events/bad-id", ""},
		{"RegenerateGroupSeeds with Bad ID", "POST", "/admin/events/bad-id/regenerate-seeds", ""},
		{"Export with Bad ID", "GET", "/admin/events/bad-id/export", ""},
		{"ExportPDF with Bad ID", "GET", "/admin/events/bad-id/export/pdf", ""},
		{"MovePlayer with Bad ID", "POST", "/admin/events/bad-id/move-player", "playerId=bad-player&targetGroupId=bad-group"},
		{"PublicDetail with Bad ID", "GET", "/public/events/bad-id", ""},
		{"PublicTVDashboard with Bad ID", "GET", "/public/events/bad-id/tv", ""},
		{"PublicList trigger error", "GET", "/public/events", ""},
		{"Board with Bad ID", "GET", "/events/bad-id/board", ""},
		{"BoardColumns with Bad ID", "GET", "/events/bad-id/board/columns", ""},
        
		{"DELETE team", "DELETE", "/events/" + validID + "/teams/" + teamID.String(), ""},
		{"POST create team", "POST", "/events/" + validID + "/teams", "name=SuccessTeam"},
		{"POST assign player", "POST", "/events/" + validID + "/teams/" + teamID.String() + "/players", "playerId=" + p1.ID},
		{"DELETE player from team", "DELETE", "/events/" + validID + "/teams/" + teamID.String() + "/players/" + p1.ID, ""},
		{"POST finish", "POST", "/admin/events/" + validID + "/finish", ""},
		{"POST add group", "POST", "/admin/events/" + validID + "/groups", "divisionName=Open2"},
		{"POST recalc elo", "POST", "/admin/events/" + validID + "/recalculate-elo", ""},
		{"GET edit form", "GET", "/admin/events/" + validID + "/edit", ""},
		{"POST add official", "POST", "/admin/events/" + validID + "/officials", "playerId=" + p2.ID},
		{"GET detail", "GET", "/admin/events/" + validID, ""},
		{"GET public list", "GET", "/public/events", ""},
		{"GET public detail", "GET", "/public/events/" + validID, ""},
		{"GET public tv", "GET", "/public/events/" + validID + "/tv", ""},
		{"GET board", "GET", "/events/" + validID + "/board", ""},
		{"GET board columns", "GET", "/events/" + validID + "/board/columns", ""},
		{"POST toggle lock", "POST", "/admin/events/" + validID + "/toggle-seeding-lock", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.method == "GET" || tt.method == "DELETE" {
				req = httptest.NewRequest(tt.method, tt.route, nil)
			} else {
				req = httptest.NewRequest(tt.method, tt.route, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			req.Header.Set("Cookie", sessionCookie)
			req.Header.Set("HX-Request", "true")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			_ = resp
            
            req2 := httptest.NewRequest(tt.method, tt.route, strings.NewReader(tt.body))
            if tt.body != "" {
				req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
            req2.Header.Set("Cookie", sessionCookie)
            app.Test(req2)
		})
	}
    
    bodyParserEndpoints := []string{
		"/admin/events/bad-id/divisions/bad-div/knockout/seeds",
		"/admin/events/bad-id/groups",
		"/events/bad-id/teams/bad-team/players",
		"/admin/events/bad-id/participants/elo-before",
		"/admin/events/bad-id/officials",
		"/events/bad-id/teams",
		"/admin/events/bad-id/move-player",
		"/events/bad-id", 
		"/events", 
	}

	for _, ep := range bodyParserEndpoints {
		req := httptest.NewRequest("POST", ep, bytes.NewBufferString("{invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", sessionCookie)
		app.Test(req)

		reqPut := httptest.NewRequest("PUT", ep, bytes.NewBufferString("{invalid json"))
		reqPut.Header.Set("Content-Type", "application/json")
		reqPut.Header.Set("Cookie", sessionCookie)
		app.Test(reqPut)
	}

	t.Run("Create Error Path", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/events", strings.NewReader("bad data"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", sessionCookie)
		app.Test(req)

		req2 := httptest.NewRequest("POST", "/events", strings.NewReader("name=EmptyType"))
		req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req2.Header.Set("Cookie", sessionCookie)
		app.Test(req2)
	})

	t.Run("BuildBoardCards Coverage", func(t *testing.T) {
		p1 := &playerDomain.Player{ID: "p1"}
		p2 := &playerDomain.Player{ID: "p2"}

		evGroups := &tournamentDomain.Event{
			ID:     "e1",
			Format: "groups_elimination",
			Type:   "singles",
			Matches: []tournamentDomain.Match{
				{
					ID:     "m1",
					Status: "in_progress",
					Stage:  "group",
					TeamA:  []*playerDomain.Player{p1},
					TeamB:  []*playerDomain.Player{p2},
				},
				{
					ID:     "m2",
					Status: "finished",
					Stage:  "knockout",
					TeamA:  []*playerDomain.Player{},
					TeamB:  []*playerDomain.Player{},
				},
			},
			Groups: []tournamentDomain.Group{
				{
					ID:      "g1",
					Name:    "Group 1 - Foo",
					Players: []*playerDomain.Player{p1, p2},
				},
			},
			Participants: []*playerDomain.Player{p1, p2},
		}

		evRoundRobin := &tournamentDomain.Event{
			ID:     "e2",
			Format: "round_robin",
			Type:   "singles",
			Participants: []*playerDomain.Player{p1, p2},
			Matches: []tournamentDomain.Match{
				{
					ID:     "m3",
					Status: "scheduled",
					Stage:  "group",
					TeamA:  []*playerDomain.Player{p1},
					TeamB:  []*playerDomain.Player{p2},
				},
			},
		}

		evElimination := &tournamentDomain.Event{
			ID:     "e3",
			Format: "elimination",
			Type:   "singles",
			Participants: []*playerDomain.Player{p1, p2},
		}

		maxElo := int16(2000)
		divs := []*divisionDomain.Division{
			{Name: "Div 1", MinElo: 0, MaxElo: &maxElo},
			{Name: "Div 2", MinElo: 1000},
		}

		handler.BuildBoardCards(evGroups, divs)
		handler.BuildBoardCards(evRoundRobin, divs)
		handler.BuildBoardCards(evElimination, divs)

		handler.FilterBoardCards([]appEvent.BoardCard{}, "", nil)
		handler.FilterBoardCards([]appEvent.BoardCard{{PlayerAName: "foo", DivisionName: "Div 1"}}, "foo", []string{"Div 1"})
		handler.FilterBoardCards([]appEvent.BoardCard{{PlayerAName: "foo", DivisionName: "Div 1"}}, "", []string{"Div 2"})
	})
}
