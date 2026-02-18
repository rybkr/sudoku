package board

import "math/rand"

// jigsawPresets is the collection of hand-crafted, validated jigsaw region
// maps.  Each entry is an [81]int where the value at index pos (row*9+col)
// is the region number (0–8) that cell belongs to.
//
// Invariants (verified by NewLayout / Validate at package init):
//   - Exactly 9 regions numbered 0–8.
//   - Each region contains exactly 9 cells.
//   - Each region is orthogonally contiguous.
var jigsawPresets = [...]([CellCount]int){

	// Preset 0 — "Zigzag": wide diagonal stripes that zigzag across the grid.
	//   0 0 0 1 1 1 2 2 2
	//   0 0 1 1 1 2 2 2 2
	//   0 1 1 1 3 3 3 2 2
	//   0 3 3 3 3 3 3 4 4
	//   0 5 5 5 6 6 6 4 4
	//   0 5 5 5 6 6 6 4 4
	//   7 5 5 5 6 6 6 4 4
	//   7 7 7 7 8 8 8 8 4
	//   7 7 7 7 8 8 8 8 8
	{
		// row 0
		0, 0, 0, 1, 1, 1, 2, 2, 2,
		// row 1
		0, 0, 1, 1, 1, 2, 2, 2, 2,
		// row 2
		0, 1, 1, 1, 3, 3, 3, 2, 2,
		// row 3
		0, 3, 3, 3, 3, 3, 3, 4, 4,
		// row 4
		0, 5, 5, 5, 6, 6, 6, 4, 4,
		// row 5
		0, 5, 5, 5, 6, 6, 6, 4, 4,
		// row 6
		7, 5, 5, 5, 6, 6, 6, 4, 4,
		// row 7
		7, 7, 7, 7, 8, 8, 8, 8, 4,
		// row 8
		7, 7, 7, 7, 8, 8, 8, 8, 8,
	},

	// Preset 1 — "Staircase": regions step diagonally, like a descending stair.
	//   0 0 0 1 1 1 2 2 2
	//   0 0 1 1 1 2 2 2 2
	//   0 0 1 1 3 3 3 2 2
	//   0 0 1 3 3 3 4 4 4
	//   5 5 5 3 3 3 4 4 4
	//   5 5 5 6 6 6 6 4 4
	//   5 5 5 6 6 6 6 6 4
	//   7 7 7 7 8 8 8 8 8
	//   7 7 7 7 7 8 8 8 8
	{
		// row 0
		0, 0, 0, 1, 1, 1, 2, 2, 2,
		// row 1
		0, 0, 1, 1, 1, 2, 2, 2, 2,
		// row 2
		0, 0, 1, 1, 3, 3, 3, 2, 2,
		// row 3
		0, 0, 1, 3, 3, 3, 4, 4, 4,
		// row 4
		5, 5, 5, 3, 3, 3, 4, 4, 4,
		// row 5
		5, 5, 5, 6, 6, 6, 6, 4, 4,
		// row 6
		5, 5, 5, 6, 6, 6, 6, 6, 4,
		// row 7
		7, 7, 7, 7, 8, 8, 8, 8, 8,
		// row 8
		7, 7, 7, 7, 7, 8, 8, 8, 8,
	},

	// Preset 2 — "T-shapes": T-shaped regions radiating from the grid edges.
	//   0 0 0 0 1 1 1 1 1
	//   0 3 3 3 1 1 1 1 2
	//   0 3 3 3 4 4 4 4 2
	//   0 3 3 3 4 4 4 4 2
	//   0 5 5 5 4 6 6 6 2
	//   0 5 5 5 6 6 6 6 2
	//   7 5 5 5 6 6 8 8 2
	//   7 7 7 7 8 8 8 8 2
	//   7 7 7 7 8 8 8 2 2
	{
		// row 0
		0, 0, 0, 0, 1, 1, 1, 1, 1,
		// row 1
		0, 3, 3, 3, 1, 1, 1, 1, 2,
		// row 2
		0, 3, 3, 3, 4, 4, 4, 4, 2,
		// row 3
		0, 3, 3, 3, 4, 4, 4, 4, 2,
		// row 4
		0, 5, 5, 5, 4, 6, 6, 6, 2,
		// row 5
		0, 5, 5, 5, 6, 6, 6, 6, 2,
		// row 6
		7, 5, 5, 5, 6, 6, 8, 8, 2,
		// row 7
		7, 7, 7, 7, 8, 8, 8, 8, 2,
		// row 8
		7, 7, 7, 7, 8, 8, 8, 2, 2,
	},

	// Preset 3 — "Brick": regions offset like courses of bricks.
	//   0 0 0 0 0 1 1 1 1
	//   0 0 2 2 2 1 1 1 1
	//   0 0 2 2 2 2 2 2 1
	//   3 3 3 4 4 4 4 5 5
	//   3 3 4 4 4 5 5 5 5
	//   3 3 4 4 5 5 5 6 6
	//   3 3 7 7 7 7 7 6 6
	//   8 8 8 8 8 7 7 6 6
	//   8 8 8 8 7 7 6 6 6
	{
		// row 0
		0, 0, 0, 0, 0, 1, 1, 1, 1,
		// row 1
		0, 0, 2, 2, 2, 1, 1, 1, 1,
		// row 2
		0, 0, 2, 2, 2, 2, 2, 2, 1,
		// row 3
		3, 3, 3, 4, 4, 4, 4, 5, 5,
		// row 4
		3, 3, 4, 4, 4, 5, 5, 5, 5,
		// row 5
		3, 3, 4, 4, 5, 5, 5, 6, 6,
		// row 6
		3, 3, 7, 7, 7, 7, 7, 6, 6,
		// row 7
		8, 8, 8, 8, 8, 7, 7, 6, 6,
		// row 8
		8, 8, 8, 8, 7, 7, 6, 6, 6,
	},

	// Preset 4 — "Pinwheel": regions spiral from the top-left toward bottom-right.
	//   0 0 0 0 1 1 1 1 1
	//   0 0 0 1 1 1 1 2 2
	//   0 0 3 3 3 3 3 2 2
	//   4 4 4 4 4 3 3 2 2
	//   4 4 4 4 5 3 3 2 2
	//   6 6 6 6 5 5 5 5 2
	//   6 6 6 6 7 7 5 5 5
	//   6 7 7 7 7 7 7 7 5
	//   8 8 8 8 8 8 8 8 8
	{
		// row 0
		0, 0, 0, 0, 1, 1, 1, 1, 1,
		// row 1
		0, 0, 0, 1, 1, 1, 1, 2, 2,
		// row 2
		0, 0, 3, 3, 3, 3, 3, 2, 2,
		// row 3
		4, 4, 4, 4, 4, 3, 3, 2, 2,
		// row 4
		4, 4, 4, 4, 5, 3, 3, 2, 2,
		// row 5
		6, 6, 6, 6, 5, 5, 5, 5, 2,
		// row 6
		6, 6, 6, 6, 7, 7, 5, 5, 5,
		// row 7
		6, 7, 7, 7, 7, 7, 7, 7, 5,
		// row 8
		8, 8, 8, 8, 8, 8, 8, 8, 8,
	},
}

func init() {
	// Validate all presets at startup so invalid layouts surface immediately.
	for i, rm := range jigsawPresets {
		if _, err := NewLayout(rm); err != nil {
			panic("jigsaw_presets: preset " + string(rune('0'+i)) + " failed validation: " + err.Error())
		}
	}
}

// RandomJigsawLayout returns a randomly selected jigsaw Layout.
func RandomJigsawLayout(rng *rand.Rand) *Layout {
	idx := rng.Intn(len(jigsawPresets))
	l, err := NewLayout(jigsawPresets[idx])
	if err != nil {
		// Already validated in init; this branch is unreachable in production.
		panic("jigsaw_presets: RandomJigsawLayout: " + err.Error())
	}
	return l
}
