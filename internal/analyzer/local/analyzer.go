package local

import (
	"code-review-agent/internal/models"
	"sync"
)

// LocalAnalyze performs static code analysis on code hunks without AI
func LocalAnalyze(hunks []models.DiffHunk) []models.Issue {
	registry := NewRuleRegistry()
	rules := registry.GetRules()

	// Channel to collect issues from all goroutines
	issueChan := make(chan []models.Issue, len(rules))
	var wg sync.WaitGroup

	// Launch each rule in a separate goroutine
	for _, rule := range rules {
		wg.Add(1)
		go func(r AnalysisRule) {
			defer wg.Done()
			issues := r.Check(hunks)
			issueChan <- issues
		}(rule)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(issueChan)
	}()

	// Collect all issues from the channel
	var allIssues []models.Issue
	for issues := range issueChan {
		allIssues = append(allIssues, issues...)
	}

	return allIssues
}
