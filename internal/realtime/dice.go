package realtime

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
)

// DiceResult represents the outcome of a dice roll.
type DiceResult struct {
	Formula  string `json:"formula"`
	Results  []int  `json:"results"`
	Modifier int    `json:"modifier"`
	Total    int    `json:"total"`
}

// diceRegex matches dice formulas: NdS, NdS+M, NdS-M, dS
var diceRegex = regexp.MustCompile(`^(\d*)d(\d+)([+-]\d+)?$`)

// RollDice parses a dice formula and rolls the dice using crypto/rand.
// Supported formats: NdS, NdS+M, NdS-M, dS (shorthand for 1dS).
func RollDice(formula string) (*DiceResult, error) {
	if formula == "" {
		return nil, fmt.Errorf("dice: empty formula")
	}

	matches := diceRegex.FindStringSubmatch(formula)
	if matches == nil {
		return nil, fmt.Errorf("dice: invalid formula %q", formula)
	}

	// Parse count (N). Default to 1 if omitted (e.g. "d6").
	count := 1
	if matches[1] != "" {
		var err error
		count, err = strconv.Atoi(matches[1])
		if err != nil {
			return nil, fmt.Errorf("dice: invalid count in %q", formula)
		}
	}

	// Parse sides (S).
	sides, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, fmt.Errorf("dice: invalid sides in %q", formula)
	}

	// Validate ranges.
	if count < 1 || count > 100 {
		return nil, fmt.Errorf("dice: count must be 1-100, got %d", count)
	}
	if sides < 2 || sides > 1000 {
		return nil, fmt.Errorf("dice: sides must be 2-1000, got %d", sides)
	}

	// Parse modifier (+M or -M).
	modifier := 0
	if matches[3] != "" {
		modifier, err = strconv.Atoi(matches[3])
		if err != nil {
			return nil, fmt.Errorf("dice: invalid modifier in %q", formula)
		}
	}

	// Roll dice using crypto/rand.
	results := make([]int, count)
	total := 0
	for i := 0; i < count; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(sides)))
		if err != nil {
			return nil, fmt.Errorf("dice: random generation failed: %w", err)
		}
		results[i] = int(n.Int64()) + 1 // 1-based
		total += results[i]
	}
	total += modifier

	return &DiceResult{
		Formula:  formula,
		Results:  results,
		Modifier: modifier,
		Total:    total,
	}, nil
}
