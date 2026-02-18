package generator

import (
	"errors"
	"github.com/rybkr/sudoku/internal/board"
	"github.com/rybkr/sudoku/internal/solver"
	"math/rand"
	"time"
)

const (
	MinValidClueCount = 17
	MaxValidClueCount = 80
	DefaultClueCount  = 32
)

var (
	ErrGenerationFailed = errors.New("failed to generate valid puzzle")
	ErrInvalidClueCount = errors.New("clue count must be between 17 and 80")
	ErrDiggingFailed    = errors.New("failed to remove proper number of clues")
)

// Generator creates Sudoku puzzles.
type Generator struct {
	options *Options
	rng     *rand.Rand
}

// New creates a puzzle generator with the given options.
func New(options *Options) *Generator {
	if options == nil {
		options = DefaultOptions(DefaultClueCount)
	}

	seed := options.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	return &Generator{
		options: options,
		rng:     rand.New(rand.NewSource(seed)),
	}
}

// Generate creates a new Sudoku puzzle.
// Returns the puzzle and its solution, or an error if generation fails.
func (g *Generator) Generate() (puzzle *board.Board, solution *board.Board, err error) {
	if g.options.ClueCount < MinValidClueCount || g.options.ClueCount > MaxValidClueCount {
		return nil, nil, ErrInvalidClueCount
	}

	start := time.Now()
	timeout := g.options.Timeout

	for {
		if time.Since(start) >= timeout {
			return nil, nil, ErrGenerationFailed
		}

		// Generate a complete valid board
		solution, err = g.generateSolution()
		if err != nil {
			continue
		}

		// Remove clues to create the puzzle
		puzzle, err = g.removeCells(solution)
		if err != nil {
			continue
		}

		// Verify uniqueness if required
		if g.options.EnsureUnique {
			if !g.hasUniqueSolution(puzzle) {
				continue
			}
		}

		return puzzle, solution, nil
	}
}

// generateSolution creates a complete valid Sudoku board.
func (g *Generator) generateSolution() (*board.Board, error) {
	// Pass the layout so the solver operates with the correct region structure.
	b := board.New(g.options.Layout)

	// Use solver with randomization to generate a complete board
	s := solver.New(b, &solver.Options{
		MaxSolutions: 1,
		Randomize:    true,
		Timeout:      g.options.Timeout,
	})

	return s.Solve()
}

// removeCells removes clues from a complete board to create a puzzle.
func (g *Generator) removeCells(solution *board.Board) (*board.Board, error) {
	puzzle := solution.Clone()

	// Calculate how many cells to remove
	targetClues := g.options.ClueCount
	cellsToRemove := board.CellCount - targetClues

	// Create shuffled list of all positions
	positions := g.rng.Perm(board.CellCount)

	// Remove cells until we reach target clues
	cellsRemoved := 0
	for _, pos := range positions {
		if cellsRemoved >= cellsToRemove {
			break
		}

		// Try removing this cell
		val := puzzle.Get(pos)
		if val == board.EmptyCell {
			continue
		}

		puzzle.Clear(pos)
		cellsRemoved++

		// Verify the puzzle still has a unique solution
		if g.options.EnsureUnique {
			if !g.hasUniqueSolution(puzzle) {
				// Restore the cells
				puzzle.SetForce(pos, val)
				cellsRemoved--
			}
		}
	}

	if cellsRemoved == cellsToRemove {
		return puzzle, nil
	} else {
		return puzzle, ErrDiggingFailed
	}
}

// hasUniqueSolution checks if the puzzle has exactly one solution.
func (g *Generator) hasUniqueSolution(puzzle *board.Board) bool {
	s := solver.New(puzzle, &solver.Options{
		MaxSolutions: 2,
		Randomize:    false,
		Timeout:      g.options.Timeout,
	})

	solutions := g.countSolutions(s)
	return solutions == 1
}

// countSolutions counts the number of solutions for a puzzle.
func (g *Generator) countSolutions(s *solver.Solver) int {
	count := 0

	// Use backtracking to count solutions
	var backtrack func(*board.Board) bool
	backtrack = func(b *board.Board) bool {
		// Apply constraint propagation
		tempSolver := solver.New(b, &solver.Options{
			MaxSolutions: 1,
			Randomize:    false,
		})

		if err := tempSolver.PropagateConstraints(); err != nil {
			return false
		}

		// Check if solved
		if tempSolver.Board.EmptyCount() == 0 {
			count++
			return count < 2 // Stop after finding 2 solutions
		}

		// Find MRV cell
		pos, candidates := tempSolver.FindMRVCell()
		if len(candidates) == 0 {
			return false
		}

		for _, val := range candidates {
			if count >= 2 {
				return false
			}

			clone := tempSolver.Board.Clone()
			clone.SetForce(pos, val)

			backtrack(clone)
		}

		return count < 2
	}

	backtrack(s.Board.Clone())
	return count
}

// GenerateWithClueCount is a convenience function to generate a puzzle with a specific clue count.
func GenerateWithClueCount(clueCount int) (*board.Board, *board.Board, error) {
	gen := New(DefaultOptions(clueCount))
	return gen.Generate()
}
