package tournaments

const PlayerEnrolledEventName = "PlayerEnrolledEvent"

type PlayerEnrolledEvent struct {
	TournamentID string
	PlayerID     string
}

func (e PlayerEnrolledEvent) EventName() string {
	return PlayerEnrolledEventName
}
