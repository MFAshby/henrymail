package web

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"henrymail/config"
	"henrymail/logic"
	"henrymail/models"
	"log"
	"net/http"
	"time"
)

type AuthenticatedHandler = func(w http.ResponseWriter, r *http.Request, u *models.User)

type UserClaims struct {
	jwt.StandardClaims
	*models.User
}

func (c UserClaims) Valid() error {
	return c.StandardClaims.Valid()
}

const (
	JwtSecretKeyName = "jwt_secret"
)

func (w *wa) jwtSecret() []byte {
	key, e := models.KeyByName(w.db, JwtSecretKeyName)
	if e != nil {
		log.Print(e)
		log.Println("Generating new jwtSecret")
		newSecret := make([]byte, 64)
		_, e := rand.Read(newSecret)
		if e != nil {
			// Unable to generate a random key, can't recover
			log.Fatal(e)
		}

		if key == nil {
			key = &models.Key{
				Name: JwtSecretKeyName,
			}
		}
		key.Key = newSecret
		e = key.Save(w.db)
		if e != nil {
			log.Fatal(e)
		}
	}
	return key.Key
}

func (wa *wa) login(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" {
		wa.loginView.render(w, nil)
		return
	}

	usr, err := logic.Login(wa.db, username, password)
	if err != nil {
		wa.loginView.render(w, err)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, UserClaims{
		jwt.StandardClaims{},
		usr,
	})
	tokenString, err := token.SignedString(wa.jwtSecret())
	if err != nil {
		wa.loginView.render(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     config.GetString(config.JwtCookieName),
		Value:    tokenString,
		HttpOnly: true,
		Secure:   config.GetBool(config.WebAdminUseTls),
		Domain:   config.GetCookieDomain(),
		Expires:  time.Now().Add(time.Hour * 240),
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (wa *wa) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     config.GetString(config.JwtCookieName),
		Value:    "bogus",
		Expires:  time.Now(),
		HttpOnly: true,
		Secure:   config.GetBool(config.WebAdminUseTls),
		Domain:   config.GetCookieDomain(),
	})
	wa.loginView.render(w, nil)
}

func (wa *wa) checkAdmin(next AuthenticatedHandler) http.Handler {
	return wa.checkLogin(func(w http.ResponseWriter, r *http.Request, user *models.User) {
		if !user.Admin {
			wa.renderError(w, errors.New("You are not an administrator"))
			return
		}
		next(w, r, user)
	})
}

func (wa *wa) checkLogin(next AuthenticatedHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, e := r.Cookie(config.GetString(config.JwtCookieName))
		if e == http.ErrNoCookie {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		var claims UserClaims
		_, err := jwt.ParseWithClaims(cookie.Value, &claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return wa.jwtSecret(), nil
		})
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		err = claims.Valid()
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		next(w, r, claims.User)
	})
}
