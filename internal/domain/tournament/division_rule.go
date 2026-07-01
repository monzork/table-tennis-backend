package tournament

// DivisionRule defines match rules for a specific division within a tournament.
// This allows different divisions to have different match formats (e.g., Best of 3 vs Best of 5).
type DivisionRule struct {
	ID           string
	TournamentID string
	DivisionID   string
	BestOf       int // e.g. 3, 5, or 7
	PointsToWin  int // e.g. 11
	PointsMargin int // must win by this many (e.g. 2)
}

// NewDivisionRule creates a new DivisionRule with validation.
func NewDivisionRule(tournamentID, divisionID string, bestOf, pointsToWin, pointsMargin int) (*DivisionRule, error) {
	if tournamentID == "" {
		return nil, ErrInvalidTournamentID
	}
	if divisionID == "" {
		return nil, ErrInvalidDivisionID
	}
	if bestOf < 3 || bestOf%2 == 0 {
		return nil, ErrInvalidBestOf
	}
	if pointsToWin < 1 {
		return nil, ErrInvalidPointsToWin
	}
	if pointsMargin < 1 {
		return nil, ErrInvalidPointsMargin
	}

	return &DivisionRule{
		ID:           generateDivisionRuleID(tournamentID, divisionID),
		TournamentID: tournamentID,
		DivisionID:   divisionID,
		BestOf:       bestOf,
		PointsToWin:  pointsToWin,
		PointsMargin: pointsMargin,
	}, nil
}

// generateDivisionRuleID creates a deterministic ID from tournament and division IDs.
func generateDivisionRuleID(tournamentID, divisionID string) string {
	return tournamentID + "-" + divisionID
}

// ToStageRule converts a DivisionRule to a StageRule for compatibility with existing match logic.
func (dr *DivisionRule) ToStageRule() StageRule {
	return StageRule{
		ID:           dr.ID,
		TournamentID: dr.TournamentID,
		Stage:        "division_override",
		BestOf:       dr.BestOf,
		PointsToWin:  dr.PointsToWin,
		PointsMargin: dr.PointsMargin,
	}
}

// Domain validation errors
var (
	ErrInvalidTournamentID = errorf("tournament ID is required")
	ErrInvalidDivisionID   = errorf("division ID is required")
	ErrInvalidBestOf       = errorf("best_of must be an odd number >= 3")
	ErrInvalidPointsToWin  = errorf("points_to_win must be >= 1")
	ErrInvalidPointsMargin = errorf("points_margin must be >= 1")
)

// Simple error constructor
func errorf(msg string) error {
	return &validationError{msg: msg}
}

type validationError struct {
	msg string
}

func (e *validationError) Error() string {
	return e.msg
}
