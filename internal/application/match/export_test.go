package match

// Test-only exports for unexported helpers, so match_usecase_test.go (package
// match_test) can cover them directly without duplicating their logic.

func IsValidIDForTest(id string) bool {
	return isValidID(id)
}

func GetSubMatchAlignmentsForTest(roundNumber int, teamFormat string) (string, string) {
	return getSubMatchAlignments(roundNumber, teamFormat)
}
