// Package jigsaw implements randomized region-map generation for jigsaw Sudoku
// boards.  It has zero external dependencies and no knowledge of the board or
// layout packages so that it can be imported without creating an import cycle.
package jigsaw

import "math/rand"

const (
	gridSize   = 9   // cells per row / column
	regionSize = 9   // cells per region
	totalCells = 81  // gridSize * gridSize
	maxRetries = 200 // upper bound on swap-balancing restarts before panicking
)

// GenerateRegionMap produces a valid jigsaw region map using a two-phase
// approach: uncapped Voronoi assignment followed by boundary-swap balancing.
//
// The returned [81]int assigns each cell position (row*9 + col) to a region
// index in [0, 8].  Every region is guaranteed to be exactly 9 cells and
// orthogonally contiguous.
//
// The function panics only if the internal retry budget is exhausted, which
// should not happen in practice.
func GenerateRegionMap(rng *rand.Rand) [totalCells]int {
	for range maxRetries {
		result, ok := tryGenerate(rng)
		if ok {
			return result
		}
	}
	panic("jigsaw: GenerateRegionMap exceeded max retries — this should never happen")
}

// tryGenerate runs one generation attempt.
//
// Phase 1 — Voronoi partition (no size cap):
//
//	Place one seed per 3×3 macro-box.  Run a BFS/Dijkstra from all seeds
//	simultaneously.  Each cell is assigned to the first region whose wavefront
//	reaches it (i.e. the nearest seed, with random tie-breaking).  Because
//	there is no size cap, every cell is reachable and no region is ever
//	isolated.  Regions are guaranteed contiguous by the BFS construction.
//
// Phase 2 — boundary-swap balancing:
//
//	Repeatedly find an over-sized region (> 9 cells) that shares a border
//	with an under-sized region (< 9 cells), pick a boundary cell of the
//	over-sized region that is adjacent to the under-sized region, and
//	transfer it — but only if the transfer preserves contiguity of both
//	regions.  Repeat until all regions have exactly 9 cells.
//
//	If no valid transfer exists after scanning all boundaries, return false
//	(rare; caused by a degenerate seed layout that cannot be balanced).
func tryGenerate(rng *rand.Rand) ([totalCells]int, bool) {
	// --- Phase 1: Uncapped Voronoi partition ---

	var assigned [totalCells]int
	for i := range assigned {
		assigned[i] = -1
	}

	// BFS queue; each entry is (pos, region).
	type qentry struct{ pos, region int }
	queue := make([]qentry, 0, totalCells*4)

	seeds := chooseSeedCells(rng)
	for r, pos := range seeds {
		assigned[pos] = r
		queue = append(queue, qentry{pos, r})
	}

	// Standard multi-source BFS: process entries in FIFO order.
	// Randomize the order of neighbor exploration to vary shapes.
	//
	// To introduce genuine randomness (not just BFS level-order), we shuffle
	// each level of the BFS queue.  We track level boundaries with two indices.
	head := 0
	for head < len(queue) {
		// Shuffle the current BFS frontier level for randomness.
		// A "level" ends when we have processed all entries queued before the
		// start of this level's expansion.  We use a levelEnd marker.
		levelEnd := len(queue)
		// Shuffle entries [head, levelEnd) before processing.
		for i := levelEnd - 1; i > head; i-- {
			j := head + rng.Intn(i-head+1)
			queue[i], queue[j] = queue[j], queue[i]
		}

		for head < levelEnd {
			e := queue[head]
			head++
			// Expand to unassigned neighbors.
			for _, nb := range orthogonalNeighbors(e.pos) {
				if assigned[nb] == -1 {
					assigned[nb] = e.region
					queue = append(queue, qentry{nb, e.region})
				}
			}
		}
	}

	// Compute initial region sizes.
	var regionSizes [gridSize]int
	for _, r := range assigned {
		regionSizes[r]++
	}

	// --- Phase 2: Boundary-swap balancing ---
	// Transfer border cells from over-sized regions to under-sized neighbors
	// until all regions have exactly regionSize cells.
	//
	// We iterate over random permutations of border cells to avoid systematic
	// bias in the resulting shapes.
	return balanceRegions(assigned, regionSizes, rng)
}

