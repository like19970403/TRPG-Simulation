package realtime

import (
	"testing"
)

func TestRollDice_2d6(t *testing.T) {
	result, err := RollDice("2d6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Formula != "2d6" {
		t.Errorf("Formula = %q, want %q", result.Formula, "2d6")
	}
	if len(result.Results) != 2 {
		t.Fatalf("len(Results) = %d, want 2", len(result.Results))
	}
	for i, v := range result.Results {
		if v < 1 || v > 6 {
			t.Errorf("Results[%d] = %d, want 1-6", i, v)
		}
	}
	if result.Modifier != 0 {
		t.Errorf("Modifier = %d, want 0", result.Modifier)
	}
	expectedTotal := result.Results[0] + result.Results[1]
	if result.Total != expectedTotal {
		t.Errorf("Total = %d, want %d", result.Total, expectedTotal)
	}
}

func TestRollDice_1d20Plus3(t *testing.T) {
	result, err := RollDice("1d20+3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(result.Results))
	}
	if result.Results[0] < 1 || result.Results[0] > 20 {
		t.Errorf("Results[0] = %d, want 1-20", result.Results[0])
	}
	if result.Modifier != 3 {
		t.Errorf("Modifier = %d, want 3", result.Modifier)
	}
	if result.Total != result.Results[0]+3 {
		t.Errorf("Total = %d, want %d", result.Total, result.Results[0]+3)
	}
}

func TestRollDice_3d8Minus2(t *testing.T) {
	result, err := RollDice("3d8-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 3 {
		t.Fatalf("len(Results) = %d, want 3", len(result.Results))
	}
	if result.Modifier != -2 {
		t.Errorf("Modifier = %d, want -2", result.Modifier)
	}
	sum := 0
	for _, v := range result.Results {
		if v < 1 || v > 8 {
			t.Errorf("die result = %d, want 1-8", v)
		}
		sum += v
	}
	if result.Total != sum-2 {
		t.Errorf("Total = %d, want %d", result.Total, sum-2)
	}
}

func TestRollDice_Shortform_d6(t *testing.T) {
	result, err := RollDice("d6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(result.Results))
	}
	if result.Results[0] < 1 || result.Results[0] > 6 {
		t.Errorf("Results[0] = %d, want 1-6", result.Results[0])
	}
}

func TestRollDice_EmptyFormula(t *testing.T) {
	_, err := RollDice("")
	if err == nil {
		t.Fatal("expected error for empty formula")
	}
}

func TestRollDice_InvalidLetters(t *testing.T) {
	_, err := RollDice("abc")
	if err == nil {
		t.Fatal("expected error for invalid formula")
	}
}

func TestRollDice_ZeroDice(t *testing.T) {
	_, err := RollDice("0d6")
	if err == nil {
		t.Fatal("expected error for 0 dice")
	}
}

func TestRollDice_OneSidedDie(t *testing.T) {
	_, err := RollDice("1d1")
	if err == nil {
		t.Fatal("expected error for 1-sided die")
	}
}

func TestRollDice_TooManyDice(t *testing.T) {
	_, err := RollDice("101d6")
	if err == nil {
		t.Fatal("expected error for >100 dice")
	}
}

func TestRollDice_ModifierZero(t *testing.T) {
	result, err := RollDice("1d6+0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Modifier != 0 {
		t.Errorf("Modifier = %d, want 0", result.Modifier)
	}
	if result.Total != result.Results[0] {
		t.Errorf("Total = %d, want %d", result.Total, result.Results[0])
	}
}

func TestRollDice_ResultStructure(t *testing.T) {
	result, err := RollDice("4d10+5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Formula != "4d10+5" {
		t.Errorf("Formula = %q, want %q", result.Formula, "4d10+5")
	}
	if len(result.Results) != 4 {
		t.Fatalf("len(Results) = %d, want 4", len(result.Results))
	}
	if result.Modifier != 5 {
		t.Errorf("Modifier = %d, want 5", result.Modifier)
	}
	sum := 0
	for _, v := range result.Results {
		sum += v
	}
	if result.Total != sum+5 {
		t.Errorf("Total = %d, want %d", result.Total, sum+5)
	}
}

func TestRollDice_Distribution(t *testing.T) {
	// Roll 1d6 many times and check all results are in [1,6].
	for i := 0; i < 100; i++ {
		result, err := RollDice("1d6")
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		if result.Results[0] < 1 || result.Results[0] > 6 {
			t.Fatalf("iteration %d: result %d out of range [1,6]", i, result.Results[0])
		}
	}
}
