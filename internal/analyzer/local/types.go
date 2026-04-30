package local

import (
	"code-review-agent/internal/analyzer/local/rules"
	"code-review-agent/internal/models"
)

// AnalysisRule defines the interface for all local analysis rules
type AnalysisRule interface {
	Name() string
	Check(hunks []models.DiffHunk) []models.Issue
}

// RuleRegistry holds all registered analysis rules
type RuleRegistry struct {
	rules []AnalysisRule
}

// NewRuleRegistry creates a new rule registry with all available rules
func NewRuleRegistry() *RuleRegistry {
	return &RuleRegistry{
		rules: []AnalysisRule{
			&LargeFunctionRule{},
			&TodoCommentRule{},
			&HardcodedSecretsRule{},
			&SQLInjectionRule{},
			&DeprecatedFunctionRule{},
			&MissingErrorHandlingRule{},
		},
	}
}

// GetRules returns all registered rules
func (r *RuleRegistry) GetRules() []AnalysisRule {
	return r.rules
}

// AddRule adds a new rule to the registry
func (r *RuleRegistry) AddRule(rule AnalysisRule) {
	r.rules = append(r.rules, rule)
}

// Rule implementations
type LargeFunctionRule struct{}

func (r *LargeFunctionRule) Name() string {
	return "large_function"
}

func (r *LargeFunctionRule) Check(hunks []models.DiffHunk) []models.Issue {
	return rules.CheckLargeFunction(hunks)
}

type TodoCommentRule struct{}

func (r *TodoCommentRule) Name() string {
	return "todo_comment"
}

func (r *TodoCommentRule) Check(hunks []models.DiffHunk) []models.Issue {
	return rules.CheckTodoComment(hunks)
}

type HardcodedSecretsRule struct{}

func (r *HardcodedSecretsRule) Name() string {
	return "hardcoded_secrets"
}

func (r *HardcodedSecretsRule) Check(hunks []models.DiffHunk) []models.Issue {
	return rules.CheckHardcodedSecrets(hunks)
}

type SQLInjectionRule struct{}

func (r *SQLInjectionRule) Name() string {
	return "sql_injection"
}

func (r *SQLInjectionRule) Check(hunks []models.DiffHunk) []models.Issue {
	return rules.CheckSQLInjection(hunks)
}

type DeprecatedFunctionRule struct{}

func (r *DeprecatedFunctionRule) Name() string {
	return "deprecated_function"
}

func (r *DeprecatedFunctionRule) Check(hunks []models.DiffHunk) []models.Issue {
	return rules.CheckDeprecatedFunction(hunks)
}

type MissingErrorHandlingRule struct{}

func (r *MissingErrorHandlingRule) Name() string {
	return "missing_error_handling"
}

func (r *MissingErrorHandlingRule) Check(hunks []models.DiffHunk) []models.Issue {
	return rules.CheckMissingErrorHandling(hunks)
}
