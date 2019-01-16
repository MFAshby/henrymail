package web

import (
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"henrymail/config"
	"henrymail/model"
	"net/http"
	"time"
)

type UserClaims struct {
	jwt.StandardClaims
	*model.Usr
}

func (c UserClaims) Valid() error {
	return c.StandardClaims.Valid()
}

func (wa *wa) login(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")
	if email == "" {
		wa.loginView.Render(w, nil)
		return
	}

	usr, err := wa.lg.Login(email, password)
	if err != nil {
		wa.loginView.Render(w, err)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, UserClaims{
		jwt.StandardClaims{},
		usr,
	})
	tokenString, err := token.SignedString(wa.jwtSecret)
	if err != nil {
		wa.loginView.Render(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     config.GetString(config.JwtCookieName),
		Value:    tokenString,
		HttpOnly: true,
		Secure:   config.GetBool(config.WebAdminUseTls),
		Domain:   config.GetString(config.ServerName) + config.GetString(config.WebAdminAddress),
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
		Domain:   config.GetString(config.ServerName) + config.GetString(config.WebAdminAddress),
	})
	wa.loginView.Render(w, nil)
	w.WriteHeader(200)
}

func (wa *wa) checkAdmin(next AuthenticatedHandler) http.Handler {
	return wa.checkLogin(func(w http.ResponseWriter, r *http.Request, user *model.Usr) {
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
			return wa.jwtSecret, nil
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
		next(w, r, claims.Usr)
	})
}
