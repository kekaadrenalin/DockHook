package user

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kekaadrenalin/dockhook/pkg/helper"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func Test_CreateUser_happy(t *testing.T) {
	tmpFile, err := createTempFile(t, "users:")
	if err != nil {
		panic(any(err))
	}
	defer os.Remove(tmpFile.Name())

	// Подготовка тестовых данных
	testUser := User{
		Username: "testuser",
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "testpassword",
	}

	createdUser, err := CreateUser(tmpFile.Name(), testUser, true)

	assert.NoError(t, err, "expected no error during user creation")
	assert.Equal(t, testUser.Username, createdUser.Username, "expected username to match")
	assert.Equal(t, testUser.Email, createdUser.Email, "expected email to match")
	assert.Equal(t, testUser.Name, createdUser.Name, "expected name to match")
	assert.NotEqual(t, testUser.Password, createdUser.Password, "expected password to be hashed")
}

func Test_CreateUser_error_exists(t *testing.T) {
	testUser := User{
		Username: "testuser",
		Email:    "test@example.com",
		Name:     "Test User",
		Password: helper.Sha512sum("testpassword"),
	}
	body := generateYml(t, testUser)

	tmpFile, err := createTempFile(t, body)
	if err != nil {
		panic(any(err))
	}
	defer os.Remove(tmpFile.Name())

	_, err = CreateUser(tmpFile.Name(), testUser, true)

	assert.Error(t, err, "expected error during user creation")
}

func Test_CreateUser_error_writable(t *testing.T) {
	tmpFile, err := createTempFile(t, "users:")
	if err != nil {
		panic(any(err))
	}
	defer os.Remove(tmpFile.Name())

	if err = os.Chmod(tmpFile.Name(), 0400); err != nil {
		t.Errorf("Error changing file permissions: %s", err)
	}

	testUser := User{
		Username: "testuser",
		Email:    "test@example.com",
		Name:     "Test User",
		Password: helper.Sha512sum("testpassword"),
	}

	_, err = CreateUser(tmpFile.Name(), testUser, true)

	assert.Error(t, err, "expected error during user creation")
}

func Test_FindByPassword_happy(t *testing.T) {
	testUser := User{
		Username: "testuser",
		Email:    "test@example.com",
		Name:     "Test User",
		Password: helper.Sha512sum("testpassword"),
	}

	usersDB := UsersDatabase{
		Users: map[string]*User{
			testUser.Username: &testUser,
		},
	}

	foundUser := usersDB.FindByPassword(testUser.Username, "testpassword")
	assert.NotNil(t, foundUser, "expected user to be found with correct password")
	assert.Equal(t, testUser.Username, foundUser.Username, "expected username to match")

	// Проверяем поиск с неверным паролем
	notFoundUser := usersDB.FindByPassword(testUser.Username, "wrongpassword")
	assert.Nil(t, notFoundUser, "expected no user to be found with incorrect password")
}

func Test_FindByPassword_wrong_password(t *testing.T) {
	testUser := User{
		Username: "testuser",
		Email:    "test@example.com",
		Name:     "Test User",
		Password: helper.Sha512sum("testpassword"),
	}

	usersDB := UsersDatabase{
		Users: map[string]*User{
			testUser.Username: &testUser,
		},
	}

	notFoundUser := usersDB.FindByPassword(testUser.Username, "wrongpassword")
	assert.Nil(t, notFoundUser, "expected no user to be found with incorrect password")
}

func Test_RequireAuthentication_happy(t *testing.T) {
	srv := httptest.NewServer(RequireAuthentication(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	assert.NoError(t, err, "expected no error in HTTP request")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "expected HTTP status 401 Unauthorized")
}

func createTempFile(t *testing.T, startString string) (*os.File, error) {
	tmpFile, err := os.CreateTemp("", "test-users.yaml")
	if err != nil {
		t.Fatalf("failed to create temporary file: %s", err)
	}

	if _, err = tmpFile.WriteString(startString); err != nil {
		return nil, err
	}

	return tmpFile, nil
}

func generateYml(t *testing.T, user User) string {
	data := map[string]map[string]User{
		"users": {
			user.Username: user,
		},
	}

	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		t.Fatalf("failed generate yml string: %s", err)
	}

	return string(yamlBytes)
}
