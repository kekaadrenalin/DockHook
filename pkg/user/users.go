package user

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/go-chi/jwtauth/v5"
	"github.com/kekaadrenalin/dockhook/pkg/helper"
	"gopkg.in/yaml.v3"
)

type User struct {
	Username string `json:"username" yaml:"-"`
	Email    string `json:"email" yaml:"email"`
	Name     string `json:"name" yaml:"name"`
	Password string `json:"-" yaml:"password"`
}

type UsersDatabase struct {
	Users    map[string]*User `yaml:"users"`
	LastRead time.Time        `yaml:"-"`
	LastSave time.Time        `yaml:"-"`
	Path     string           `yaml:"-"`
}

type contextKey string

const remoteUser contextKey = "remoteUser"

var ErrInvalidCredentials = errors.New("invalid credentials")

func newUser(username, email, name string) User {
	return User{
		Username: username,
		Email:    email,
		Name:     name,
	}
}

func ReadUsersFromFile(path string) (UsersDatabase, error) {
	users, err := decodeUsersFromFile(path)
	if err != nil {
		return users, err
	}

	users.LastRead = time.Now()
	users.Path = path

	return users, nil
}

func CreateUser(path string, user User, hashPassword bool) (User, error) {
	if hashPassword {
		user.Password = helper.Sha512sum(user.Password)
	}

	users, err := ReadUsersFromFile(path)
	if err != nil {
		return user, err
	}

	if users.Users == nil {
		users.Users = make(map[string]*User)
	}

	if _, exists := users.Users[user.Username]; exists {
		return user, fmt.Errorf("user %s is exists", user.Username)
	}

	users.Users[user.Username] = &user

	if _, err := saveUsersToFile(users, path); err != nil {
		return user, err
	}

	return user, nil
}

func saveUsersToFile(users UsersDatabase, path string) (UsersDatabase, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if _, err = helper.CreateDir(path); err != nil {
			return users, err
		}
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return users, err
	}
	defer file.Close()

	data, err := yaml.Marshal(&users)
	if err != nil {
		return users, err
	}

	if _, err = file.Write(data); err != nil {
		return users, err
	}

	users.LastSave = time.Now()

	return users, nil
}

func decodeUsersFromFile(path string) (UsersDatabase, error) {
	users := UsersDatabase{}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		users.Users = map[string]*User{}

		return users, nil
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return users, err
	}

	if err := yaml.NewDecoder(file).Decode(&users); err != nil {
		log.Warningf("wrong file: %s\n", err)

		return users, nil
	}
	defer file.Close()

	for username, user := range users.Users {
		user.Username = username
		if user.Password == "" {
			log.Fatalf("User %s has no password", username)
		}

		if len(user.Password) != 128 {
			log.Fatalf("User %s has an invalid password hash", username)
		}

		if user.Name == "" {
			user.Name = username
		}
	}

	return users, nil
}

func (u *UsersDatabase) readFileIfChanged() error {
	if u.Path == "" {
		return nil
	}

	info, err := os.Stat(u.Path)
	if err != nil {
		return err
	}

	if info.ModTime().After(u.LastRead) {
		log.Infof("Found changes to %s. Updating users...", u.Path)
		users, err := decodeUsersFromFile(u.Path)
		if err != nil {
			return err
		}
		u.Users = users.Users
		u.LastRead = time.Now()
	}

	return nil
}

func (u *UsersDatabase) Find(username string) *User {
	if err := u.readFileIfChanged(); err != nil {
		log.Errorf("Error reading users file: %s", err)
	}

	user, ok := u.Users[username]
	if !ok {
		return nil
	}

	return user
}

func (u *UsersDatabase) FindByPassword(username, password string) *User {
	user := u.Find(username)
	if user == nil {
		return nil
	}

	if user.Password != helper.Sha512sum(password) {
		return nil
	}

	return user
}

//goland:noinspection GoNameStartsWithPackageName
func UserFromContext(ctx context.Context) *User {
	if user, ok := ctx.Value(remoteUser).(User); ok {
		return &user
	}

	if _, claims, err := jwtauth.FromContext(ctx); err == nil {
		username, ok := claims["username"].(string)
		if !ok || username == "" {
			return nil
		}

		email := claims["email"].(string)
		name := claims["name"].(string)
		user := newUser(username, email, name)

		return &user
	}

	return nil
}

func RequireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user != nil {
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	})
}
