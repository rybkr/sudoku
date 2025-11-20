package solver

import (
	"github.com/rybkr/sudoku/internal/board"
)

// Difficulty returns an integer measure of a board's difficulty.
func Difficulty(b *board.Board) int {
	s := New(b, nil)
	return s.traceDifficulty()
}

// traceDifficulty implements the measure of a board's difficulty.
func (s *Solver) traceDifficulty() int {
	if s.Board.EmptyCount() == 0 {
		return 0
	}

	cell, candidates := s.FindMRVCell()
	if len(candidates) == 0 {
		// If no tiles can be placed, nothing more can be done.
		return 0
	}

	score := 0
	for _, candidate := range candidates {
		s.Board.SetForce(cell, candidate)
		score += 1 + s.traceDifficulty()
		s.Board.Clear(cell)
	}
	return score
}
