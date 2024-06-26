package user

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"golang.org/x/time/rate"
)

type basicAuthContext struct {
	UsersDatabase UsersDatabase
	rateLimiter   map[string]*rate.Limiter
	blockedUsers  map[string]time.Time
	failureCount  map[string]int64
	mu            sync.Mutex
}

func NewBasicAuth(userDatabase UsersDatabase) *basicAuthContext {
	return &basicAuthContext{
		UsersDatabase: userDatabase,
		rateLimiter:   make(map[string]*rate.Limiter),
		blockedUsers:  make(map[string]time.Time),
		failureCount:  make(map[string]int64),
	}
}

func (a *basicAuthContext) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			a.httpError(w, http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(auth, "Basic ") {
			a.httpError(w, http.StatusUnauthorized)
			return
		}

		payload, err := base64.StdEncoding.DecodeString(auth[len("Basic "):])
		if err != nil {
			a.httpError(w, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(string(payload), ":", 2)
		if len(parts) != 2 {
			a.httpError(w, http.StatusUnauthorized)
			return
		}

		username, password := parts[0], parts[1]

		if a.isBlocked(username) {
			a.httpError(w, http.StatusTooManyRequests)
			return
		}

		if !a.allow(username) {
			a.blockUser(w, r, username)
			a.httpError(w, http.StatusTooManyRequests)
			return
		}

		if !a.authenticate(username, password) {
			a.touchFailureCount(w, r, username)
			a.httpError(w, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), remoteUser, User{Username: username})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *basicAuthContext) CreateToken(username, password string) (string, error) {
	log.Fatalf("CreateToken not implemented for proxy auth")

	return "", nil
}

func (a *basicAuthContext) authenticate(username, password string) bool {
	user := a.UsersDatabase.FindByPassword(username, password)

	return user != nil
}

func (a *basicAuthContext) httpError(w http.ResponseWriter, status int) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	http.Error(w, http.StatusText(status), status)
}

func (a *basicAuthContext) allow(username string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	limiter, exists := a.rateLimiter[username]
	if !exists {
		limiter = rate.NewLimiter(1, 5)
		a.rateLimiter[username] = limiter
	}

	return limiter.Allow()
}

func (a *basicAuthContext) isBlocked(username string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	blockTime, exists := a.blockedUsers[username]
	if !exists {
		return false
	}

	if time.Now().After(blockTime) {
		delete(a.blockedUsers, username)
		delete(a.failureCount, username)
		return false
	}

	return true
}

func (a *basicAuthContext) blockUser(w http.ResponseWriter, r *http.Request, username string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	log.Warningf("blocked user %s", username)
	log.Infof("RemoteAddr: %+v\n", r.RemoteAddr)
	log.Infof("Header: %+v\n", r.Header)

	blockTimer := 1 * time.Hour
	a.blockedUsers[username] = time.Now().Add(blockTimer)

	w.Header().Set("Retry-After", fmt.Sprintf("%d", int64(blockTimer.Seconds())))
}

func (a *basicAuthContext) touchFailureCount(w http.ResponseWriter, r *http.Request, username string) {
	a.failureCount[username] += 1

	if a.failureCount[username] > 10 {
		a.blockUser(w, r, username)
		return
	}

	log.Debugf("failure count touch: %s\n", username)
}
