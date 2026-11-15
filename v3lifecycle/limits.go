package lifecycle

import "math"

// ValidateLimits never supplies a default or interprets zero as unlimited.
func ValidateLimits(l Limits) Result {
	values := []int{l.MaxIDBytes, l.MaxMembers, l.MaxAxes, l.MaxRefs, l.MaxObligations, l.MaxPartitions, l.MaxRanges, l.MaxDepth}
	for _, v := range values {
		if v <= 0 {
			return result(NeedsDecision, "EXPLICIT_POSITIVE_LIMIT_REQUIRED", "Limits")
		}
	}
	if l.MaxRefs > math.MaxInt/16 || l.MaxIDBytes > math.MaxInt/(l.MaxRefs*16) {
		return result(NeedsDecision, "LIMIT_BUDGET_OVERFLOW", "Limits")
	}
	return coherent()
}
