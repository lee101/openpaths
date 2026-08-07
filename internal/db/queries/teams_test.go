package queries

import (
	"errors"
	"testing"
	"time"
)

func TestValidateInviteBindsEmailAndState(t *testing.T) {
	now := time.Now()
	valid := TeamInvite{Email: "Invited@Example.com", Expires: now.Add(time.Hour)}
	if err := validateInvite(valid, "invited@example.com", now); err != nil {
		t.Fatalf("valid invite rejected: %v", err)
	}
	if err := validateInvite(valid, "other@example.com", now); !errors.Is(err, errInviteEmail) {
		t.Fatalf("wrong email error = %v", err)
	}
	usedAt := now.Add(-time.Minute)
	used := valid
	used.Accepted = &usedAt
	if err := validateInvite(used, valid.Email, now); !errors.Is(err, errInviteUsed) {
		t.Fatalf("used invite error = %v", err)
	}
	expired := valid
	expired.Expires = now.Add(-time.Minute)
	if err := validateInvite(expired, valid.Email, now); !errors.Is(err, errInviteExpired) {
		t.Fatalf("expired invite error = %v", err)
	}
}
