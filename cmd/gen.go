package cmd

import (
	"embed"
	"fmt"
	"html/template"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rybkr/sudoku/internal/board"
	"github.com/rybkr/sudoku/internal/generator"
	"github.com/rybkr/sudoku/internal/solver"
)

//go:embed templates/*.html
var templateFS embed.FS

// PuzzlePage holds pre-rendered data for a single puzzle page in the HTML template.
type PuzzlePage struct {
	PuzzleNumber int
	Difficulty   int
	GridHTML     template.HTML
}

// TemplateData holds data for HTML template rendering.
type TemplateData struct {
	TitlePrefix string
	BodyFont    string
	HeadingFont string
	ThemeClass  string
	PuzzlePages []PuzzlePage
}

var (
	numPuzzles int
	clueCount  string
	outputFile string
	theme      string
	boardType  string
	timeout    time.Duration
)

const (
	// difficultyMin is the lowest acceptable solver difficulty score (0–100).
	// Puzzles scoring below this threshold are too easy and are discarded.
	difficultyMin = 67
	// difficultyMax is the highest acceptable solver difficulty score (0–100).
	// Puzzles scoring above this threshold are considered unsolvable by normal
	// human techniques and are discarded.
	difficultyMax = 100
	// difficultyMaxRetries caps how many consecutive out-of-range puzzles the
	// generator will discard before giving up, preventing an infinite loop when
	// the requested clue count cannot yield puzzles in the target difficulty range.
	difficultyMaxRetries = 50
)

func init() {
	genCmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate Sudoku puzzles",
		Long: `Generate one or more Sudoku puzzles with a specified difficulty level.

Examples:
  sudoku gen --clueCount 40
  sudoku gen -n 5 --clueCount 30
  sudoku gen --clueCount 20 --timeout 15s
  sudoku gen --type jigsaw -n 4 -o puzzles.html`,
		RunE: runGen,
	}

	genCmd.Flags().IntVarP(&numPuzzles, "number", "n", 1, "Number of puzzles to generate")
	genCmd.Flags().StringVarP(&clueCount, "clueCount", "c", fmt.Sprintf("%d", generator.DefaultClueCount), "Number of clues 17-80 or range like 28:32")
	genCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (e.g., puzzles.html)")
	genCmd.Flags().StringVarP(&theme, "theme", "t", "", "Theme for HTML output (e.g., princess)")
	genCmd.Flags().StringVar(&boardType, "type", "standard", "Board type: standard or jigsaw")
	genCmd.Flags().DurationVar(&timeout, "timeout", 10*time.Second, "Generation timeout per puzzle")

	rootCmd.AddCommand(genCmd)
}

// parseClueCountRange parses a clue count string which can be:
// - A single number: "32"
// - A range: "28:32"
// Returns min, max, and an error
func parseClueCountRange(s string) (min, max int, err error) {
	parts := strings.Split(s, ":")
	if len(parts) == 1 {
		// Single number
		val, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid clue count: %w", err)
		}
		return val, val, nil
	} else if len(parts) == 2 {
		// Range
		minVal, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid clue count min: %w", err)
		}
		maxVal, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid clue count max: %w", err)
		}
		if minVal > maxVal {
			return 0, 0, fmt.Errorf("clue count min (%d) cannot be greater than max (%d)", minVal, maxVal)
		}
		return minVal, maxVal, nil
	}
	return 0, 0, fmt.Errorf("invalid clue count format: %s (use format like '32' or '28:32')", s)
}

