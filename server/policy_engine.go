// Policy Engine — thin wrapper over internal/policy.
package server

import (
	"database/sql"

	"github.com/onsomlem/cocopilot/internal/policy"
)

// Type aliases for backward compatibility.
type PolicyContext = policy.PolicyContext
type PolicyViolation = policy.PolicyViolation
type PolicyEngine = policy.PolicyEngine
type policyRateTracker = policy.RateTracker

// NewPolicyEngine creates a new PolicyEngine.
func NewPolicyEngine(db *sql.DB) *PolicyEngine { return policy.NewPolicyEngine(db) }

// newPolicyRateTracker creates a new rate tracker (used by tests).
func newPolicyRateTracker() *policyRateTracker { return policy.NewRateTracker() }

// EvaluatePolicy delegates to the internal policy package.
func EvaluatePolicy(ctx PolicyContext, policies []Policy, rt *policyRateTracker, database *sql.DB) (bool, []PolicyViolation) {
	return policy.EvaluatePolicy(ctx, policies, rt, database)
}
