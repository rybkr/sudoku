package board

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidPosition = errors.New("position out of bounds")
	ErrInvalidValue    = errors.New("value must be between 1-9")
	ErrIllegalMove     = errors.New("move violates Sudoku constraints")
)

// IsValid reports whether a board satisfies Sudoku constraints.
// Empty cells are ignored for validation.
func (b *Board) IsValid() bool {
	var rowCheck, colCheck, regionCheck [9]uint

	for pos := range CellCount {
		val := b.Get(pos)
		if val == EmptyCell {
			continue
		}

		row, col, region := posToRow[pos], posToCol[pos], b.layout.PosToRegion[pos]
		mask := uint(1 << (val - 1))

		// Check for duplicates in row, column, or region
		if rowCheck[row]&mask != 0 ||
			colCheck[col]&mask != 0 ||
			regionCheck[region]&mask != 0 {
			return false
		}

		rowCheck[row] |= mask
		colCheck[col] |= mask
		regionCheck[region] |= mask
	}

	return true
}

// isValidPosition reports whether a given position is in bounds of a Sudoku board.
func isValidPosition(pos int) bool {
	return pos >= 0 && pos < CellCount
}

// validatePosition checks if a position is within board bounds.
func (b *Board) validatePosition(pos int) error {
	if !isValidPosition(pos) {
		return fmt.Errorf("%w: position %d must be in range [0, %d)", ErrInvalidPosition, pos, CellCount)
	}
	return nil
}

// isValidValue reports whether a given number is valid on a Sudoku board.
func isValidValue(num int) bool {
	return (num >= 1 && num <= 9) || num == EmptyCell
}

// validateValue checks if a value is valid for Sudoku (1-9).
func (b *Board) validateValue(val int) error {
	if !isValidValue(val) {
		return fmt.Errorf("%w: got %d", ErrInvalidValue, val)
	}
	return nil
}
