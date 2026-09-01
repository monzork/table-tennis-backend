package match

import (
	"context"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
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
	EventID        string
	Stage          string
	BestOf         int
	PointsToWin    int
	PointsMargin   int
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
	matchRepo      event.MatchRepository
	tournamentRepo event.Repository
	playerRepo     player.Repository
	createMatchUC  *CreateMatchUseCase
	teamMatchUC    *TeamMatchOrchestratorUseCase
}

func NewGetScoreFormViewUseCase(
	matchRepo event.MatchRepository,
	tournamentRepo event.Repository,
	playerRepo player.Repository,
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
	// Load existing match (pin, referee, table number, status, sets, resolved
	// TeamA/TeamB players) once — every other lookup below reuses it instead
	// of hitting the DB again.
	var existingMatch *event.Match
	matchStatus := "scheduled"
	refereeIDStr := ""
	var matchTableNumber *int
	var matchPin string
	if isValidID(matchID) {
		if em, err := uc.matchRepo.GetByID(ctx, matchID); err == nil {
			existingMatch = em
			matchPin = em.Pin
			if em.RefereeID != nil {
				refereeIDStr = *em.RefereeID
			}
			matchTableNumber = em.TableNumber
			matchStatus = em.Status
		}
	}

	// Fetch tournament (lite — no heavy Matches relation)
	var tourney *event.Event
	if tID != "" {
		if t, err := uc.tournamentRepo.GetByIDLite(ctx, tID); err == nil {
			tourney = t
		}
	}

	pointsToWin, pointsMargin := 11, 2
	if tourney != nil {
		stageRule := tourney.GetEffectiveStageRule(stage)
		pointsToWin = stageRule.PointsToWin
		pointsMargin = stageRule.PointsMargin
	}

	// Detect teams mode
	var isTeams, isSubMatch bool
	if tourney != nil && tourney.Type == "teams" {
		if existingMatch != nil && existingMatch.TeamMatchID != nil {
			isSubMatch = true
		}
		if !isSubMatch {
			// Signal to handler to use team-match form
			isTeams = true

			// Ensure parent match & sub-matches exist. p1Id/p2Id are team IDs here;
			// CreateMatchUseCase resolves team-based matches by team ID via its own
			// teamPlayersMap, so it must receive the team IDs directly, not their players' IDs.
			if !isValidID(matchID) && tourney != nil {
				var teamAExists, teamBExists bool
				for _, team := range tourney.Teams {
					if team.ID == p1Id {
						teamAExists = true
					}
					if team.ID == p2Id {
						teamBExists = true
					}
				}
				if teamAExists && teamBExists {
					if newMatch, err := uc.createMatchUC.Execute(ctx, tID, "teams", []string{p1Id}, []string{p2Id}, stage); err == nil && newMatch != nil {
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
				IsTeams: true,
				MatchID: matchID,
				EventID: tID,
				Stage:   stage,
			}, nil
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
			if len(existingMatch.TeamA) > 0 {
				p1A = existingMatch.TeamA[0]
			}
			if len(existingMatch.TeamA) > 1 {
				p2A = existingMatch.TeamA[1]
			}
			if len(existingMatch.TeamB) > 0 {
				p1B = existingMatch.TeamB[0]
			}
			if len(existingMatch.TeamB) > 1 {
				p2B = existingMatch.TeamB[1]
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
		} else if existingMatch != nil && len(existingMatch.TeamA) > 0 {
			playerAName = existingMatch.TeamA[0].FirstNameWithSecond() + " " + existingMatch.TeamA[0].LastNameWithSecond()
		}
		if p2Id != "" {
			if p, err := uc.playerRepo.GetById(ctx, p2Id); err == nil {
				playerBName = p.FirstNameWithSecond() + " " + p.LastNameWithSecond()
			}
		} else if existingMatch != nil && len(existingMatch.TeamB) > 0 {
			playerBName = existingMatch.TeamB[0].FirstNameWithSecond() + " " + existingMatch.TeamB[0].LastNameWithSecond()
		}
	}

	// Load existing set scores
	existingScores := make(map[int]event.MatchSet)
	if existingMatch != nil {
		for _, sm := range existingMatch.Sets {
			existingScores[sm.Number] = sm
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
		EventID:        tID,
		Stage:          stage,
		BestOf:         bestOf,
		PointsToWin:    pointsToWin,
		PointsMargin:   pointsMargin,
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
