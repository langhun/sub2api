package service

import "fmt"

// OpsClientDisconnectExclusionSQL returns a SQL predicate that excludes
// client-owned disconnect logs (499) from aggregate counting queries while
// still keeping the raw log rows available for drilldown/debugging.
func OpsClientDisconnectExclusionSQL(statusExpr, ownerExpr string) string {
	return fmt.Sprintf(
		"NOT (COALESCE(%s, '') = 'client' AND COALESCE(%s, 0) = 499)",
		ownerExpr,
		statusExpr,
	)
}

// OpsCountableErrorSQL returns a SQL predicate for error aggregates that
// should count only real service-visible errors and exclude client disconnects.
func OpsCountableErrorSQL(statusExpr, ownerExpr string) string {
	return fmt.Sprintf(
		"COALESCE(%s, 0) >= 400 AND %s",
		statusExpr,
		OpsClientDisconnectExclusionSQL(statusExpr, ownerExpr),
	)
}
