package event

import (
	"testing"

	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
)

func TestBuildBoardCards(t *testing.T) {
	p1 := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "Anderson", SinglesElo: 1200}
	p2 := &playerDomain.Player{ID: "p2", FirstName: "Bob", LastName: "Brown", SinglesElo: 1100}
	p3 := &playerDomain.Player{ID: "p3", FirstName: "Carl", LastName: "Clark", SinglesElo: 1000}

	t.Run("classifies existing matches by status and derives group/division names", func(t *testing.T) {
		table := 3
		mScheduled := tournamentDomain.Match{ID: "m1", Status: "scheduled", Stage: "group", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}}
		mInProgress := tournamentDomain.Match{ID: "m2", Status: "in_progress", Stage: "group", TeamA: []*playerDomain.Player{p2}, TeamB: []*playerDomain.Player{p3}, TableNumber: &table}
		mFinished := tournamentDomain.Match{ID: "m3", Status: "finished", Stage: "group", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p3}}
		mSub := tournamentDomain.Match{ID: "m4", Status: "scheduled", TeamMatchID: strPtr("parent1")}

		ev := &tournamentDomain.Event{
			ID:            "t1",
			Type:          "singles",
			EventCategory: "open",
			Format:        "elimination",
			Participants:  []*playerDomain.Player{p1, p2, p3},
			Groups: []tournamentDomain.Group{
				{ID: "g1", Name: "Open Bracket - Group A", Players: []*playerDomain.Player{p1, p2, p3}},
			},
			Matches: []tournamentDomain.Match{mScheduled, mInProgress, mFinished, mSub},
		}

		scheduled, inProgress, finished := BuildBoardCards(ev, nil)

		foundReal := false
		for _, c := range scheduled {
			if c.MatchID == "m1" {
				foundReal = true
			}
			if c.MatchID == "m4" {
				t.Errorf("expected sub-match m4 to be excluded from scheduled cards")
			}
		}
		if !foundReal {
			t.Errorf("expected the real scheduled match m1 to be present, got %+v", scheduled)
		}
		if len(inProgress) != 1 {
			t.Fatalf("expected 1 in-progress card, got %d", len(inProgress))
		}
		if len(finished) != 1 {
			t.Fatalf("expected 1 finished card, got %d", len(finished))
		}
		if inProgress[0].GroupName != "Group A" {
			t.Errorf("expected group name 'Group A', got %q", inProgress[0].GroupName)
		}
	})

	t.Run("marks in-match players on virtual scheduled cards and assigns queue position", func(t *testing.T) {
		table := 1
		mInProgress := tournamentDomain.Match{
			ID: "m1", Status: "in_progress", Stage: "group",
			TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}, TableNumber: &table,
		}
		ev := &tournamentDomain.Event{
			ID:            "t1",
			Type:          "singles",
			EventCategory: "open",
			Format:        "round_robin",
			Participants:  []*playerDomain.Player{p1, p2, p3},
			Matches:       []tournamentDomain.Match{mInProgress},
		}

		scheduled, inProgress, _ := BuildBoardCards(ev, nil)
		if len(inProgress) != 1 {
			t.Fatalf("expected 1 in-progress card, got %d", len(inProgress))
		}
		// p3 should have a virtual scheduled match against either p1 or p2 via round robin
		foundInMatchFlag := false
		for _, c := range scheduled {
			if c.P1InMatch || c.P2InMatch {
				foundInMatchFlag = true
			}
			if c.QueuePosition == 0 {
				t.Errorf("expected non-zero queue position for scheduled card %+v", c)
			}
		}
		if !foundInMatchFlag {
			t.Errorf("expected at least one scheduled card involving an in-match player to be flagged")
		}
	})
}

