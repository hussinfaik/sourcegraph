// Package sharedsecret generates client-authenticated OAuth2 access
// tokens derived from the ID key to allow different components of
// Sourcegraph to communicate securely without sharing the private
// key.
package sharedsecret

import (
	"sync"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"sourcegraph.com/sourcegraph/sourcegraph/auth"
	"sourcegraph.com/sourcegraph/sourcegraph/auth/accesstoken"
	"sourcegraph.com/sourcegraph/sourcegraph/auth/idkey"
	"sourcegraph.com/sourcegraph/sourcegraph/go-sourcegraph/sourcegraph"
)

// TokenSource returns an OAuth2 token source that produces temporary,
// client-authenticated shared secret access tokens that may be
// verified using the public ID key.
//
// Using a TokenSource to generate temporary secrets is preferable to
// using permanent shared secrets (which this package could
// theoretically also generate) because if tokens are leaked, they
// will eventually expire. It also fits more cleanly into the rest of
// the architecture, which assumes OAuth2. Finally, it avoids us
// having to develop our own signature scheme, which is easy to mess
// up (and could lead to a security vulnerability).
func TokenSource(k *idkey.IDKey, scope ...string) oauth2.TokenSource {
	return &tokenSource{k, scope}
}

type tokenSource struct {
	k     *idkey.IDKey
	scope []string
}

func (ts *tokenSource) Token() (*oauth2.Token, error) {
	return accesstoken.New(ts.k, auth.Actor{
		ClientID: ts.k.ID,
		Scope:    auth.UnmarshalScope(ts.scope),
	}, map[string]string{"GrantType": "SharedSecret"}, expiry)
}

// SelfSignedTokenSource returns an OAuth2 token source whose tokens
// are HMAC-signed using a key derived from the private ID key. They
// are shorter than tokens generated by TokenSource (~180 chars. vs
// 500+ chars) and thus are suitable for use in places where long
// tokens exceed length restrictions (e.g., git <1.9 credentials).
func ShortTokenSource(k *idkey.IDKey, scope ...string) oauth2.TokenSource {
	return &shortTokenSource{k, scope}
}

type shortTokenSource struct {
	k     *idkey.IDKey
	scope []string
}

func (ts *shortTokenSource) Token() (*oauth2.Token, error) {
	return accesstoken.NewSelfSigned(ts.k, ts.scope, map[string]string{"GrantType": "ShortSharedSecret"}, expiry)
}

// defensiveReuseTokenSource is a oauth2.TokenSource that holds a single token in memory
// and validates its expiry before each call to retrieve it with Token.
// If it's going to expire in less than defensiveExpiry, it will be auto-refreshed
// using the new TokenSource. It is based on oauth2.reuseTokenSource, except that
// it is more defensive in refreshing the access token.
type defensiveReuseTokenSource struct {
	new oauth2.TokenSource // called when t is going to expire soon.

	mu sync.Mutex // guards t
	t  *oauth2.Token
}

// Token returns the current token if it's still valid for defensiveExpiry duration,
// else will refresh the current token (using r.Context for HTTP client information)
// and return the new one.
func (s *defensiveReuseTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.valid() {
		return s.t, nil
	}
	t, err := s.new.Token()
	if err != nil {
		return nil, err
	}
	s.t = t
	return t, nil
}

// valid reports whether s.t is non-nil, has an AccessToken, and is not
// set to expire within defensiveExpiry time.
func (s *defensiveReuseTokenSource) valid() bool {
	if s.t == nil || s.t.AccessToken == "" || s.t.Expiry.IsZero() {
		return false
	}
	return s.t.Expiry.Add(-defensiveExpiry).Before(time.Now())
}

// DefensiveReuseTokenSource returns a TokenSource which repeatedly returns
// the same token as long as it's valid, starting with t. It is based on
// oauth2.ReuseTokenSource, with the only difference that it defensively
// generates a new token if the current token's expiry is less than
// defensiveExpiry into the future.
//
// DefensiveReuseTokenSource is typically used when the token must be passed
// to a forked subprocess that cannot refresh the token, thus guaranteeing that
// the subprocess will have a token that is valid for at least defensiveExpiry
// duration.
func DefensiveReuseTokenSource(t *oauth2.Token, src oauth2.TokenSource) oauth2.TokenSource {
	// Don't wrap a defensiveReuseTokenSource in itself. That would work,
	// but cause an unnecessary number of mutex operations.
	// Just build the equivalent one.
	if rt, ok := src.(*defensiveReuseTokenSource); ok {
		if t == nil {
			// Just use it directly.
			return rt
		}
		src = rt.new
	}
	return &defensiveReuseTokenSource{
		t:   t,
		new: src,
	}
}

// defensiveExpiry must be less than expiry (below), otherwise
// DefensiveReuseTokenSource will generate a new token every time.
//
// This also must be long enough for any single subprocess CLI
// operation to complete; for example, the worker runs "src"
// subprocesses (such as for importing srclib data) and passes the
// access token to them, and the token must be valid for the entire
// duration of the operation (which could be 10+ minutes for large
// imports).
const defensiveExpiry = 60 * time.Minute

// expiry must be greater than golang.org/x/oauth2's expiryDelta,
// which currently is 10 seconds. Otherwise tokens will be considered
// invalid immediately when they are issued.
//
// This also must be long enough for any single subprocess CLI
// operation to complete; for example, the worker runs "src"
// subprocesses (such as for importing srclib data) and passes the
// access token to them, and the token must be valid for the entire
// duration of the operation (which could be 10+ minutes for large
// imports).
const expiry = 3 * 60 * time.Minute

// NewContext returns a copy of ctx that uses a shared secret
// TokenSource as API credentials to authenticate future calls.
func NewContext(ctx context.Context, scope ...string) context.Context {
	return sourcegraph.WithCredentials(ctx, TokenSource(idkey.FromContext(ctx), scope...))
}
