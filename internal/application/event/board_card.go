package event

type BoardCard struct {
	MatchID        string
	Status         string
	Stage          string
	BestOf         int
	PlayerAName    string
	PlayerBName    string
	P1Id           string
	P2Id           string
	TableNumber    *int
	ScoreA         int
	ScoreB         int
	Pin            string
	GroupName      string
	DivisionName   string
	P1InMatch      bool
	P2InMatch      bool
	TournamentID   string
	TournamentName string
	QueuePosition  int
	RoundNumber    int
	Category       string
}
