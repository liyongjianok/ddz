package game

import "testing"

func TestCanBeatSameType(t *testing.T) {
	testCases := []struct {
		name      string
		candidate []string
		previous  []string
		want      bool
	}{
		{name: "single_higher", candidate: []string{"S4"}, previous: []string{"S3"}, want: true},
		{name: "single_lower", candidate: []string{"S3"}, previous: []string{"S4"}, want: false},
		{name: "pair_higher", candidate: []string{"S6", "H6"}, previous: []string{"S5", "H5"}, want: true},
		{name: "pair_lower", candidate: []string{"S5", "H5"}, previous: []string{"S6", "H6"}, want: false},
		{name: "triple_with_pair_higher", candidate: []string{"S4", "H4", "D4", "S6", "H6"}, previous: []string{"S3", "H3", "D3", "S5", "H5"}, want: true},
		{name: "straight_same_length_higher", candidate: []string{"S4", "H5", "D6", "C7", "S8"}, previous: []string{"S3", "H4", "D5", "C6", "S7"}, want: true},
		{name: "straight_different_length", candidate: []string{"S4", "H5", "D6", "C7", "S8", "H9"}, previous: []string{"S3", "H4", "D5", "C6", "S7"}, want: false},
		{name: "pair_straight_same_length_higher", candidate: []string{"S4", "H4", "S5", "H5", "S6", "H6"}, previous: []string{"S3", "H3", "S4", "H4", "S5", "H5"}, want: true},
		{name: "airplane_same_length_higher", candidate: []string{"S4", "H4", "D4", "S5", "H5", "D5"}, previous: []string{"S3", "H3", "D3", "S4", "H4", "D4"}, want: true},
		{name: "airplane_with_singles_same_length_higher", candidate: []string{"S4", "H4", "D4", "S5", "H5", "D5", "S7", "H8"}, previous: []string{"S3", "H3", "D3", "S4", "H4", "D4", "S7", "H8"}, want: true},
		{name: "airplane_with_pairs_same_length_higher", candidate: []string{"S4", "H4", "D4", "S5", "H5", "D5", "S7", "H7", "S8", "H8"}, previous: []string{"S3", "H3", "D3", "S4", "H4", "D4", "S7", "H7", "S8", "H8"}, want: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			candidate := mustRecognize(t, tc.candidate)
			previous := mustRecognize(t, tc.previous)
			if got := CanBeat(candidate, previous); got != tc.want {
				t.Fatalf("CanBeat() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCanBeatSpecialCases(t *testing.T) {
	testCases := []struct {
		name      string
		candidate []string
		previous  []string
		want      bool
	}{
		{name: "bomb_beats_non_bomb", candidate: []string{"S3", "H3", "D3", "C3"}, previous: []string{"SA"}, want: true},
		{name: "higher_bomb_beats_lower_bomb", candidate: []string{"S8", "H8", "D8", "C8"}, previous: []string{"S7", "H7", "D7", "C7"}, want: true},
		{name: "lower_bomb_loses_to_higher_bomb", candidate: []string{"S7", "H7", "D7", "C7"}, previous: []string{"S8", "H8", "D8", "C8"}, want: false},
		{name: "rocket_beats_bomb", candidate: []string{"BJ", "RJ"}, previous: []string{"S8", "H8", "D8", "C8"}, want: true},
		{name: "nothing_beats_rocket", candidate: []string{"S9", "H9", "D9", "C9"}, previous: []string{"BJ", "RJ"}, want: false},
		{name: "different_non_bomb_types_cannot_compare", candidate: []string{"S4", "H4"}, previous: []string{"S3"}, want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			candidate := mustRecognize(t, tc.candidate)
			previous := mustRecognize(t, tc.previous)
			if got := CanBeat(candidate, previous); got != tc.want {
				t.Fatalf("CanBeat() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCanBeatRejectsZeroGroups(t *testing.T) {
	if CanBeat(CardGroup{}, CardGroup{}) {
		t.Fatal("zero groups should not compare")
	}
}

func mustRecognize(t *testing.T, codes []string) CardGroup {
	t.Helper()
	group, err := Recognize(mustCards(t, codes))
	if err != nil {
		t.Fatalf("Recognize(%v) error = %v", codes, err)
	}
	return group
}
