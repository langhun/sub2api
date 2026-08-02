package supporttickets

import "testing"

func TestUserReplyStateMachine(t *testing.T) {
	for _, tt := range []struct {
		status string
		want   bool
	}{
		{StatusOpen, true},
		{StatusWaitingUser, true},
		{StatusResolved, false},
		{StatusClosed, false},
	} {
		if got := isAllowedUserReply(tt.status); got != tt.want {
			t.Fatalf("isAllowedUserReply(%q) = %t, want %t", tt.status, got, tt.want)
		}
	}
}

func TestUserReopenOnlyAllowsResolvedToOpen(t *testing.T) {
	for _, tt := range []struct {
		from string
		to   string
		want bool
	}{
		{StatusResolved, StatusOpen, true},
		{StatusOpen, StatusOpen, false},
		{StatusWaitingUser, StatusOpen, false},
		{StatusClosed, StatusOpen, false},
		{StatusResolved, StatusWaitingUser, false},
	} {
		if got := isAllowedUserStatusTransition(tt.from, tt.to); got != tt.want {
			t.Fatalf("isAllowedUserStatusTransition(%q, %q) = %t, want %t", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestAdminStatusStateMachine(t *testing.T) {
	for _, tt := range []struct {
		from string
		to   string
		want bool
	}{
		{StatusOpen, StatusWaitingUser, true},
		{StatusOpen, StatusResolved, true},
		{StatusOpen, StatusClosed, true},
		{StatusWaitingUser, StatusOpen, true},
		{StatusWaitingUser, StatusResolved, true},
		{StatusResolved, StatusOpen, true},
		{StatusResolved, StatusClosed, true},
		{StatusOpen, StatusOpen, false},
		{StatusResolved, StatusWaitingUser, false},
		{StatusClosed, StatusOpen, false},
	} {
		if got := isAllowedAdminStatusTransition(tt.from, tt.to); got != tt.want {
			t.Fatalf("isAllowedAdminStatusTransition(%q, %q) = %t, want %t", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestStatusTimestampAssignments(t *testing.T) {
	for _, tt := range []struct {
		from string
		to   string
		want string
	}{
		{StatusOpen, StatusResolved, ", resolved_at=NOW(), closed_at=NULL"},
		{StatusResolved, StatusClosed, ", closed_at=NOW()"},
		{StatusOpen, StatusClosed, ", resolved_at=NULL, closed_at=NOW()"},
		{StatusWaitingUser, StatusOpen, ", resolved_at=NULL, closed_at=NULL"},
	} {
		if got := statusTimestampAssignments(tt.from, tt.to); got != tt.want {
			t.Fatalf("statusTimestampAssignments(%q, %q) = %q, want %q", tt.from, tt.to, got, tt.want)
		}
	}
}