// boardToHTML converts a board to a safe HTML table for embedding in the template.
// For jigsaw layouts, each cell receives directional border classes (border-top,
// border-right, border-bottom, border-left) where the cell abuts a different region,
// and a region-N class for background shading.
// For standard layouts the existing nth-child CSS handles thick 3×3-box borders.
func boardToHTML(b *board.Board) template.HTML {
	layout := b.Layout()
	isJigsaw := layout.Type == "jigsaw"

	var sb strings.Builder
	sb.WriteString(`<div class="sudoku-grid"><table>`)
	for row := range 9 {
		sb.WriteString("<tr>")
		for col := range 9 {
			pos := board.MakePos(row, col)
			val := b.Get(pos)
			region := layout.PosToRegion[pos]

			// Build CSS class list for this cell.
			var classes []string
			if val == board.EmptyCell {
				classes = append(classes, "empty")
			}
			if isJigsaw {
				// Add region shading class.
				classes = append(classes, fmt.Sprintf("region-%d", region))

				// Add a directional border class for each edge where the
				// adjacent cell belongs to a different region (or is outside
				// the grid, which also marks a boundary).
				if row == 0 || layout.PosToRegion[board.MakePos(row-1, col)] != region {
					classes = append(classes, "border-top")
				}
				if row == 8 || layout.PosToRegion[board.MakePos(row+1, col)] != region {
					classes = append(classes, "border-bottom")
				}
				if col == 0 || layout.PosToRegion[board.MakePos(row, col-1)] != region {
					classes = append(classes, "border-left")
				}
				if col == 8 || layout.PosToRegion[board.MakePos(row, col+1)] != region {
					classes = append(classes, "border-right")
				}
			}

			classAttr := ""
			if len(classes) > 0 {
				classAttr = ` class="` + strings.Join(classes, " ") + `"`
			}

			if val == board.EmptyCell {
				fmt.Fprintf(&sb, "<td%s></td>", classAttr)
			} else {
				fmt.Fprintf(&sb, "<td%s>%d</td>", classAttr, val)
			}
		}
		sb.WriteString("</tr>")
	}
	sb.WriteString("</table></div>")
	return template.HTML(sb.String())
}

// generateHTML creates an HTML file with puzzles using templates.
func generateHTML(filename string, puzzles []*board.Board, difficulties []int, theme string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create HTML file: %w", err)
	}
	defer file.Close()

	// Determine theme-specific values.
	var titlePrefix, bodyFont, headingFont, themeClass string
	switch strings.ToLower(theme) {
	case "princess":
		titlePrefix = "Princess Puzzle"
		bodyFont = "'Georgia', 'Times New Roman', serif"
		headingFont = "'Playfair Display', 'Georgia', serif"
		themeClass = "princess-theme"
	default:
		titlePrefix = "Sudoku Puzzle"
		bodyFont = "Arial, sans-serif"
		headingFont = "Arial, sans-serif"
		themeClass = ""
	}

	// Pre-render each puzzle board into HTML so the template stays logic-free.
	pages := make([]PuzzlePage, len(puzzles))
	for i, p := range puzzles {
		pages[i] = PuzzlePage{
			PuzzleNumber: i + 1,
			Difficulty:   difficulties[i],
			GridHTML:     boardToHTML(p),
		}
	}

	tmpl, err := template.ParseFS(templateFS, "templates/puzzles.html")
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	data := TemplateData{
		TitlePrefix: titlePrefix,
		BodyFont:    bodyFont,
		HeadingFont: headingFont,
		ThemeClass:  themeClass,
		PuzzlePages: pages,
	}

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

