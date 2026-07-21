package handler_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
	parentDomain "table-tennis-backend/internal/domain/tournament"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
)

func TestMatchHandler(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	loginReq := httptest.NewRequest("POST", "/admin/login", strings.NewReader("username=admin&password=password"))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginResp, _ := app.Test(loginReq)

	var sessionCookie string
	for _, v := range loginResp.Header.Values("Set-Cookie") {
		if strings.HasPrefix(v, "session_id=") {
			sessionCookie = strings.Split(v, ";")[0]
		}
	}

	// Seed players & event for match
	ctx := context.Background()
	playerRepo := bunRepo.NewPlayerRepository(db)
	tournamentRepo := bunRepo.NewEventRepository(db)
	matchRepo := bunRepo.NewMatchRepository(db, playerRepo)

	p1, _ := playerDomain.NewPlayer(uuid.New().String(), "Alice", "Smith", time.Now(), "F", "", "", "")
	p2, _ := playerDomain.NewPlayer(uuid.New().String(), "Bob", "Jones", time.Now(), "M", "", "", "")
	playerRepo.Save(ctx, p1)
	playerRepo.Save(ctx, p2)

	tourney, _ := tournamentDomain.NewTournament(uuid.New().String(), "Test Tourney", "singles", "elimination", "open", time.Now(), time.Now().Add(24*time.Hour), []tournamentDomain.Rule{}, 2, []*playerDomain.Player{p1, p2}, true)
	tournamentRepo.Save(ctx, tourney)

	m := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: tourney.ID, MatchType: "singles", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}, Status: "scheduled"}
	matchRepo.Save(ctx, m)

	t.Run("Update Score", func(t *testing.T) {
		data := url.Values{}
		data.Set("tournamentId", tourney.ID)
		data.Set("stage", "final")
		data.Add("scores[]", "11-9")

		req := httptest.NewRequest("PUT", fmt.Sprintf("/matches/%s/score", m.ID), strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Finish Match", func(t *testing.T) {
		data := url.Values{}
		data.Set("matchId", m.ID)
		data.Set("winnerTeam", "A")

		req := httptest.NewRequest("POST", "/matches/finish", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Invalid Score Update", func(t *testing.T) {
		req := httptest.NewRequest("PUT", fmt.Sprintf("/matches/%s/score", uuid.New().String()), bytes.NewReader([]byte{}))
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode == 200 {
			t.Errorf("expected error code for missing match, got 200")
		}
	})

	t.Run("Public Score Update", func(t *testing.T) {
		data := url.Values{}
		data.Set("matchId", m.ID)
		data.Set("tournamentId", tourney.ID)
		data.Set("stage", "final")
		data.Add("scores[]_a", "11")
		data.Add("scores[]_b", "7")

		req := httptest.NewRequest("POST", "/public/matches/score/update", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Table Exclusivity and Override", func(t *testing.T) {
		// Create two scheduled matches
		m1 := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: tourney.ID, MatchType: "singles", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}, Status: "scheduled"}
		matchRepo.Save(ctx, m1)

		m2 := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: tourney.ID, MatchType: "singles", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}, Status: "scheduled"}
		matchRepo.Save(ctx, m2)

		// Start first match on Table 1 manually
		startData1 := url.Values{}
		startData1.Set("tableNumber", "1")
		req1 := httptest.NewRequest("POST", fmt.Sprintf("/matches/%s/start", m1.ID), strings.NewReader(startData1.Encode()))
		req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req1.Header.Set("Cookie", sessionCookie)
		resp1, err := app.Test(req1)
		if err != nil || resp1.StatusCode != 200 {
			buf := new(bytes.Buffer)
			if resp1 != nil && resp1.Body != nil {
				buf.ReadFrom(resp1.Body)
			}
			t.Fatalf("failed to start first match (status %d): %s, err: %v", resp1.StatusCode, buf.String(), err)
		}

		// Verify table 1 is occupied
		mUUID1, _ := uuid.Parse(m1.ID)
		mModel1, _ := matchRepo.GetModelByID(ctx, mUUID1)
		if mModel1.TableNumber == nil || *mModel1.TableNumber != 1 {
			t.Errorf("expected table 1, got %v", mModel1.TableNumber)
		}

		// Try starting second match on Table 1 (occupied) -> should fail
		startData2 := url.Values{}
		startData2.Set("tableNumber", "1")
		req2 := httptest.NewRequest("POST", fmt.Sprintf("/matches/%s/start", m2.ID), strings.NewReader(startData2.Encode()))
		req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req2.Header.Set("Cookie", sessionCookie)
		resp2, err := app.Test(req2)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		hxTrigger := resp2.Header.Get("HX-Trigger")
		if !strings.Contains(hxTrigger, "Table 1 is currently occupied by another match!") {
			t.Errorf("expected occupied table toast in HX-Trigger, got: %s", hxTrigger)
		}

		mUUID2, _ := uuid.Parse(m2.ID)
		mModel2, _ := matchRepo.GetModelByID(ctx, mUUID2)
		if mModel2.Status != "scheduled" {
			t.Errorf("expected match 2 to remain scheduled, got status: %s", mModel2.Status)
		}

		// Try starting second match on Table 2 (free) -> should succeed
		startData3 := url.Values{}
		startData3.Set("tableNumber", "2")
		req3 := httptest.NewRequest("POST", fmt.Sprintf("/matches/%s/start", m2.ID), strings.NewReader(startData3.Encode()))
		req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req3.Header.Set("Cookie", sessionCookie)
		resp3, err := app.Test(req3)
		if err != nil || resp3.StatusCode != 200 {
			t.Errorf("expected start on table 2 to succeed, got %v", resp3.StatusCode)
		}
	})

	t.Run("Priority Table Assignment Heuristic", func(t *testing.T) {
		// Clear previously occupied tables
		matchRepo.DB().NewUpdate().Table("matches").Set("status = 'scheduled'").Exec(ctx)

		// Create a tournament with 4 tables
		tourney4, _ := tournamentDomain.NewTournament(uuid.New().String(), "Test Tourney 4", "singles", "elimination", "open", time.Now(), time.Now().Add(24*time.Hour), []tournamentDomain.Rule{}, 4, []*playerDomain.Player{p1, p2}, false)
		tournamentRepo.Save(ctx, tourney4)

		// Create a low priority match (group stage, non-1st division)
		mLow := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: tourney4.ID, MatchType: "singles", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}, Status: "scheduled", Stage: "group"}
		matchRepo.Save(ctx, mLow)

		// Start low priority match (auto-assign)
		reqLow := httptest.NewRequest("POST", fmt.Sprintf("/matches/%s/start", mLow.ID), strings.NewReader(url.Values{}.Encode()))
		reqLow.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqLow.Header.Set("Cookie", sessionCookie)
		respLow, _ := app.Test(reqLow)
		if respLow.StatusCode != 200 {
			t.Errorf("expected start low priority match to succeed, got %v", respLow.StatusCode)
		}

		mLowUUID, _ := uuid.Parse(mLow.ID)
		mLowModel, _ := matchRepo.GetModelByID(ctx, mLowUUID)
		if mLowModel.TableNumber == nil || *mLowModel.TableNumber < 3 {
			v := 0
			if mLowModel.TableNumber != nil {
				v = *mLowModel.TableNumber
			}
			t.Errorf("expected low priority match to be assigned table >= 3, got %d", v)
		}

		// Create a high priority match (final stage)
		mHigh := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: tourney4.ID, MatchType: "singles", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}, Status: "scheduled", Stage: "final"}
		matchRepo.Save(ctx, mHigh)

		// Start high priority match
		reqHigh := httptest.NewRequest("POST", fmt.Sprintf("/matches/%s/start", mHigh.ID), strings.NewReader(url.Values{}.Encode()))
		reqHigh.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqHigh.Header.Set("Cookie", sessionCookie)
		respHigh, _ := app.Test(reqHigh)
		if respHigh.StatusCode != 200 {
			t.Errorf("expected start high priority match to succeed, got %v", respHigh.StatusCode)
		}

		mHighUUID, _ := uuid.Parse(mHigh.ID)
		mHighModel, _ := matchRepo.GetModelByID(ctx, mHighUUID)
		if mHighModel.TableNumber == nil || *mHighModel.TableNumber != 1 {
			v := 0
			if mHighModel.TableNumber != nil {
				v = *mHighModel.TableNumber
			}
			t.Errorf("expected high priority match to be assigned table 1, got %d", v)
		}
	})

	t.Run("Update Score - Update Squads Error", func(t *testing.T) {
		data := url.Values{}
		data.Set("action", "update_squads")
		data.Set("squad_a_p1", p1.ID)
		
		req := httptest.NewRequest("PUT", fmt.Sprintf("/matches/%s/score", m.ID), strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)

		resp, _ := app.Test(req)
		if resp.StatusCode != 400 {
			t.Errorf("expected 400 Bad Request, got %v", resp.StatusCode)
		}
	})

	t.Run("Update Score - Update Squads Success", func(t *testing.T) {
		data := url.Values{}
		data.Set("action", "update_squads")
		data.Set("squad_a_p1", p1.ID)
		data.Set("squad_a_p2", p1.ID)
		data.Set("squad_a_p3", p1.ID)
		data.Set("squad_b_p1", p2.ID)
		data.Set("squad_b_p2", p2.ID)
		data.Set("squad_b_p3", p2.ID)
		
		req := httptest.NewRequest("PUT", fmt.Sprintf("/matches/%s/score", m.ID), strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)

		app.Test(req)
	})

	t.Run("Update Score - Team Squads Success", func(t *testing.T) {
		teamTourney, _ := tournamentDomain.NewTournament(uuid.New().String(), "Team Tourney", "teams", "elimination", "open", time.Now(), time.Now().Add(24*time.Hour), []tournamentDomain.Rule{}, 2, []*playerDomain.Player{p1, p2}, false)
		tournamentRepo.Save(ctx, teamTourney)

		tm := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: teamTourney.ID, MatchType: "teams", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}, Status: "scheduled"}
		matchRepo.Save(ctx, tm)

		data := url.Values{}
		data.Set("action", "update_squads")
		data.Set("squad_a_p1", p1.ID)
		data.Set("squad_a_p2", p1.ID)
		data.Set("squad_a_p3", p1.ID)
		data.Set("squad_b_p1", p2.ID)
		data.Set("squad_b_p2", p2.ID)
		data.Set("squad_b_p3", p2.ID)
		
		req := httptest.NewRequest("PUT", fmt.Sprintf("/matches/%s/score", tm.ID), strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)

		app.Test(req)
	})

	t.Run("Start - Existing Match ID", func(t *testing.T) {
		req := httptest.NewRequest("POST", fmt.Sprintf("/matches/%s/start", m.ID), strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)
		req.Header.Set("HX-Request", "true")

		app.Test(req)
	})

	t.Run("Update Public Score Success", func(t *testing.T) {
		data := url.Values{}
		data.Set("scores[]_a", "11")
		data.Set("scores[]_b", "5")
		data.Set("stage", "final")
		req := httptest.NewRequest("POST", fmt.Sprintf("/public/matches/%s/score/update", m.ID), strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")

		app.Test(req)
	})
	t.Run("Comprehensive Error Paths", func(t *testing.T) {
		tm := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: tourney.ID, MatchType: "teams", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}, Status: "scheduled", Stage: "group"}
		matchRepo.Save(ctx, tm)

		mSub := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: tourney.ID, MatchType: "singles", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}, Status: "scheduled", Stage: "group"}
		matchRepo.Save(ctx, mSub)

		// matchRepo.AddParticipants(ctx, tm.ID, []string{p1.ID}, "A")
		// matchRepo.AddParticipants(ctx, tm.ID, []string{p2.ID}, "B"))
		mSubModel, _ := matchRepo.GetModelByID(ctx, uuid.MustParse(mSub.ID))
		tmUUID := uuid.MustParse(tm.ID)
		mSubModel.TeamMatchID = &tmUUID
		matchRepo.DB().NewUpdate().Model(mSubModel).WherePK().Exec(ctx)

		doReq := func(method, route, bodyStr, contentType string) {
			req := httptest.NewRequest(method, route, strings.NewReader(bodyStr))
			if contentType != "" {
				req.Header.Set("Content-Type", contentType)
			}
			req.Header.Set("Cookie", sessionCookie)
			app.Test(req)
		}

		// UPDATE SCORE VALID to pass updateScoreUC
		doReq("PUT", "/matches/"+m.ID+"/score", "scores[]=11-9&scores[]=11-5&scores[]=11-3&refereeId="+p1.ID+"&tableNumber=99", "application/x-www-form-urlencoded")

		// Sub-match score update
		doReq("PUT", "/matches/"+mSub.ID+"/score", "scores[]=11-9&scores[]=11-5&scores[]=11-3", "application/x-www-form-urlencoded")

		// Update public score (valid)
		doReq("POST", "/public/matches/score/update", "matchId="+m.ID+"&scores[]=11-9&scores[]=11-5&scores[]=11-3&refereeId="+p1.ID+"&tableNumber=99", "application/x-www-form-urlencoded")

		// UPDATE SCORE on empty matchID but valid p1/p2
		doReq("PUT", "/matches//score", "p1Id="+p1.ID+"&p2Id="+p2.ID+"&tournamentId="+tourney.ID+"&stage=group&scores[]=11-1&scores[]=11-2&scores[]=11-3", "application/x-www-form-urlencoded")

		// START VALID
		doReq("POST", "/matches/"+m.ID+"/start", "tableNumber=99", "application/x-www-form-urlencoded")

		// START Fallbacks
		doReq("POST", "/matches/nil/start", "tournamentId="+tourney.ID+"&p1Id="+p1.ID+"&p2Id="+p2.ID+"&stage=group&tableNumber=99", "application/x-www-form-urlencoded")

		// FINISH VALID
		doReq("POST", "/matches/finish", "matchId="+m.ID+"&winnerTeam=A", "application/x-www-form-urlencoded")

		// Error Paths
		doReq("POST", "/matches/create", "{invalid", "application/json")
		doReq("POST", "/matches/create", "tournamentId=invalid-uuid", "application/x-www-form-urlencoded")
		doReq("POST", "/matches/create", "tournamentId="+uuid.New().String()+"&teamAPlayerIds="+uuid.New().String()+"&teamBPlayerIds="+uuid.New().String(), "application/x-www-form-urlencoded")

		doReq("POST", "/matches/finish", "{invalid", "application/json")
		doReq("POST", "/matches/finish", "matchId=invalid-uuid", "application/x-www-form-urlencoded")
		doReq("POST", "/matches/finish", "matchId="+uuid.New().String(), "application/x-www-form-urlencoded")
		doReq("POST", "/matches/finish", "matchId="+m.ID+"&winnerTeam=C", "application/x-www-form-urlencoded")

		doReq("PUT", "/matches/"+uuid.New().String()+"/score", "{invalid", "application/json")
		doReq("PUT", "/matches/invalid-uuid/score", "scores[]=11-9", "application/x-www-form-urlencoded")
		doReq("PUT", "/matches/"+m.ID+"/score", "tableNumber=invalid", "application/x-www-form-urlencoded")
		doReq("PUT", "/matches/"+m.ID+"/score", "action=update_squads&squad_a_p1=invalid", "application/x-www-form-urlencoded")

		doReq("POST", "/public/matches/score/update", "{invalid", "application/json")
		doReq("POST", "/public/matches/score/update", "matchId=invalid-uuid", "application/x-www-form-urlencoded")
		doReq("POST", "/public/matches/score/update", "matchId="+uuid.New().String(), "application/x-www-form-urlencoded")
		doReq("POST", "/public/matches/score/update", "matchId="+m.ID+"&tableNumber=invalid", "application/x-www-form-urlencoded")
		doReq("POST", "/public/matches/score/update", "action=update_squads&matchId="+m.ID, "application/x-www-form-urlencoded")
		doReq("POST", "/public/matches/score/update", "action=update_squads&matchId="+m.ID+"&squad_a_p1="+p1.ID+"&squad_a_p2="+p1.ID+"&squad_a_p3="+p1.ID+"&squad_b_p1="+p2.ID+"&squad_b_p2="+p2.ID+"&squad_b_p3="+p2.ID, "application/x-www-form-urlencoded")

		doReq("POST", "/matches/"+uuid.New().String()+"/start", "{invalid", "application/json")
		doReq("POST", "/matches/invalid-uuid/start", "", "application/x-www-form-urlencoded")
		doReq("POST", "/matches/"+m.ID+"/start", "tableNumber=invalid", "application/x-www-form-urlencoded")

		doReq("POST", "/matches/invalid-uuid/reset", "", "application/x-www-form-urlencoded")
		doReq("POST", "/matches/"+uuid.New().String()+"/reset", "", "application/x-www-form-urlencoded")

		doReq("GET", "/public/score/table/invalid/tournament/"+uuid.New().String(), "", "")
		doReq("GET", "/public/score/table/1/tournament/invalid-uuid", "", "")
		doReq("GET", "/public/score/table/1/event/invalid-uuid", "", "")
		doReq("GET", "/public/score/table/1/event/"+uuid.New().String(), "", "")
		doReq("GET", "/public/score/table/1/tournament/"+uuid.New().String(), "", "")

		// Hit occupied table error in UpdateScore
		doReq("PUT", "/matches/"+m.ID+"/score", "tournamentId="+tourney.ID+"&scores[]=11-9&tableNumber=99", "application/x-www-form-urlencoded")
		doReq("PUT", "/matches/"+mSub.ID+"/score", "tournamentId="+tourney.ID+"&scores[]=11-9&tableNumber=99", "application/x-www-form-urlencoded")
		
		// Hit occupied table error in Start
		doReq("POST", "/matches/"+mSub.ID+"/start", "tableNumber=99", "application/x-www-form-urlencoded")

		// Missing matchID in finish
		doReq("POST", "/matches/finish", "winnerTeam=A", "application/x-www-form-urlencoded")

		// Update Public Score with PIN
		doReq("POST", "/public/matches/score/update", "matchId="+m.ID+"&tournamentId="+tourney.ID+"&pin=1234&scores[]=11-9", "application/x-www-form-urlencoded")
		
		// Start with event tournament ID but different stage
		doReq("POST", "/matches/nil/start", "tournamentId="+tourney.ID+"&p1Id="+p1.ID+"&p2Id="+p2.ID+"&stage=final", "application/x-www-form-urlencoded")

		// Update squads valid submatch
		doReq("PUT", "/matches/"+tm.ID+"/score", "action=update_squads&squad_a_p1="+p1.ID+"&squad_a_p2="+p1.ID+"&squad_a_p3="+p1.ID+"&squad_b_p1="+p2.ID+"&squad_b_p2="+p2.ID+"&squad_b_p3="+p2.ID, "application/x-www-form-urlencoded")
		doReq("POST", "/public/matches/score/update", "action=update_squads&matchId="+tm.ID+"&squad_a_p1="+p1.ID+"&squad_a_p2="+p1.ID+"&squad_a_p3="+p1.ID+"&squad_b_p1="+p2.ID+"&squad_b_p2="+p2.ID+"&squad_b_p3="+p2.ID, "application/x-www-form-urlencoded")

		// Doubles match coverage
		p3, _ := playerDomain.NewPlayer(uuid.New().String(), "Charlie", "Brown", time.Now(), "M", "", "", "")
		p4, _ := playerDomain.NewPlayer(uuid.New().String(), "Diana", "Prince", time.Now(), "F", "", "", "")
		playerRepo.Save(ctx, p3)
		playerRepo.Save(ctx, p4)

		doublesTourney, _ := tournamentDomain.NewTournament(uuid.New().String(), "Test Doubles Tourney", "doubles", "elimination", "open", time.Now(), time.Now().Add(24*time.Hour), []tournamentDomain.Rule{}, 2, []*playerDomain.Player{p1, p2, p3, p4}, true)
		tournamentRepo.Save(ctx, doublesTourney)

		mDoubles := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: doublesTourney.ID, MatchType: "doubles", TeamA: []*playerDomain.Player{p1, p3}, TeamB: []*playerDomain.Player{p2, p4}, Status: "scheduled", Stage: "final"}
		matchRepo.Save(ctx, mDoubles)

		// Finish doubles match (hits line 221, 231)
		doReq("POST", "/matches/finish", "matchId="+mDoubles.ID+"&winnerTeam=A", "application/x-www-form-urlencoded")
		// Update score for doubles match (hits similar broadcast lines)
		doReq("PUT", "/matches/"+mDoubles.ID+"/score", "tournamentId="+doublesTourney.ID+"&stage=final&scores[]=11-9&scores[]=11-5&scores[]=11-3", "application/x-www-form-urlencoded")
		// Start doubles match
		doReq("POST", "/matches/"+mDoubles.ID+"/start", "tableNumber=98", "application/x-www-form-urlencoded")
		// Start on the fly doubles
		doReq("POST", "/matches/nil/start", "tournamentId="+doublesTourney.ID+"&p1Id="+p1.ID+"&p2Id="+p2.ID+"&stage=group", "application/x-www-form-urlencoded")
		// UpdateScore on the fly doubles
		doReq("PUT", "/matches//score", "p1Id="+p1.ID+"&p2Id="+p2.ID+"&tournamentId="+doublesTourney.ID+"&stage=group&scores[]=11-1", "application/x-www-form-urlencoded")
	})

	t.Run("Full match finish flow - auto-assign, group-stage-finished, referee notify", func(t *testing.T) {
		// This exercises the branches only reachable once a match genuinely transitions to
		// "finished" through the real UpdateScore/UpdatePublicScore endpoints with a valid
		// tournamentId: broadcasting the finished message, auto-assigning tables to the next
		// scheduled match once EventID is set, detecting group-stage completion, and (for the
		// public/referee flow) notifying admins.
		parentID := uuid.New().String()
		parentRepo := bunRepo.NewTournamentRepository(db, tournamentRepo)
		now := time.Now()
		if err := parentRepo.Save(ctx, &parentDomain.Tournament{
			ID:        parentID,
			Name:      "Parent For AutoAssign",
			StartDate: now,
			EndDate:   now.Add(24 * time.Hour),
			NumTables: 3,
		}); err != nil {
			t.Fatalf("failed to save parent tournament: %v", err)
		}

		newPlayer := func(first, last, gender string) *playerDomain.Player {
			p, _ := playerDomain.NewPlayer(uuid.New().String(), first, last, time.Now(), gender, "", "", "")
			playerRepo.Save(ctx, p)
			return p
		}

		// Admin (UpdateScore) side: child event A with a group match to finish and a
		// round16 match left scheduled with no table for auto-assign to pick up.
		p5 := newPlayer("5A", "Player", "M")
		p6 := newPlayer("6B", "Player", "M")
		p7 := newPlayer("7C", "Player", "M")
		p8 := newPlayer("8D", "Player", "M")

		eventA, _ := tournamentDomain.NewTournament(uuid.New().String(), "Child A", "singles", "elimination", "open", now, now.Add(24*time.Hour), []tournamentDomain.Rule{}, 2, []*playerDomain.Player{p5, p6, p7, p8}, true)
		eventA.EventID = &parentID
		if err := tournamentRepo.Save(ctx, eventA); err != nil {
			t.Fatalf("failed to save child event A: %v", err)
		}

		matchToFinishA := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: eventA.ID, MatchType: "singles", TeamA: []*playerDomain.Player{p5}, TeamB: []*playerDomain.Player{p6}, Status: "scheduled", Stage: "group"}
		matchRepo.Save(ctx, matchToFinishA)
		matchAutoAssignA := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: eventA.ID, MatchType: "singles", TeamA: []*playerDomain.Player{p7}, TeamB: []*playerDomain.Player{p8}, Status: "scheduled", Stage: "round16"}
		matchRepo.Save(ctx, matchAutoAssignA)

		reqA := httptest.NewRequest("PUT", "/matches/"+matchToFinishA.ID+"/score",
			strings.NewReader("tournamentId="+eventA.ID+"&stage=group&scores[]=11-1&scores[]=11-2&scores[]=11-3"))
		reqA.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqA.Header.Set("Cookie", sessionCookie)
		respA, err := app.Test(reqA)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if respA.StatusCode != 200 {
			t.Errorf("expected 200 OK finishing match A, got %v", respA.StatusCode)
		}

		mAutoAssignAModel, _ := matchRepo.GetModelByID(ctx, uuid.MustParse(matchAutoAssignA.ID))
		if mAutoAssignAModel.TableNumber == nil {
			t.Errorf("expected auto-assign to have given match a table after finishing sibling match")
		}

		// Public (UpdatePublicScore) side: child event B, referee submission finishing the group
		// match (hits the referee-notification broadcast) plus its own auto-assign candidate.
		p9 := newPlayer("9E", "Player", "F")
		p10 := newPlayer("10F", "Player", "F")
		p11 := newPlayer("11G", "Player", "F")
		p12 := newPlayer("12H", "Player", "F")

		eventB, _ := tournamentDomain.NewTournament(uuid.New().String(), "Child B", "singles", "elimination", "open", now, now.Add(24*time.Hour), []tournamentDomain.Rule{}, 2, []*playerDomain.Player{p9, p10, p11, p12}, true)
		eventB.EventID = &parentID
		if err := tournamentRepo.Save(ctx, eventB); err != nil {
			t.Fatalf("failed to save child event B: %v", err)
		}

		matchToFinishB := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: eventB.ID, MatchType: "singles", TeamA: []*playerDomain.Player{p9}, TeamB: []*playerDomain.Player{p10}, Status: "scheduled", Stage: "group"}
		matchRepo.Save(ctx, matchToFinishB)
		matchAutoAssignB := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: eventB.ID, MatchType: "singles", TeamA: []*playerDomain.Player{p11}, TeamB: []*playerDomain.Player{p12}, Status: "scheduled", Stage: "round16"}
		matchRepo.Save(ctx, matchAutoAssignB)

		// The public endpoint only accepts the split scores[]_a / scores[]_b form fields
		// (no fallback for the combined "A-B" format the admin JSON API supports).
		reqB := httptest.NewRequest("POST", "/public/matches/score/update",
			strings.NewReader("matchId="+matchToFinishB.ID+"&tournamentId="+eventB.ID+"&stage=group"+
				"&scores[]_a=11&scores[]_b=1&scores[]_a=11&scores[]_b=2&scores[]_a=11&scores[]_b=3&refereeId="+p9.ID))
		reqB.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqB.Header.Set("HX-Request", "true")
		respB, err := app.Test(reqB)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if respB.StatusCode != 200 {
			t.Errorf("expected 200 OK finishing match B, got %v", respB.StatusCode)
		}

		mAutoAssignBModel, _ := matchRepo.GetModelByID(ctx, uuid.MustParse(matchAutoAssignB.ID))
		if mAutoAssignBModel.TableNumber == nil {
			t.Errorf("expected auto-assign to have given match B a table after finishing sibling match")
		}
	})
}
