package bracket

// Test-only exports for unexported helpers so external tests (package
// bracket_test) can unit-test their branches directly. getMatchWinner in
// particular is only ever invoked internally on losers-bracket rounds whose
// Match field is attached after the fact, so its "resolved" branches are
// otherwise unreachable through the public BuildBracket entry point.

func GetMatchWinnerForTest(m BracketMatch) *MatchSlot {
	return getMatchWinner(m)
}

func GetMatchLoserForTest(m BracketMatch) *MatchSlot {
	return getMatchLoser(m)
}
