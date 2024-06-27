package user

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kekaadrenalin/dockhook/pkg/helper"
	"github.com/stretchr/testify/assert"
)

func Test_AuthBasic_AuthMiddleware_happy(t *testing.T) {
	usersDB := UsersDatabase{
		Users: map[string]*User{
			"test_user": {Username: "test_user", Password: helper.Sha512sum("test_pass")},
		},
	}
	authContext := NewBasicAuth(usersDB)

	handler := authContext.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server := httptest.NewServer(handler)
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header.Set("Authorization", "Basic "+basicAuth("test_user", "test_pass"))

	// Выполнение запроса
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func Test_AuthBasic_AuthMiddleware_error(t *testing.T) {
	usersDB := UsersDatabase{
		Users: map[string]*User{
			"test_user": {Username: "test_user", Password: helper.Sha512sum("test_pass")},
		},
	}
	authContext := NewBasicAuth(usersDB)

	handler := authContext.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server := httptest.NewServer(handler)
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header.Set("Authorization", "Basic "+basicAuth("test_user", "wrong_pass"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func Test_AuthBasic_AuthMiddleware_block_user(t *testing.T) {
	usersDB := UsersDatabase{
		Users: map[string]*User{
			"test_user": {Username: "test_user", Password: helper.Sha512sum("test_pass")},
		},
	}
	authContext := NewBasicAuth(usersDB)

	handler := authContext.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server := httptest.NewServer(handler)
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header.Set("Authorization", "Basic "+basicAuth("test_user", "wrong_pass"))

	for i := 0; i < 11; i++ {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		resp.Body.Close()
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
}

func Test_AuthBasic_AuthMiddleware_block_user_timeout(t *testing.T) {
	usersDB := UsersDatabase{
		Users: map[string]*User{
			"test_user": {Username: "test_user", Password: helper.Sha512sum("test_pass")},
		},
	}
	authContext := NewBasicAuth(usersDB)
	authContext.blockedUsers["test_user"] = time.Now().Add(-2 * time.Hour)

	handler := authContext.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server := httptest.NewServer(handler)
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header.Set("Authorization", "Basic "+basicAuth("test_user", "test_pass"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
