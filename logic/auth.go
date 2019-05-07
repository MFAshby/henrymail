package logic

import (
	"database/sql"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"henrymail/config"
	"henrymail/database"
	"henrymail/models"
)

/**
 * User administration functions
 */
func Login(username, password string) (*models.User, error) {
	user, e := models.UserByUsername(database.DB, username)
	if e != nil {
		return nil, e
	}
	e = bcrypt.CompareHashAndPassword(user.Passwordbytes, []byte(password))
	if e != nil {
		return nil, e
	}
	return user, e
}

func NewUser(db *sql.DB, username, password string, admin bool) (*models.User, error) {
	passwordBytes, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if e != nil {
		return nil, e
	}
	user := &models.User{
		Username:      username,
		Passwordbytes: passwordBytes,
		Admin:         admin,
	}
	e = database.Transact(db, func(tx *sql.Tx) error {
		e := user.Save(tx)
		if e != nil {
			return e
		}

		defaultMailboxes := config.GetStringSlice(config.DefaultMailboxes)
		for _, name := range defaultMailboxes {
			mailbox := &models.Mailbox{
				Userid:      user.ID,
				Uidnext:     1,
				Uidvalidity: 1,
				Subscribed:  true,
				Name:        name,
			}
			e = mailbox.Save(tx)
			if e != nil {
				return e
			}
		}
		return nil
	})
	return user, e
}

func ChangePassword(username, existingpassword, newpassword, newpassword2 string) error {
	user, e := Login(username, existingpassword)
	if e != nil {
		return e
	}
	if newpassword == "" {
		return errors.New("You must enter a password")
	}
	if newpassword != newpassword2 {
		return errors.New("Passwords don't match")
	}
	passwordBytes, e := bcrypt.GenerateFromPassword([]byte(newpassword), bcrypt.DefaultCost)
	if e != nil {
		return e
	}
	user.Passwordbytes = passwordBytes
	return user.Save(database.DB)
}
