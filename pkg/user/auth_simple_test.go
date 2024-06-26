package user

import (
	"testing"

	"github.com/kekaadrenalin/dockhook/pkg/helper"
	"github.com/stretchr/testify/assert"
)

func Test_AuthSimple_CreateToken_happy(t *testing.T) {
	usersDB := UsersDatabase{
		Users: map[string]*User{
			"test_user": {Username: "test_user", Password: helper.Sha512sum("test_pass")},
		},
	}
	authContext := NewSimpleAuth(usersDB)

	token, err := authContext.CreateToken("test_user", "test_pass")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func Test_AuthSimple_CreateToken_error(t *testing.T) {
	usersDB := UsersDatabase{
		Users: map[string]*User{
			"test_user": {Username: "test_user", Password: helper.Sha512sum("test_pass")},
		},
	}
	authContext := NewSimpleAuth(usersDB)

	token, err := authContext.CreateToken("test_user", "wrong_pass")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidCredentials, err)
	assert.Empty(t, token)
}

func Test_AuthSimple_AuthMiddleware_happy(t *testing.T) {
	usersDB := UsersDatabase{
		Users: map[string]*User{
			"test_user": {Username: "test_user", Password: helper.Sha512sum("test_pass")},
		},
	}
	authContext := NewSimpleAuth(usersDB)

	token, err := authContext.CreateToken("test_user", "test_pass")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func Test_AuthSimple_AuthMiddleware_wrong_password(t *testing.T) {
	usersDB := UsersDatabase{
		Users: map[string]*User{
			"test_user": {Username: "test_user", Password: helper.Sha512sum("test_pass")},
		},
	}
	authContext := NewSimpleAuth(usersDB)

	token, err := authContext.CreateToken("test_user", "wrong_pass")
	assert.Error(t, err)
	assert.Empty(t, token)
}
