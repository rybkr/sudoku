package solver

import (
	"context"
	"errors"
	"math/bits"
	"math/rand"
	"time"

	"github.com/rybkr/sudoku/internal/board"
)

var (
	ErrNoSolution        = errors.New("puzzle has no solution")
	ErrMultipleSolutions = errors.New("puzzle has multiple solutions")
	ErrInvalidPuzzle     = errors.New("puzzle violates Sudoku constraints")
	ErrTimeout           = errors.New("solver timeout exceeded")
)

// Solver implements algorithms for solving Sudoku puzzles.
type Solver struct {
	Board   *board.Board
	options *Options
	rng     *rand.Rand
}

// New creates a solver for the given board.
func New(b *board.Board, options *Options) *Solver {
	if options == nil {
		options = DefaultOptions()
	}

	s := &Solver{
		Board:   b.Clone(),
		options: options,
	}

	if options.Randomize {
		s.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	return s
}

// Solve attempts to solve the puzzle.
// Returns the solved board or an error if unsolvable.
func (s *Solver) Solve() (*board.Board, error) {
	if !s.Board.IsValid() {
		return nil, ErrInvalidPuzzle
	}

	// fillThreeBoxes seeds 3 diagonal 3×3 boxes simultaneously — valid only for
	// standard layouts where those boxes share no row, column, or region constraints.
	if s.Board.EmptyCount() == board.CellCount && s.Board.Layout().Type == "standard" {
		s.fillThreeBoxes()
	}

	// Constraint propagation is faster, try this first
	if err := s.PropagateConstraints(); err != nil {
		return nil, err
	}
	if s.Board.EmptyCount() == 0 {
		return s.Board, nil
	}

	// Start backtracking with MRV heuristic
	// MRV = Minimum Remaining Values, guess on the most constrained cells first
	// to reduce total search space
	ctx, cancel := s.makeContext()
	defer cancel()

	if !s.backtrack(ctx) {
		return nil, ErrNoSolution
	} else {
		return s.Board, nil
	}
}

// PropagateConstraints applies constraint propagation techniques.
func (s *Solver) PropagateConstraints() error {
	changed := true
	iterations := 0
	maxIterations := board.CellCount * board.CellCount

	for changed && iterations < maxIterations {
		changed = false
		iterations++

		if s.applyNakedSingles() {
			changed = true
		}
		if s.applyHiddenSingles() {
			changed = true
		}

		if s.hasContradiction() {
			return ErrNoSolution
		}
	}

	return nil
}

// applyNakedSingles fills cells with only one candidate.
func (s *Solver) applyNakedSingles() bool {
	changed := false

	for pos := range board.CellCount {
		if s.Board.Get(pos) == board.EmptyCell {
			mask := s.Board.GetCandidatesMask(pos)

			if mask == 0 {
				break // Will be caught by contradiction check
			}

			// Check if only one bit is set
			if bits.OnesCount(mask) == 1 {
				val := bits.TrailingZeros(mask) + 1
				s.Board.SetForce(pos, val)
				changed = true
			}
		}
	}

	return changed
}

// applyHiddenSingles finds values that can only go in one place within a unit.
func (s *Solver) applyHiddenSingles() bool {
	changed := false

	for row := range 9 {
		changed = s.findHiddenSinglesInRow(row) || changed
	}
	for col := range 9 {
		changed = s.findHiddenSinglesInCol(col) || changed
	}
	for region := range 9 {
		changed = s.findHiddenSinglesInRegion(region) || changed
	}

	return changed
}

// findHiddenSinglesInRow checks for hidden singles in the provided row.
func (s *Solver) findHiddenSinglesInRow(row int) bool {
	changed := false

	// Track where each value can go
	valuePossibilities := make([][]int, 10)

	for col := range 9 {
		if s.Board.Get(board.MakePos(row, col)) == board.EmptyCell {
			candidates := s.Board.GetCandidates(board.MakePos(row, col))
			for _, val := range candidates {
				valuePossibilities[val] = append(valuePossibilities[val], row*9+col)
			}
		}
	}

	// Find values with only one possible position
	for val := 1; val <= 9; val++ {
		if len(valuePossibilities[val]) == 1 {
			pos := valuePossibilities[val][0]
			s.Board.SetForce(pos, val)
			changed = true
		}
	}

	return changed
}

// findHiddenSinglesInCol checks for hidden singles in the provided col.
func (s *Solver) findHiddenSinglesInCol(col int) bool {
	changed := false

	// Track where each value can go
	valuePossibilities := make([][]int, 10)

	for row := range 9 {
		if s.Board.Get(board.MakePos(row, col)) == board.EmptyCell {
			candidates := s.Board.GetCandidates(board.MakePos(row, col))
			for _, val := range candidates {
				valuePossibilities[val] = append(valuePossibilities[val], row*9+col)
			}
		}
	}

	// Find values with only one possible position
	for val := 1; val <= 9; val++ {
		if len(valuePossibilities[val]) == 1 {
			pos := valuePossibilities[val][0]
			s.Board.SetForce(pos, val)
			changed = true
		}
	}

	return changed
}

// findHiddenSinglesInRegion checks for hidden singles in the provided region.
// For standard layouts, regions are 3×3 boxes; for jigsaw layouts, regions are
// arbitrary contiguous shapes. Cell enumeration uses RegionCells so the algorithm
// is identical for both variants.
func (s *Solver) findHiddenSinglesInRegion(region int) bool {
	changed := false
	valuePossibilities := make([][]int, 10)

	for _, pos := range s.Board.RegionCells(region) {
		if s.Board.Get(pos) == board.EmptyCell {
			candidates := s.Board.GetCandidates(pos)
			for _, val := range candidates {
				valuePossibilities[val] = append(valuePossibilities[val], pos)
			}
		}
	}

	for val := 1; val <= 9; val++ {
		if len(valuePossibilities[val]) == 1 {
			pos := valuePossibilities[val][0]
			s.Board.SetForce(pos, val)
			changed = true
		}
	}

	return changed
}

// hasContradiction checks if the board has reached an invalid state.
func (s *Solver) hasContradiction() bool {
	for pos := range board.CellCount {
		if s.Board.Get(pos) == board.EmptyCell && s.Board.GetCandidatesMask(pos) == 0 {
			return true
		}
	}
	return false
}

// backtrack implements recursive backtracking with MRV heuristic.
func (s *Solver) backtrack(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	default:
	}

	// Apply constraint propagation at each level
	if err := s.PropagateConstraints(); err != nil {
		return false
	}

	// Check if we've already solved it
	if s.Board.EmptyCount() == 0 {
		return true
	}

	// Find the cell with the minimum remaining values
	pos, candidates := s.FindMRVCell()
	if len(candidates) == 0 {
		return false
	}

	// Randomize candidates if needed
	if s.options.Randomize && s.rng != nil {
		s.rng.Shuffle(len(candidates), func(i, j int) {
			candidates[i], candidates[j] = candidates[j], candidates[i]
		})
	}

	for _, val := range candidates {
		s.Board.SetForce(pos, val)
		if s.backtrack(ctx) {
			return true
		}
		s.Board.Clear(pos)
	}

	return false
}

