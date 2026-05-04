package formatter

import (
	"code-review-agent/internal/models"
	"encoding/json"
)

func FormatJSON(result *models.AnalysisResult) string {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}
