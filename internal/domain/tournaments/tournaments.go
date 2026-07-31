package tournaments

const PlayerEnrolledEventName = "PlayerEnrolledEvent"

type PlayerEnrolledEvent struct {
	EventID  string
	PlayerID string
}

func (e PlayerEnrolledEvent) EventName() string {
	return PlayerEnrolledEventName
}
