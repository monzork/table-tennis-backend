package match

import (
	"context"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

type SubMatchVM struct {
	ID             string
	MatchType      string
	RoundNumber    int
	TeamAPlayer1ID string
	TeamAPlayer2ID string
	TeamBPlayer1ID string
	TeamBPlayer2ID string
	PlayerAName    string
	PlayerBName    string
	AlignmentA     string
	AlignmentB     string
	ScoreA         int
	ScoreB         int
	Status         string
	WinnerTeam     string
}

type TeamMatchFormView struct {
	MatchID      string
	TournamentID string
	Stage        string
	BestOf       int
	TeamA        *event.Team
	TeamB        *event.Team
	TeamFormat   string
	SubMatches   []SubMatchVM
	SquadAP1     string
	SquadAP2     string
	SquadAP3     string
	SquadBP1     string
	SquadBP2     string
	SquadBP3     string
	Participants []*player.Player
	Pin          string
	RefereeID    string
	TableNumber  *int
}

type GetTeamMatchFormViewUseCase struct {
	matchRepo      *bun.MatchRepository
	tournamentRepo *bun.TournamentRepository
}

func NewGetTeamMatchFormViewUseCase(matchRepo *bun.MatchRepository, tournamentRepo *bun.TournamentRepository) *GetTeamMatchFormViewUseCase {
	return &GetTeamMatchFormViewUseCase{
		matchRepo:      matchRepo,
		tournamentRepo: tournamentRepo,
	}
}

func (uc *GetTeamMatchFormViewUseCase) Execute(ctx context.Context, matchID, tournamentID, stage string) (*TeamMatchFormView, error) {
	parentUUID, _ := uuid.Parse(matchID)

	t, err := uc.tournamentRepo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	parent, err := uc.matchRepo.GetModelByID(ctx, parentUUID)
	if err != nil {
		return nil, err
	}

	bestOf := 5
	if t != nil {
		bestOf = t.GetEffectiveStageRule(stage, parent.DivisionID).BestOf
	}

	teamFormat := t.TeamFormat
	if teamFormat == "" {
		teamFormat = "olympic"
	}

	var teamA, teamB *event.Team
	for _, team := range t.Teams {
		if team.ID == parent.TeamAPlayer1ID.String() {
			teamA = team
		} else {
			for _, p := range team.Players {
				if p.ID == parent.TeamAPlayer1ID.String() {
					teamA = team
					break
				}
			}
		}
		if team.ID == parent.TeamBPlayer1ID.String() {
			teamB = team
		} else {
			for _, p := range team.Players {
				if p.ID == parent.TeamBPlayer1ID.String() {
					teamB = team
					break
				}
			}
		}
	}

	var subMatches []bun.MatchModel
	_ = uc.matchRepo.DB().NewSelect().Model(&subMatches).Where("team_match_id = ?", parentUUID).Order("round_number ASC").Scan(ctx)

	playerNames := make(map[string]string)
	var playerModels []bun.PlayerModel
	_ = uc.matchRepo.DB().NewSelect().Model(&playerModels).Scan(ctx)
	for _, pm := range playerModels {
		playerNames[pm.ID.String()] = pm.FullName()
	}

	var squadAP1, squadAP2, squadAP3 string
	var squadBP1, squadBP2, squadBP3 string
	for _, sm := range subMatches {
		if teamFormat == "olympic" {
			switch sm.RoundNumber {
			case 3:
				squadAP1 = sm.TeamAPlayer1ID.String()
				squadBP1 = sm.TeamBPlayer1ID.String()
			case 4:
				squadAP2 = sm.TeamAPlayer1ID.String()
				squadBP2 = sm.TeamBPlayer1ID.String()
			case 2:
				squadAP3 = sm.TeamAPlayer1ID.String()
				squadBP3 = sm.TeamBPlayer1ID.String()
			}
		} else {
			switch sm.RoundNumber {
			case 1:
				squadAP1 = sm.TeamAPlayer1ID.String()
				squadBP1 = sm.TeamBPlayer1ID.String()
			case 2:
				squadAP2 = sm.TeamAPlayer1ID.String()
				squadBP2 = sm.TeamBPlayer1ID.String()
			case 3:
				squadAP3 = sm.TeamAPlayer1ID.String()
				squadBP3 = sm.TeamBPlayer1ID.String()
			}
		}
	}

	if squadAP1 != "" && squadAP1 != "00000000-0000-0000-0000-000000000000" && teamB != nil {
		isSwapped := false
		for _, p := range teamB.Players {
			if p.ID == squadAP1 {
				isSwapped = true
				break
			}
		}
		if isSwapped {
			teamA, teamB = teamB, teamA
		}
	}

	if squadAP1 == "00000000-0000-0000-0000-000000000000" && teamA != nil && len(teamA.Players) > 0 {
		squadAP1 = teamA.Players[0].ID
		if len(teamA.Players) > 1 {
			squadAP2 = teamA.Players[1].ID
		}
		if len(teamA.Players) > 2 {
			squadAP3 = teamA.Players[2].ID
		}
	}
	if squadBP1 == "00000000-0000-0000-0000-000000000000" && teamB != nil && len(teamB.Players) > 0 {
		squadBP1 = teamB.Players[0].ID
		if len(teamB.Players) > 1 {
			squadBP2 = teamB.Players[1].ID
		}
		if len(teamB.Players) > 2 {
			squadBP3 = teamB.Players[2].ID
		}
	}

	var subMatchVMs []SubMatchVM
	for _, sm := range subMatches {
		teamAP2Str := ""
		teamBP2Str := ""
		if sm.TeamAPlayer2ID != nil {
			teamAP2Str = sm.TeamAPlayer2ID.String()
		}
		if sm.TeamBPlayer2ID != nil {
			teamBP2Str = sm.TeamBPlayer2ID.String()
		}

		var pAName, pBName string
		if sm.MatchType == "doubles" {
			pAName = playerNames[sm.TeamAPlayer1ID.String()]
			if teamAP2Str != "" {
				pAName += " & " + playerNames[teamAP2Str]
			}
			pBName = playerNames[sm.TeamBPlayer1ID.String()]
			if teamBP2Str != "" {
				pBName += " & " + playerNames[teamBP2Str]
			}
		} else {
			pAName = playerNames[sm.TeamAPlayer1ID.String()]
			pBName = playerNames[sm.TeamBPlayer1ID.String()]
		}

		var setModels []bun.MatchSetModel
		_ = uc.matchRepo.DB().NewSelect().Model(&setModels).Where("match_id = ?", sm.ID).Scan(ctx)

		winsA, winsB := 0, 0
		for _, set := range setModels {
			if set.ScoreA > set.ScoreB {
				winsA++
			} else if set.ScoreB > set.ScoreA {
				winsB++
			}
		}

		wt := ""
		if sm.WinnerTeam != nil {
			wt = *sm.WinnerTeam
		}

		alignA, alignB := getSubMatchAlignments(sm.RoundNumber, teamFormat)

		subMatchVMs = append(subMatchVMs, SubMatchVM{
			ID:             sm.ID.String(),
			MatchType:      sm.MatchType,
			RoundNumber:    sm.RoundNumber,
			TeamAPlayer1ID: sm.TeamAPlayer1ID.String(),
			TeamAPlayer2ID: teamAP2Str,
			TeamBPlayer1ID: sm.TeamBPlayer1ID.String(),
			TeamBPlayer2ID: teamBP2Str,
			PlayerAName:    pAName,
			PlayerBName:    pBName,
			AlignmentA:     alignA,
			AlignmentB:     alignB,
			ScoreA:         winsA,
			ScoreB:         winsB,
			Status:         sm.Status,
			WinnerTeam:     wt,
		})
	}

	var refereeIDStr string
	if parent.RefereeID != nil {
		refereeIDStr = parent.RefereeID.String()
	}

	return &TeamMatchFormView{
		MatchID:      matchID,
		TournamentID: tournamentID,
		Stage:        stage,
		BestOf:       bestOf,
		TeamA:        teamA,
		TeamB:        teamB,
		TeamFormat:   teamFormat,
		SubMatches:   subMatchVMs,
		SquadAP1:     squadAP1,
		SquadAP2:     squadAP2,
		SquadAP3:     squadAP3,
		SquadBP1:     squadBP1,
		SquadBP2:     squadBP2,
		SquadBP3:     squadBP3,
		Participants: t.Participants,
		Pin:          parent.Pin,
		RefereeID:    refereeIDStr,
		TableNumber:  parent.TableNumber,
	}, nil
}

func getSubMatchAlignments(roundNumber int, teamFormat string) (string, string) {
	if teamFormat == "" {
		teamFormat = "olympic"
	}
	if teamFormat == "olympic" {
		switch roundNumber {
		case 1:
			return "A & B", "X & Y"
		case 2:
			return "C", "Z"
		case 3:
			return "A", "X"
		case 4:
			return "B", "Y"
		case 5:
			return "C", "X"
		}
	} else {
		// Corbillon or other format
		switch roundNumber {
		case 1:
			return "A", "X"
		case 2:
			return "B", "Y"
		case 3:
			return "C", "Z"
		case 4:
			return "A", "Y"
		case 5:
			return "B", "X"
		}
	}
	return "", ""
}
