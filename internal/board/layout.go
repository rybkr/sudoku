package board

import "fmt"

// Layout describes the region structure of a Sudoku board.
// In standard Sudoku all regions are 3×3 boxes; jigsaw layouts use
// irregular, contiguous 9-cell regions of any shape.
//
// Layout is immutable after construction — it is safe to share the same
// pointer across Board clones.
type Layout struct {
	// Type is a human-readable identifier: "standard" or "jigsaw".
	Type string

	// PosToRegion maps a cell position (0–80) to its region index (0–8).
	PosToRegion [CellCount]int

	// RegionToCells is the inverse: given a region index the slice contains
	// the 9 cell positions that belong to it, in ascending order.
	RegionToCells [9][9]int
}

// StandardLayout returns the Layout for a classic 3×3-box Sudoku.
// The mapping formula is identical to the original posToBox init.
func StandardLayout() *Layout {
	var rm [CellCount]int
	for pos := range CellCount {
		rm[pos] = 3*int(pos/27) + int((pos%9)/3)
	}
	l, err := NewLayout(rm)
	if err != nil {
		// Standard layout is hard-coded and always valid; panic on bugs.
		panic("standard layout failed validation: " + err.Error())
	}
	l.Type = "standard"
	return l
}

// NewLayout builds a Layout from an arbitrary region map and validates it.
// regionMap[pos] must be in [0, 8] for every pos.
// Returns an error if validation fails.
func NewLayout(regionMap [CellCount]int) (*Layout, error) {
	l := &Layout{
		Type:        "jigsaw",
		PosToRegion: regionMap,
	}
	if err := l.buildRegionToCells(); err != nil {
		return nil, err
	}
	if err := l.Validate(); err != nil {
		return nil, err
	}
	return l, nil
}

// buildRegionToCells fills the RegionToCells inverse table and checks that
// each region receives exactly 9 cells. It is called before Validate so that
// Validate can rely on RegionToCells being populated.
func (l *Layout) buildRegionToCells() error {
	// counts[r] tracks how many cells have been assigned to region r.
	var counts [9]int

	for pos := range CellCount {
		r := l.PosToRegion[pos]
		if r < 0 || r > 8 {
			return fmt.Errorf("layout: cell %d has out-of-range region %d (must be 0–8)", pos, r)
		}
		if counts[r] >= 9 {
			return fmt.Errorf("layout: region %d has more than 9 cells", r)
		}
		l.RegionToCells[r][counts[r]] = pos
		counts[r]++
	}

	for r := range 9 {
		if counts[r] != 9 {
			return fmt.Errorf("layout: region %d has %d cells, expected 9", r, counts[r])
		}
	}
	return nil
}

// Validate checks that all 9 regions are valid: correct size and contiguous
// (orthogonally connected). buildRegionToCells must have been called first.
func (l *Layout) Validate() error {
	for r := range 9 {
		if err := l.validateContiguous(r); err != nil {
			return err
		}
	}
	return nil
}

// validateContiguous performs a BFS/flood-fill to verify that all 9 cells of
// region r are reachable from each other via orthogonal adjacency.
func (l *Layout) validateContiguous(region int) error {
	cells := l.RegionToCells[region]

	// Build a set of positions in this region for O(1) membership test.
	inRegion := [CellCount]bool{}
	for _, pos := range cells {
		inRegion[pos] = true
	}

	// BFS from the first cell.
	visited := [CellCount]bool{}
	queue := [CellCount]int{}
	head, tail := 0, 0

	queue[tail] = cells[0]
	tail++
	visited[cells[0]] = true
	visitedCount := 1

	for head < tail {
		pos := queue[head]
		head++

		row, col := pos/9, pos%9

		// Explore all 4 orthogonal neighbors.
		neighbors := [4]int{
			(row-1)*9 + col, // up
			(row+1)*9 + col, // down
			row*9 + col - 1, // left
			row*9 + col + 1, // right
		}
		valid := [4]bool{
			row > 0,   // up boundary
			row < 8,   // down boundary
			col > 0,   // left boundary
			col < 8,   // right boundary
		}

		for i, nb := range neighbors {
			if !valid[i] {
				continue
			}
			if inRegion[nb] && !visited[nb] {
				visited[nb] = true
				visitedCount++
				queue[tail] = nb
				tail++
			}
		}
	}

	if visitedCount != 9 {
		return fmt.Errorf("layout: region %d is not contiguous (%d of 9 cells reachable from cell %d)",
			region, visitedCount, cells[0])
	}
	return nil
}
