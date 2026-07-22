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
	tournamentRepo *bun.EventRepository
}

func NewGetTeamMatchFormViewUseCase(matchRepo *bun.MatchRepository, tournamentRepo *bun.EventRepository) *GetTeamMatchFormViewUseCase {
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

	subMatches, err := uc.matchRepo.GetSubMatches(ctx, matchID)
	if err != nil {
		return nil, err
	}

	var squadAP1, squadAP2, squadAP3 string
	var squadBP1, squadBP2, squadBP3 string
	for _, sm := range subMatches {
		a1, b1 := teamPlayerID(sm.TeamA, 0), teamPlayerID(sm.TeamB, 0)
		if teamFormat == "olympic" {
			switch sm.RoundNumber {
			case 3:
				squadAP1, squadBP1 = a1, b1
			case 4:
				squadAP2, squadBP2 = a1, b1
			case 2:
				squadAP3, squadBP3 = a1, b1
			}
		} else {
			switch sm.RoundNumber {
			case 1:
				squadAP1, squadBP1 = a1, b1
			case 2:
				squadAP2, squadBP2 = a1, b1
			case 3:
				squadAP3, squadBP3 = a1, b1
			}
		}
	}

	if squadAP1 != "" && teamB != nil {
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

	if squadAP1 == "" && teamA != nil && len(teamA.Players) > 0 {
		squadAP1 = teamA.Players[0].ID
		if len(teamA.Players) > 1 {
			squadAP2 = teamA.Players[1].ID
		}
		if len(teamA.Players) > 2 {
			squadAP3 = teamA.Players[2].ID
		}
	}
	if squadBP1 == "" && teamB != nil && len(teamB.Players) > 0 {
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
		teamAP1Str, teamAP2Str := teamPlayerID(sm.TeamA, 0), teamPlayerID(sm.TeamA, 1)
		teamBP1Str, teamBP2Str := teamPlayerID(sm.TeamB, 0), teamPlayerID(sm.TeamB, 1)

		var pAName, pBName string
		if sm.MatchType == "doubles" {
			pAName = teamPlayerName(sm.TeamA, 0)
			if teamAP2Str != "" {
				pAName += " & " + teamPlayerName(sm.TeamA, 1)
			}
			pBName = teamPlayerName(sm.TeamB, 0)
			if teamBP2Str != "" {
				pBName += " & " + teamPlayerName(sm.TeamB, 1)
			}
		} else {
			pAName = teamPlayerName(sm.TeamA, 0)
			pBName = teamPlayerName(sm.TeamB, 0)
		}

		alignA, alignB := getSubMatchAlignments(sm.RoundNumber, teamFormat)

		subMatchVMs = append(subMatchVMs, SubMatchVM{
			ID:             sm.ID,
			MatchType:      sm.MatchType,
			RoundNumber:    sm.RoundNumber,
			TeamAPlayer1ID: teamAP1Str,
			TeamAPlayer2ID: teamAP2Str,
			TeamBPlayer1ID: teamBP1Str,
			TeamBPlayer2ID: teamBP2Str,
			PlayerAName:    pAName,
			PlayerBName:    pBName,
			AlignmentA:     alignA,
			AlignmentB:     alignB,
			ScoreA:         sm.ScoreA(),
			ScoreB:         sm.ScoreB(),
			Status:         sm.Status,
			WinnerTeam:     sm.WinnerTeam,
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

func teamPlayerID(team []*player.Player, idx int) string {
	if idx >= len(team) {
		return ""
	}
	return team[idx].ID
}

func teamPlayerName(team []*player.Player, idx int) string {
	if idx >= len(team) {
		return ""
	}
	return team[idx].FullName()
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
