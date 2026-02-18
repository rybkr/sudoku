package board

import (
	"math/rand"

	"github.com/rybkr/sudoku/internal/jigsaw"
)

// RandomJigsawLayout generates a random valid jigsaw layout using randomized
// region growing.  Each call produces a unique irregular region map guaranteed
// to satisfy Layout constraints (9 regions, 9 cells each, all contiguous).
func RandomJigsawLayout(rng *rand.Rand) *Layout {
	regionMap := jigsaw.GenerateRegionMap(rng)
	l, err := NewLayout(regionMap)
	if err != nil {
		// GenerateRegionMap always produces valid maps; reaching here indicates
		// a bug in the generator rather than an expected runtime condition.
		panic("jigsaw region generation produced invalid layout: " + err.Error())
	}
	return l
}