// FindMRVCell finds the empty cell with fewest candidates.
func (s *Solver) FindMRVCell() (int, []int) {
	mrvPos := -1
	mrvCount := 10
	var mrvCandidates []int

	for pos := range board.CellCount {
		if s.Board.Get(pos) == board.EmptyCell {
			candidates := s.Board.GetCandidates(pos)
			count := len(candidates)

			if count < mrvCount {
				mrvCount = count
				mrvPos = pos
				mrvCandidates = candidates

				if count <= 1 {
					break
				}
			}
		}
	}

	return mrvPos, mrvCandidates
}

// fillThreeBoxes fills three 3x3 boxes (27 cells total) that are all independent.
func (s *Solver) fillThreeBoxes() {
	boxColumns := []int{0, 3, 6}
	if s.options.Randomize && s.rng != nil {
		s.rng.Shuffle(len(boxColumns), func(i, j int) {
			boxColumns[i], boxColumns[j] = boxColumns[j], boxColumns[i]
		})
	}
	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}

	for i, boxRow := range []int{0, 3, 6} {
		boxCol := boxColumns[i]
		if s.options.Randomize && s.rng != nil {
			s.rng.Shuffle(len(nums), func(i, j int) {
				nums[i], nums[j] = nums[j], nums[i]
			})
		}
		for j, val := range nums {
			dr, dc := int(j/3), j%3
			pos := (boxRow+dr)*9 + boxCol + dc
			s.Board.SetForce(pos, val)
		}
	}
}
