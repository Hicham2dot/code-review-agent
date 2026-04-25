package parser

import (
	"os"
	"testing"
)

func readFixture(t *testing.T, path string) string {

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read fixture: %v", err)
	}
	return string(content)
}

func TestParseDiff_SecurityIssue(t *testing.T) {
	content := readFixture(t, "../../tests/fixtures/security_issue.diff")
	hunks := ParseDiff(content)

	// doit produire exactement 1 hunk
	if len(hunks) != 1 {
		t.Fatalf("attendu 1 hunk, obtenu %d", len(hunks))
	}

	h := hunks[0]

	// fichier correct
	if h.File != "auth.go" {
		t.Errorf("File : attendu 'auth.go', obtenu '%s'", h.File)
	}

	// numéro de ligne de début (le @@ indique +10)
	if h.StartLine != 10 {
		t.Errorf("StartLine : attendu 10, obtenu %d", h.StartLine)
	}

	// doit avoir exactement 1 ligne supprimée (le mot de passe hardcodé)
	if len(h.RemovedLines) != 1 {
		t.Errorf("RemovedLines : attendu 1, obtenu %d", len(h.RemovedLines))
	}

	// doit avoir exactement 1 ligne ajoutée (la variable d'env)
	if len(h.AddedLines) != 1 {
		t.Errorf("AddedLines : attendu 1, obtenu %d", len(h.AddedLines))
	}

	// vérifie le contenu de la ligne dangereuse supprimée
	if len(h.RemovedLines) > 0 {
		want := `    password := "admin123"`
		if h.RemovedLines[0] != want {
			t.Errorf("RemovedLines[0] :\n  attendu : %q\n  obtenu  : %q", want, h.RemovedLines[0])
		}
	}
}