// balanceRegions adjusts assigned (modified in place) so that every region
// ends up with exactly regionSize cells.  It returns the balanced map and true
// on success, or a zero map and false if no valid swap sequence exists.
func balanceRegions(
	assigned [totalCells]int,
	regionSizes [gridSize]int,
	rng *rand.Rand,
) ([totalCells]int, bool) {
	// Maximum swap iterations: generous upper bound to detect stuck states.
	const maxIter = totalCells * 10

	for range maxIter {
		// Check if we're done.
		done := true
		for _, s := range regionSizes {
			if s != regionSize {
				done = false
				break
			}
		}
		if done {
			return assigned, true
		}

		// Find all boundary cells: cells whose region is over-sized and that
		// have at least one orthogonal neighbor belonging to an under-sized
		// region.  Shuffle to randomize the swap order each iteration.
		type candidate struct{ pos, fromRegion, toRegion int }
		candidates := make([]candidate, 0, totalCells)
		for pos := range totalCells {
			r := assigned[pos]
			if regionSizes[r] <= regionSize {
				continue // region is not over-sized
			}
			for _, nb := range orthogonalNeighbors(pos) {
				nr := assigned[nb]
				if nr != r && regionSizes[nr] < regionSize {
					candidates = append(candidates, candidate{pos, r, nr})
				}
			}
		}

		if len(candidates) == 0 {
			// No swap is possible — stuck.
			return [totalCells]int{}, false
		}

		// Shuffle candidates.
		for i := len(candidates) - 1; i > 0; i-- {
			j := rng.Intn(i + 1)
			candidates[i], candidates[j] = candidates[j], candidates[i]
		}

		// Try each candidate until one is valid (preserves contiguity).
		swapped := false
		for _, c := range candidates {
			if isContiguousAfterRemoval(assigned, c.pos, c.fromRegion) {
				// Transfer the cell.
				assigned[c.pos] = c.toRegion
				regionSizes[c.fromRegion]--
				regionSizes[c.toRegion]++
				swapped = true
				break
			}
		}

		if !swapped {
			// All candidates would break contiguity — stuck.
			return [totalCells]int{}, false
		}
	}

	// Ran out of iterations without converging.
	return [totalCells]int{}, false
}

// isContiguousAfterRemoval reports whether the cells of region r that remain
// after removing pos are still orthogonally contiguous.
// Returns true immediately if region r has only 1 cell left after removal
// (trivially contiguous) — but that case never occurs during balancing
// because we only remove from over-sized regions (size ≥ 10).
func isContiguousAfterRemoval(assigned [totalCells]int, pos, r int) bool {
	// Collect all cells of region r except pos.
	var cells [totalCells]int
	n := 0
	var start int
	foundStart := false
	for p := range totalCells {
		if assigned[p] == r && p != pos {
			cells[n] = p
			n++
			if !foundStart {
				start = p
				foundStart = true
			}
		}
	}
	if n == 0 {
		return true // empty region is trivially connected
	}

	// BFS from start; count reachable cells within the region minus pos.
	var inRegion [totalCells]bool
	for i := range n {
		inRegion[cells[i]] = true
	}

	var visited [totalCells]bool
	var bfsQueue [totalCells]int
	head, tail := 0, 0
	bfsQueue[tail] = start
	tail++
	visited[start] = true
	count := 1

	for head < tail {
		p := bfsQueue[head]
		head++
		for _, nb := range orthogonalNeighbors(p) {
			if inRegion[nb] && !visited[nb] {
				visited[nb] = true
				bfsQueue[tail] = nb
				tail++
				count++
			}
		}
	}
	return count == n
}

// chooseSeedCells returns 9 seed positions spread across the board by placing
// one seed inside each of the nine 3×3 macro-boxes at a random position.
func chooseSeedCells(rng *rand.Rand) [gridSize]int {
	var seeds [gridSize]int
	seedIdx := 0
	for boxRow := range 3 {
		for boxCol := range 3 {
			cells := shuffledBoxCells(rng, boxRow, boxCol)
			seeds[seedIdx] = cells[0]
			seedIdx++
		}
	}
	return seeds
}

// shuffledBoxCells returns the 9 cell positions inside the 3×3 macro-box
// identified by (boxRow, boxCol) ∈ [0,2]² in a uniformly random order.
func shuffledBoxCells(rng *rand.Rand, boxRow, boxCol int) [regionSize]int {
	var cells [regionSize]int
	startRow := boxRow * 3
	startCol := boxCol * 3
	i := 0
	for r := startRow; r < startRow+3; r++ {
		for c := startCol; c < startCol+3; c++ {
			cells[i] = r*gridSize + c
			i++
		}
	}
	// Fisher-Yates shuffle.
	for j := regionSize - 1; j > 0; j-- {
		k := rng.Intn(j + 1)
		cells[j], cells[k] = cells[k], cells[j]
	}
	return cells
}

// orthogonalNeighbors returns the in-bounds orthogonal neighbors of pos.
// A stack-local [4]int backs the result slice to avoid heap allocation.
func orthogonalNeighbors(pos int) []int {
	row, col := pos/gridSize, pos%gridSize
	var buf [4]int
	n := 0
	if row > 0 {
		buf[n] = (row-1)*gridSize + col
		n++
	}
	if row < gridSize-1 {
		buf[n] = (row+1)*gridSize + col
		n++
	}
	if col > 0 {
		buf[n] = row*gridSize + col - 1
		n++
	}
	if col < gridSize-1 {
		buf[n] = row*gridSize + col + 1
		n++
	}
	return buf[:n]
}

