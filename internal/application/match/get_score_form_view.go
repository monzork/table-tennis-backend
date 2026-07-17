package match

import (
	"context"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

// SetVM is a single set view model for the score entry form.
type SetVM struct {
	Number int
	ScoreA interface{}
	ScoreB interface{}
}

// ScoreFormView holds all data needed to render the score entry form template (both admin and public).
// When IsTeams is true, the caller should redirect to the team match form instead.
type ScoreFormView struct {
	MatchID        string
	TournamentID   string
	Stage          string
	BestOf         int
	PlayerA        string
	PlayerB        string
	Sets           []SetVM
	P1Id           string
	P2Id           string
	IsSubMatch     bool
	IsDoubles      bool
	IsTeams        bool // sentinel: caller should use renderTeamMatchFormInternal
	PlayerANames   string
	PlayerBNames   string
	Pin            string
	RefereeID      string
	TableNumber    *int
	TableNumberVal int
	Status         string
	Participants   []*player.Player
}

// GetScoreFormViewUseCase orchestrates all data fetching needed to render the score entry form.
type GetScoreFormViewUseCase struct {
	matchRepo      *bun.MatchRepository
	tournamentRepo *bun.EventRepository
	playerRepo     *bun.PlayerRepository
	createMatchUC  *CreateMatchUseCase
	teamMatchUC    *TeamMatchOrchestratorUseCase
}

func NewGetScoreFormViewUseCase(
	matchRepo *bun.MatchRepository,
	tournamentRepo *bun.EventRepository,
	playerRepo *bun.PlayerRepository,
	createMatchUC *CreateMatchUseCase,
	teamMatchUC *TeamMatchOrchestratorUseCase,
) *GetScoreFormViewUseCase {
	return &GetScoreFormViewUseCase{
		matchRepo:      matchRepo,
		tournamentRepo: tournamentRepo,
		playerRepo:     playerRepo,
		createMatchUC:  createMatchUC,
		teamMatchUC:    teamMatchUC,
	}
}

// Execute gathers all data for the score form. If IsTeams is true in the result,
// the handler should delegate to renderTeamMatchFormInternal.
func (uc *GetScoreFormViewUseCase) Execute(
	ctx context.Context,
	matchID, tID, stage string,
	bestOf int,
	p1Id, p2Id string,
) (*ScoreFormView, error) {
	// Load match metadata (pin, referee, table number, status)
	var matchPin string
	var matchRefereeID *uuid.UUID
	var matchTableNumber *int
	matchStatus := "scheduled"
	if isValidID(matchID) {
		mUUID, _ := uuid.Parse(matchID)
		if mModel, err := uc.matchRepo.GetModelByID(ctx, mUUID); err == nil {
			matchPin = mModel.Pin
			matchRefereeID = mModel.RefereeID
			matchTableNumber = mModel.TableNumber
			matchStatus = mModel.Status
		}
	}
	refereeIDStr := ""
	if matchRefereeID != nil {
		refereeIDStr = matchRefereeID.String()
	}

	// Fetch tournament (lite — no heavy Matches relation)
	var tourney *event.Event
	if tID != "" {
		if t, err := uc.tournamentRepo.GetByIDLite(ctx, tID); err == nil {
			tourney = t
		}
	}

	// Detect teams mode
	var isTeams, isSubMatch bool
	if tourney != nil && tourney.Type == "teams" {
		if isValidID(matchID) {
			mUUID, _ := uuid.Parse(matchID)
			if existingMatch, err := uc.matchRepo.GetModelByID(ctx, mUUID); err == nil && existingMatch.TeamMatchID != nil {
				isSubMatch = true
			}
		}
		if !isSubMatch {
			// Signal to handler to use team-match form
			isTeams = true

			// Ensure parent match & sub-matches exist
			if !isValidID(matchID) && tourney != nil {
				var teamAIDs, teamBIDs []string
				for _, team := range tourney.Teams {
					if team.ID == p1Id {
						for _, p := range team.Players {
							teamAIDs = append(teamAIDs, p.ID)
						}
					}
					if team.ID == p2Id {
						for _, p := range team.Players {
							teamBIDs = append(teamBIDs, p.ID)
						}
					}
				}
				if len(teamAIDs) > 0 && len(teamBIDs) > 0 {
					if newMatch, err := uc.createMatchUC.Execute(ctx, tID, "teams", teamAIDs, teamBIDs, stage); err == nil && newMatch != nil {
						matchID = newMatch.ID
					}
				}
			}
			if isValidID(matchID) {
				var teamA, teamB *event.Team
				for _, team := range tourney.Teams {
					if team.ID == p1Id {
						teamA = team
					}
					if team.ID == p2Id {
						teamB = team
					}
				}
				teamFormat := tourney.TeamFormat
				if teamFormat == "" {
					teamFormat = "olympic"
				}
				_ = uc.teamMatchUC.EnsureTeamSubMatches(ctx, matchID, teamA, teamB, teamFormat, stage)
			}

			return &ScoreFormView{
				IsTeams:      true,
				MatchID:      matchID,
				TournamentID: tID,
				Stage:        stage,
			}, nil
		}
	}

	// Fetch existing match model
	var existingMatch *bun.MatchModel
	if isValidID(matchID) {
		mUUID, _ := uuid.Parse(matchID)
		if em, err := uc.matchRepo.GetModelByID(ctx, mUUID); err == nil {
			existingMatch = em
		}
	}

	// Determine doubles
	isDoubles := false
	if tourney != nil && (tourney.Type == "doubles" || tourney.Type == "mixed_doubles") {
		isDoubles = true
	} else if existingMatch != nil && existingMatch.MatchType == "doubles" {
		isDoubles = true
	}

	playerAName := "Player 1"
	playerBName := "Player 2"
	var playerANames, playerBNames string

	if isDoubles {
		var p1A, p2A, p1B, p2B *player.Player

		if existingMatch != nil {
			p1UUID := existingMatch.TeamAPlayer1ID
			p1BUUID := existingMatch.TeamBPlayer1ID
			if p, err := uc.playerRepo.GetById(ctx, p1UUID.String()); err == nil {
				p1A = p
			}
			if existingMatch.TeamAPlayer2ID != nil {
				if p, err := uc.playerRepo.GetById(ctx, existingMatch.TeamAPlayer2ID.String()); err == nil {
					p2A = p
				}
			}
			if p, err := uc.playerRepo.GetById(ctx, p1BUUID.String()); err == nil {
				p1B = p
			}
			if existingMatch.TeamBPlayer2ID != nil {
				if p, err := uc.playerRepo.GetById(ctx, existingMatch.TeamBPlayer2ID.String()); err == nil {
					p2B = p
				}
			}
		} else if tourney != nil {
			for _, team := range tourney.Teams {
				if team.ID == p1Id {
					if len(team.Players) > 0 {
						p1A = team.Players[0]
					}
					if len(team.Players) > 1 {
						p2A = team.Players[1]
					}
				}
				if team.ID == p2Id {
					if len(team.Players) > 0 {
						p1B = team.Players[0]
					}
					if len(team.Players) > 1 {
						p2B = team.Players[1]
					}
				}
			}
		}

		// Look up team names
		var teamAName, teamBName string
		if tourney != nil {
			if p1A != nil {
				for _, team := range tourney.Teams {
					for _, tp := range team.Players {
						if tp.ID == p1A.ID {
							teamAName = team.Name
							break
						}
					}
					if teamAName != "" {
						break
					}
				}
			}
			if p1B != nil {
				for _, team := range tourney.Teams {
					for _, tp := range team.Players {
						if tp.ID == p1B.ID {
							teamBName = team.Name
							break
						}
					}
					if teamBName != "" {
						break
					}
				}
			}
		}

		if p1A != nil {
			playerANames = p1A.FirstNameWithSecond() + " " + p1A.LastNameWithSecond()
			if p2A != nil {
				playerANames += " & " + p2A.FirstNameWithSecond() + " " + p2A.LastNameWithSecond()
			}
		}
		if p1B != nil {
			playerBNames = p1B.FirstNameWithSecond() + " " + p1B.LastNameWithSecond()
			if p2B != nil {
				playerBNames += " & " + p2B.FirstNameWithSecond() + " " + p2B.LastNameWithSecond()
			}
		}
		if teamAName != "" {
			playerAName = teamAName
		} else if playerANames != "" {
			playerAName = playerANames
		}
		if teamBName != "" {
			playerBName = teamBName
		} else if playerBNames != "" {
			playerBName = playerBNames
		}
	} else {
		// Singles
		if p1Id != "" {
			if p, err := uc.playerRepo.GetById(ctx, p1Id); err == nil {
				playerAName = p.FirstNameWithSecond() + " " + p.LastNameWithSecond()
			}
		} else if existingMatch != nil {
			if p, err := uc.playerRepo.GetById(ctx, existingMatch.TeamAPlayer1ID.String()); err == nil {
				playerAName = p.FirstNameWithSecond() + " " + p.LastNameWithSecond()
			}
		}
		if p2Id != "" {
			if p, err := uc.playerRepo.GetById(ctx, p2Id); err == nil {
				playerBName = p.FirstNameWithSecond() + " " + p.LastNameWithSecond()
			}
		} else if existingMatch != nil {
			if p, err := uc.playerRepo.GetById(ctx, existingMatch.TeamBPlayer1ID.String()); err == nil {
				playerBName = p.FirstNameWithSecond() + " " + p.LastNameWithSecond()
			}
		}
	}

	// Load existing set scores
	existingScores := make(map[int]bun.MatchSetModel)
	if isValidID(matchID) {
		if s, err := uc.matchRepo.GetSets(ctx, matchID); err == nil {
			for _, sm := range s {
				existingScores[sm.SetNumber] = sm
			}
		}
	}

	var sets []SetVM
	for i := 1; i <= bestOf; i++ {
		valA, valB := interface{}(""), interface{}("")
		if sm, ok := existingScores[i]; ok {
			valA = sm.ScoreA
			valB = sm.ScoreB
		}
		sets = append(sets, SetVM{Number: i, ScoreA: valA, ScoreB: valB})
	}

	var participants []*player.Player
	if tourney != nil {
		participants = tourney.Participants
	}

	tableNumberVal := 0
	if matchTableNumber != nil {
		tableNumberVal = *matchTableNumber
	}

	return &ScoreFormView{
		MatchID:        matchID,
		TournamentID:   tID,
		Stage:          stage,
		BestOf:         bestOf,
		PlayerA:        playerAName,
		PlayerB:        playerBName,
		Sets:           sets,
		P1Id:           p1Id,
		P2Id:           p2Id,
		IsSubMatch:     isSubMatch,
		IsDoubles:      isDoubles,
		IsTeams:        isTeams,
		PlayerANames:   playerANames,
		PlayerBNames:   playerBNames,
		Pin:            matchPin,
		RefereeID:      refereeIDStr,
		TableNumber:    matchTableNumber,
		TableNumberVal: tableNumberVal,
		Status:         matchStatus,
		Participants:   participants,
	}, nil
}

func isValidID(id string) bool {
	return id != "" && id != "nil" && id != "null" && id != "undefined"
}