func TestFilterBoardCards(t *testing.T) {
	cards := []BoardCard{
		{PlayerAName: "Alice Anderson", PlayerBName: "Bob Brown", GroupName: "Group A", DivisionName: "Open"},
		{PlayerAName: "Carl Clark", PlayerBName: "Dave Davis", GroupName: "Group B", DivisionName: "First"},
	}

	t.Run("no filters returns all", func(t *testing.T) {
		got := FilterBoardCards(cards, "", nil)
		if len(got) != 2 {
			t.Fatalf("expected 2, got %d", len(got))
		}
	})

	t.Run("filters by search query on player names", func(t *testing.T) {
		got := FilterBoardCards(cards, "alice", nil)
		if len(got) != 1 || got[0].PlayerAName != "Alice Anderson" {
			t.Fatalf("expected only Alice's match, got %+v", got)
		}
	})

	t.Run("filters by search query on group name", func(t *testing.T) {
		got := FilterBoardCards(cards, "group b", nil)
		if len(got) != 1 || got[0].GroupName != "Group B" {
			t.Fatalf("expected only Group B match, got %+v", got)
		}
	})

	t.Run("filters by division", func(t *testing.T) {
		got := FilterBoardCards(cards, "", []string{"First"})
		if len(got) != 1 || got[0].DivisionName != "First" {
			t.Fatalf("expected only First division match, got %+v", got)
		}
	})

	t.Run("combined filters must all match", func(t *testing.T) {
		got := FilterBoardCards(cards, "alice", []string{"First"})
		if len(got) != 0 {
			t.Fatalf("expected no matches, got %+v", got)
		}
	})
}

func TestFilterEventBoardCards(t *testing.T) {
	cards := []BoardCard{
		{PlayerAName: "Alice Anderson", PlayerBName: "Bob Brown", GroupName: "Group A", DivisionName: "Open", Category: "men"},
		{PlayerAName: "Carl Clark", PlayerBName: "Dave Davis", GroupName: "Group B", DivisionName: "First", Category: "women"},
	}

	t.Run("no filters returns all", func(t *testing.T) {
		got := FilterEventBoardCards(cards, "", nil, nil)
		if len(got) != 2 {
			t.Fatalf("expected 2, got %d", len(got))
		}
	})

	t.Run("filters by category", func(t *testing.T) {
		got := FilterEventBoardCards(cards, "", nil, []string{"women"})
		if len(got) != 1 || got[0].Category != "women" {
			t.Fatalf("expected only women's match, got %+v", got)
		}
	})

	t.Run("filters by query, division, and category combined", func(t *testing.T) {
		got := FilterEventBoardCards(cards, "carl", []string{"First"}, []string{"women"})
		if len(got) != 1 {
			t.Fatalf("expected 1 match, got %+v", got)
		}
	})
}

func TestBuildTableVMs(t *testing.T) {
	t.Run("nil event returns nil", func(t *testing.T) {
		if got := BuildTableVMs(nil, "", nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("zero tables returns nil", func(t *testing.T) {
		ev := &tournamentDomain.Event{NumTables: 0}
		if got := BuildTableVMs(ev, "", nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("marks in-progress tables used, excludes given match, includes global occupied", func(t *testing.T) {
		table1 := 1
		table2 := 2
		ev := &tournamentDomain.Event{
			NumTables: 3,
			Matches: []tournamentDomain.Match{
				{ID: "m1", Status: "in_progress", TableNumber: &table1},
				{ID: "m2", Status: "in_progress", TableNumber: &table2},
			},
		}
		got := BuildTableVMs(ev, "m2", []int{3})
		if len(got) != 3 {
			t.Fatalf("expected 3 tables, got %d", len(got))
		}
		if !got[0].IsUsed {
			t.Errorf("expected table 1 to be used")
		}
		if got[1].IsUsed {
			t.Errorf("expected table 2 to be free (excluded match)")
		}
		if !got[2].IsUsed {
			t.Errorf("expected table 3 to be used (globally occupied)")
		}
	})
}

func TestBuildEventTableVMs(t *testing.T) {
	t.Run("zero or negative tables returns nil", func(t *testing.T) {
		if got := buildEventTableVMs(0, nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("marks occupied tables from in-progress cards", func(t *testing.T) {
		table1 := 2
		cards := []BoardCard{{TableNumber: &table1}}
		got := buildEventTableVMs(3, cards)
		if len(got) != 3 {
			t.Fatalf("expected 3 tables, got %d", len(got))
		}
		if !got[1].IsUsed {
			t.Errorf("expected table 2 to be used")
		}
		if got[0].IsUsed || got[2].IsUsed {
			t.Errorf("expected tables 1 and 3 to be free")
		}
	})
}

func TestSortBoardCards(t *testing.T) {
	cards := []BoardCard{
		{PlayerAName: "Zoe"},
		{PlayerAName: "Alice"},
		{PlayerAName: "Mike"},
	}
	SortBoardCards(cards)
	if cards[0].PlayerAName != "Alice" || cards[1].PlayerAName != "Mike" || cards[2].PlayerAName != "Zoe" {
		t.Errorf("expected alphabetical order, got %+v", cards)
	}
}

func strPtr(s string) *string { return &s }
