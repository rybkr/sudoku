package generator

import (
	"time"

	"github.com/rybkr/sudoku/internal/board"
)

// Options configures puzzle generation behavior.
type Options struct {
	ClueCount    int           // Number of clues to add to the puzzle
	Timeout      time.Duration // Timeout limits generation time
	Seed         int64         // Seed for reproducible puzzles (0 = random)
	EnsureUnique bool          // EnsureUnique verifies single solution
	// Layout specifies the board region structure. nil means StandardLayout.
	Layout *board.Layout
}

// DefaultOptions returns standard generator options.
func DefaultOptions(clueCount int) *Options {
	clueCount = min(max(clueCount, MinValidClueCount), MaxValidClueCount)
	return &Options{
		ClueCount:    clueCount,
		Timeout:      10 * time.Second,
		Seed:         0,
		EnsureUnique: true,
		Layout:       nil, // nil → StandardLayout inside board.New
	}
}
