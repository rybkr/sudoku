package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rybkr/sudoku/internal/board"
	"github.com/rybkr/sudoku/internal/generator"
	"github.com/spf13/cobra"
)

var (
	numPuzzles int
	clueCount  string
	outputFile string
	timeout    time.Duration
)

func init() {
	genCmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate Sudoku puzzles",
		Long: `Generate one or more Sudoku puzzles with a specified difficulty level.

Examples:
  sudoku gen --clueCount 40
  sudoku gen -n 5 --clueCount 30
  sudoku gen --clueCount 20 --timeout 15s`,
		RunE: runGen,
	}

	genCmd.Flags().IntVarP(&numPuzzles, "number", "n", 1, "Number of puzzles to generate")
	genCmd.Flags().StringVarP(&clueCount, "clueCount", "c", fmt.Sprintf("%d", generator.DefaultClueCount), "Number of clues 17-80 or range like 28:32")
	genCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (e.g., puzzles.html)")
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

// generateHTML creates an HTML file with puzzles, one per page
func generateHTML(filename string, puzzles []*board.Board, solutions []*board.Board) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create HTML file: %w", err)
	}
	defer file.Close()

	// Write HTML header
	_, err = fmt.Fprintf(file, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Sudoku Puzzles</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .page {
            page-break-after: always;
            background-color: white;
            padding: 40px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .page:last-child {
            page-break-after: auto;
        }
        h1 {
            color: #333;
            margin-bottom: 30px;
            text-align: center;
        }
        h2 {
            color: #666;
            margin-top: 20px;
            margin-bottom: 15px;
            font-size: 1.2em;
        }
        .sudoku-grid {
            display: inline-block;
            border: 3px solid #000;
            margin: 20px auto;
            font-family: 'Courier New', monospace;
            font-size: 24px;
            line-height: 1.5;
        }
        .sudoku-grid table {
            border-collapse: collapse;
            margin: 0 auto;
        }
        .sudoku-grid td {
            width: 40px;
            height: 40px;
            text-align: center;
            vertical-align: middle;
            border: 1px solid #333;
            padding: 0;
        }
        .sudoku-grid td.empty {
            color: #ccc;
        }
        .sudoku-grid tr:nth-child(3n) td {
            border-bottom: 2px solid #000;
        }
        .sudoku-grid td:nth-child(3n) {
            border-right: 2px solid #000;
        }
        @media print {
            body {
                background-color: white;
            }
            .page {
                margin-bottom: 0;
                box-shadow: none;
            }
        }
    </style>
</head>
<body>
`)
	if err != nil {
		return err
	}

	// Write each puzzle on its own page
	for i := 0; i < len(puzzles); i++ {
		_, err = fmt.Fprintf(file, `    <div class="page">
        <h1>Sudoku Puzzle #%d</h1>
        <h2>Puzzle</h2>
        %s
        <h2>Solution</h2>
        %s
    </div>
`, i+1, boardToHTML(puzzles[i]), boardToHTML(solutions[i]))
		if err != nil {
			return err
		}
	}

	// Write HTML footer
	_, err = fmt.Fprintf(file, `</body>
</html>
`)
	return err
}

// boardToHTML converts a board to an HTML table representation
func boardToHTML(b *board.Board) string {
	var sb strings.Builder
	sb.WriteString("<div class=\"sudoku-grid\"><table>")

	for row := 0; row < 9; row++ {
		sb.WriteString("<tr>")
		for col := 0; col < 9; col++ {
			pos := board.MakePos(row, col)
			val := b.Get(pos)
			cellClass := ""
			cellContent := ""

			if val == board.EmptyCell {
				cellClass = "empty"
				cellContent = "·"
			} else {
				cellContent = fmt.Sprintf("%d", val)
			}

			sb.WriteString(fmt.Sprintf("<td class=\"%s\">%s</td>", cellClass, cellContent))
		}
		sb.WriteString("</tr>")
	}

	sb.WriteString("</table></div>")
	return sb.String()
}

func runGen(cmd *cobra.Command, args []string) error {
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
	var solutions []*board.Board
	outputHTML := outputFile != ""

	// Generate puzzles
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < numPuzzles; i++ {
		// Randomly select clue count from range if it's a range
		selectedClueCount := minClues
		if maxClues > minClues {
			selectedClueCount = minClues + rng.Intn(maxClues-minClues+1)
		}

		opts := generator.DefaultOptions(selectedClueCount)
		opts.Timeout = timeout
		gen := generator.New(opts)

		puzzle, solution, err := gen.Generate()
		if err != nil {
			return fmt.Errorf("generation failed: %w", err)
		}

		if outputHTML {
			// Store puzzles for HTML output
			puzzles = append(puzzles, puzzle)
			solutions = append(solutions, solution)
		} else {
			// Print to console
			fmt.Printf("Puzzle #%d (Clues: %d):\n", i+1, selectedClueCount)
			fmt.Println(puzzle.Format())
			fmt.Println("\nSolution:")
			fmt.Println(solution.Format())
			fmt.Println()
		}
	}

	// Write HTML file if output was specified
	if outputHTML {
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

		err := generateHTML(filename, puzzles, solutions)
		if err != nil {
			return fmt.Errorf("failed to write HTML file: %w", err)
		}
		fmt.Printf("Generated %d puzzle(s) in %s\n", numPuzzles, filename)
	}

	return nil
}
