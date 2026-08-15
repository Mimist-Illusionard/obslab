package models

import (
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       string `json:"id"`
	Name     string `json:"username"`
	passHash string
}

func (u *User) CheckPassword(pass string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.passHash), []byte(pass))
	return err == nil
}

func (u *User) ChangePassword(pass string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.passHash = string(hash)
	return nil
}
