package cron

import (
	"context"
	"fmt"
	"log"

	"github.com/openpaths/openpaths/internal/auth"
	"github.com/openpaths/openpaths/internal/db/queries"
)

// ServiceKeyDeps carries the query handles needed to mint a gateway key for a
// background service identity.
type ServiceKeyDeps struct {
	APIKeyQ *queries.APIKeyQueries
	UserQ   *queries.UserQueries
	CreditQ *queries.CreditQueries
}

// EnsureServiceKey guarantees a valid op- API key for a dedicated service user.
// When the user does not exist it is created with an unusable random password;
// credit is granted so requests pass BalanceCheck; stale keys from previous
// boots are revoked; and a fresh key is minted with the given rate limit.
//
// Both the model prober and the live-eval runner call the gateway through the
// normal /v1/chat/completions path, so they share this provisioning logic.
func EnsureServiceKey(ctx context.Context, deps ServiceKeyDeps, email, displayName, keyName string, rateLimitRPM int, minBalanceCents, topupCents int64) (string, string, error) {
	if deps.APIKeyQ == nil || deps.UserQ == nil {
		return "", "", fmt.Errorf("no %s API key configured and key/user queries unavailable", keyName)
	}

	user, err := deps.UserQ.GetByEmail(ctx, email)
	if err != nil {
		pwHash, herr := auth.HashPassword(randomServiceSecret())
		if herr != nil {
			return "", "", fmt.Errorf("hash service password: %w", herr)
		}
		user, err = deps.UserQ.Create(ctx, email, pwHash, displayName)
		if err != nil {
			return "", "", fmt.Errorf("create service user: %w", err)
		}
	}

	if deps.CreditQ != nil {
		if ierr := deps.CreditQ.InitBalance(ctx, user.ID); ierr != nil {
			log.Printf("%s: init balance: %v", keyName, ierr)
		}
		if bal, berr := deps.CreditQ.GetBalance(ctx, user.ID); berr == nil && bal < minBalanceCents {
			if derr := deps.CreditQ.Deposit(ctx, user.ID, topupCents, keyName+" credit grant"); derr != nil {
				log.Printf("%s: credit grant: %v", keyName, derr)
			}
		}
	}

	// Revoke stale keys from previous boots so live keys don't pile up.
	if existing, lerr := deps.APIKeyQ.ListByUser(ctx, user.ID); lerr == nil {
		for _, k := range existing {
			if k.Name == keyName && !k.Revoked {
				if rerr := deps.APIKeyQ.Revoke(ctx, k.ID, user.ID); rerr != nil {
					log.Printf("%s: revoke stale key: %v", keyName, rerr)
				}
			}
		}
	}

	raw, hash, prefix, err := auth.GenerateAPIKey()
	if err != nil {
		return "", "", fmt.Errorf("generate service key: %w", err)
	}
	key, err := deps.APIKeyQ.Create(ctx, user.ID, hash, prefix, keyName)
	if err != nil {
		return "", "", fmt.Errorf("store service key: %w", err)
	}
	if rerr := deps.APIKeyQ.SetRateLimit(ctx, key.ID, rateLimitRPM); rerr != nil {
		log.Printf("%s: set rate limit: %v", keyName, rerr)
	}
	return raw, key.ID, nil
}

func randomServiceSecret() string {
	// Reuse the API-key generator purely as a source of crypto-random bytes for
	// the service-account password (it can never be used to log in interactively).
	raw, _, _, err := auth.GenerateAPIKey()
	if err != nil {
		return "service-account-no-login"
	}
	return raw
}
