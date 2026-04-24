package parser

import (
	"code-review-agent/internal/models"
	"regexp"
	"strconv"
	"strings"
)

// hunkHeader matches lines like: @@ -10,7 +10,7 @@
// Capture group 1 = start line in new file, group 2 = line count (optional)
var hunkHeader = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

// ParseDiff parses a unified diff and returns one DiffHunk per @@ block.
func ParseDiff(content string) []models.DiffHunk {
	var hunks []models.DiffHunk
	var current *models.DiffHunk
	currentFile := ""

	for _, line := range strings.Split(content, "\n") {
		switch {

		// "+++ b/path/to/file" → nom du fichier modifié
		case strings.HasPrefix(line, "+++ b/"):
			currentFile = strings.TrimPrefix(line, "+++ b/")

		case strings.HasPrefix(line, "+++ "):
			currentFile = strings.TrimPrefix(line, "+++ ")

		// "@@ -a,b +c,d @@" → début d'un nouveau bloc
		case hunkHeader.MatchString(line):
			if current != nil {
				hunks = append(hunks, *current)
			}
			m := hunkHeader.FindStringSubmatch(line)
			start, _ := strconv.Atoi(m[1])
			count := 1
			if m[2] != "" {
				count, _ = strconv.Atoi(m[2])
			}
			current = &models.DiffHunk{
				File:      currentFile,
				StartLine: start,
				EndLine:   start + count - 1,
			}

		// ligne supprimée (ignore "--- a/file")
		case current != nil && strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			current.RemovedLines = append(current.RemovedLines, line[1:])

		// ligne ajoutée (ignore "+++ b/file")
		case current != nil && strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			current.AddedLines = append(current.AddedLines, line[1:])

		// ligne de contexte (commence par un espace)
		case current != nil && len(line) > 0 && line[0] == ' ':
			current.Context += line[1:] + "\n"
		}
	}

	// sauvegarde le dernier bloc
	if current != nil {
		hunks = append(hunks, *current)
	}

	return hunks
}
