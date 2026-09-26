package service

import (
	"encoding/json"
	"testing"
)

func TestCurrentEpochFromPayload(t *testing.T) {
	payload := map[string]any{
		"status": "OK",
		"data": map[string]any{
			"current_epoch": json.Number("42"),
			"epochs":        []any{},
		},
	}
	got, err := currentEpochFromPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	if got != 42 {
		t.Fatalf("epoch = %d, want 42", got)
	}

	// The old reader looked at the top-level "epoch" field, which Dora does not send.
	if _, err := currentEpochFromPayload(map[string]any{"epoch": json.Number("7")}); err == nil {
		t.Fatal("expected missing data.current_epoch to fail")
	}
}

func TestComparableEpochs(t *testing.T) {
	counts := map[int]int{0: 32, 1: 32, 2: 10, 3: 32}
	got, mismatches := ComparableEpochs(counts, 32, 200)
	if got != 2 {
		t.Fatalf("comparable = %d, want 2", got)
	}
	if len(mismatches) != 1 || mismatches[0] != 2 {
		t.Fatalf("mismatches = %v", mismatches)
	}

	full := map[int]int{}
	for epoch := 0; epoch < 4; epoch++ {
		full[epoch] = 32
	}
	got, mismatches = ComparableEpochs(full, 32, 4)
	if got != 4 || mismatches != nil {
		t.Fatalf("complete prefix = %d mismatches %v", got, mismatches)
	}
}

func TestAggregateUsesEarliestStakeAndComparableWindow(t *testing.T) {
	slots := []Proposal{
		{Epoch: 0, Slot: 0, Validator: 1},
		{Epoch: 0, Slot: 1, Validator: 1},
		{Epoch: 0, Slot: 2, Validator: 6},
		{Epoch: 5, Slot: 160, Validator: 6},
	}
	validators := []ValidatorStake{
		{Index: 1, Epoch: 3, EffectiveBalance: "64000000000"},
		{Index: 1, Epoch: 0, EffectiveBalance: "36000000000"},
		{Index: 6, Epoch: 0, EffectiveBalance: "36000000000"},
		{Index: 4, Epoch: 0, EffectiveBalance: "", Balance: "32000000000"},
	}

	rows := Aggregate("desw", slots, validators, 1)
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(rows))
	}
	if rows[0].ValidatorIndex != 1 || rows[0].Stake != "36000000000" || rows[0].ProposerCount != 2 {
		t.Fatalf("validator 1 row = %+v", rows[0])
	}
	if rows[1].ValidatorIndex != 4 || rows[1].Stake != "32000000000" || rows[1].ProposerCount != 0 {
		t.Fatalf("validator 4 row = %+v", rows[1])
	}
	if rows[2].ValidatorIndex != 6 || rows[2].ProposerCount != 1 {
		t.Fatalf("validator 6 row = %+v", rows[2])
	}
}
