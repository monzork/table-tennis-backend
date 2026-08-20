package event

import "time"

type BoardCard struct {
	MatchID            string
	Status             string
	Stage              string
	BestOf             int
	PlayerAName        string
	PlayerBName        string
	P1Id               string
	P2Id               string
	TableNumber        *int
	ScoreA             int
	ScoreB             int
	Pin                string
	GroupName          string
	DivisionName       string
	P1InMatch          bool
	P2InMatch          bool
	EventID            string
	TournamentName     string
	QueuePosition      int
	RoundNumber        int
	Category           string
	EstimatedStartTime *time.Time

	// HasProposal is true when a player has submitted a score proposal for
	// this match that's still awaiting confirmation (by the opponent or an
	// admin) -- see MatchRepository.ProposeScore / MatchHandler.AdminConfirmProposal.
	HasProposal bool

	// ProjectedEloWinA/LossA and ProjectedEloWinB/LossB preview the Elo
	// points player A/B would gain/lose from this specific match if they win
	// or lose, computed from both sides' current Elo. Nil when either side
	// isn't a resolved player yet.
	ProjectedEloWinA  *float64
	ProjectedEloLossA *float64
	ProjectedEloWinB  *float64
	ProjectedEloLossB *float64

	// EloDeltaA/EloDeltaB are the *actual* Elo points team A/B gained or
	// lost from this match, once Elo has been applied (event finished or
	// recalculated) -- unlike ProjectedEloWin/Loss above, these are a real
	// result, not a preview. While the owning event hasn't been finished/
	// recalculated yet, these fall back to a preview computed from current
	// Elo and the match's real winner -- see EloDeltaIsPreview.
	EloDeltaA         *float64
	EloDeltaB         *float64
	EloDeltaIsPreview bool
}