func runGen(cmd *cobra.Command, args []string) error {
	// Resolve the board layout from the --type flag.
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var layout *board.Layout
	switch boardType {
	case "jigsaw":
		layout = board.RandomJigsawLayout(rng)
	case "standard", "":
		layout = board.StandardLayout()
	default:
		return fmt.Errorf("unknown board type %q: must be standard or jigsaw", boardType)
	}

	// Parse clue count range
	minClues, maxClues, err := parseClueCountRange(clueCount)
	if err != nil {
		return err
	}

	// Validate clue count range
	if minClues < generator.MinValidClueCount || minClues > generator.MaxValidClueCount {
		return fmt.Errorf("clue count min (%d) must be between %d and %d", minClues, generator.MinValidClueCount, generator.MaxValidClueCount)
	}
	if maxClues < generator.MinValidClueCount || maxClues > generator.MaxValidClueCount {
		return fmt.Errorf("clue count max (%d) must be between %d and %d", maxClues, generator.MinValidClueCount, generator.MaxValidClueCount)
	}

	// Prepare for HTML output if output file is specified
	var puzzles []*board.Board
	var difficulties []int
	outputHTML := outputFile != ""

	// Generate puzzles
	for i := 0; i < numPuzzles; i++ {
		// Randomly select clue count from range if it's a range
		selectedClueCount := minClues
		if maxClues > minClues {
			selectedClueCount = minClues + rng.Intn(maxClues-minClues+1)
		}

		opts := generator.DefaultOptions(selectedClueCount)
		opts.Timeout = timeout
		opts.Layout = layout
		gen := generator.New(opts)

		puzzle, solution, err := gen.Generate()
		if err != nil {
			return fmt.Errorf("generation failed: %w", err)
		}

		// Calculate difficulty, retrying generation if the puzzle falls outside
		// the accepted range. The retry cap prevents an infinite loop in cases
		// where the requested clue count cannot produce puzzles in range.
		difficulty := solver.Difficulty(puzzle)
		retries := 0
		for (difficulty < difficultyMin || difficulty > difficultyMax) && retries < difficultyMaxRetries {
			puzzle, solution, err = gen.Generate()
			if err != nil {
				return fmt.Errorf("generation failed during difficulty retry: %w", err)
			}
			difficulty = solver.Difficulty(puzzle)
			retries++
		}
		if difficulty < difficultyMin || difficulty > difficultyMax {
			return fmt.Errorf("could not generate puzzle with difficulty in [%d, %d] after %d attempts", difficultyMin, difficultyMax, difficultyMaxRetries)
		}

		if outputHTML {
			// Store puzzles for HTML output
			puzzles = append(puzzles, puzzle)
			difficulties = append(difficulties, difficulty)
		} else {
			// Print to console
			fmt.Printf("Puzzle #%d (Clues: %d, Difficulty: %d):\n", i+1, selectedClueCount, difficulty)
			fmt.Println(puzzle.Format())
			fmt.Println("\nSolution:")
			fmt.Println(solution.Format())
			fmt.Println()
		}
	}

	// Write HTML file if output was specified
	if outputHTML {
		// Sort puzzles by difficulty (ascending order)
		type puzzleWithDifficulty struct {
			puzzle     *board.Board
			difficulty int
		}
		puzzleList := make([]puzzleWithDifficulty, len(puzzles))
		for i := 0; i < len(puzzles); i++ {
			puzzleList[i] = puzzleWithDifficulty{
				puzzle:     puzzles[i],
				difficulty: difficulties[i],
			}
		}
		sort.Slice(puzzleList, func(i, j int) bool {
			return puzzleList[i].difficulty < puzzleList[j].difficulty
		})

		// Extract sorted puzzles and difficulties
		sortedPuzzles := make([]*board.Board, len(puzzleList))
		sortedDifficulties := make([]int, len(puzzleList))
		for i := range puzzleList {
			sortedPuzzles[i] = puzzleList[i].puzzle
			sortedDifficulties[i] = puzzleList[i].difficulty
		}

		// Expand wildcards in filename if needed
		filename := outputFile
		if strings.Contains(filename, "*") {
			// Replace * with index for multiple files, or use a default name
			// For now, just use the filename as-is, replacing * with puzzles
			filename = strings.ReplaceAll(filename, "*", "puzzles")
		}

		// Ensure .html extension
		if filepath.Ext(filename) != ".html" {
			filename = filename + ".html"
		}

		err := generateHTML(filename, sortedPuzzles, sortedDifficulties, theme)
		if err != nil {
			return fmt.Errorf("failed to write HTML file: %w", err)
		}
		fmt.Printf("Generated %d puzzle(s) in %s\n", numPuzzles, filename)
	}

	return nil
}
