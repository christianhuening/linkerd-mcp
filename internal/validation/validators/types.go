package validators

import "time"

// Severity represents the severity level of a validation issue
type Severity string

const (
	// SeverityError indicates a critical issue that must be fixed
	SeverityError Severity = "error"
	// SeverityWarning indicates an issue that should be reviewed
	SeverityWarning Severity = "warning"
	// SeverityInfo provides informational feedback
	SeverityInfo Severity = "info"
)

// Issue represents a single validation issue
type Issue struct {
	Severity    Severity `json:"severity"`
	Message     string   `json:"message"`
	Field       string   `json:"field"`
	Remediation string   `json:"remediation,omitempty"`
	Code        string   `json:"code,omitempty"`
}

// ValidationResult represents the result of validating a single resource
type ValidationResult struct {
	ResourceType string    `json:"resourceType"`
	Name         string    `json:"name"`
	Namespace    string    `json:"namespace"`
	Valid        bool      `json:"valid"`
	Issues       []Issue   `json:"issues"`
	Timestamp    time.Time `json:"timestamp"`
}

// ClusterValidationReport represents a complete validation report for the cluster
type ClusterValidationReport struct {
	TotalResources int                `json:"totalResources"`
	ValidResources int                `json:"validResources"`
	Results        []ValidationResult `json:"results"`
	Summary        ValidationSummary  `json:"summary"`
	Timestamp      time.Time          `json:"timestamp"`
}

// ValidationSummary provides summary statistics
type ValidationSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
}

// AddIssue adds an issue to the validation result
func (vr *ValidationResult) AddIssue(severity Severity, message, field, code, remediation string) {
	vr.Issues = append(vr.Issues, Issue{
		Severity:    severity,
		Message:     message,
		Field:       field,
		Code:        code,
		Remediation: remediation,
	})
}

// Finalize marks the validation as complete and sets validity
func (vr *ValidationResult) Finalize() {
	vr.Timestamp = time.Now()
	vr.Valid = true
	for _, issue := range vr.Issues {
		if issue.Severity == SeverityError {
			vr.Valid = false
			break
		}
	}
}

// AddResult adds a validation result to the report and updates summary
func (cvr *ClusterValidationReport) AddResult(result ValidationResult) {
	cvr.Results = append(cvr.Results, result)
	cvr.TotalResources++
	if result.Valid {
		cvr.ValidResources++
	}

	for _, issue := range result.Issues {
		switch issue.Severity {
		case SeverityError:
			cvr.Summary.Errors++
		case SeverityWarning:
			cvr.Summary.Warnings++
		case SeverityInfo:
			cvr.Summary.Info++
		}
	}
}

// Finalize marks the report as complete
func (cvr *ClusterValidationReport) Finalize() {
	cvr.Timestamp = time.Now()
}
