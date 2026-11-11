package model

import (
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestTokenFromCookies(t *testing.T) {
	tests := []struct {
		name    string
		cookies []*http.Cookie
		want    string
	}{
		{"empty", nil, ""},
		{"no token", []*http.Cookie{{Name: "acw_tc", Value: "x"}}, ""},
		{"nil cookie", []*http.Cookie{nil}, ""},
		{"found", []*http.Cookie{{Name: "acw_tc", Value: "x"}, {Name: "token", Value: "abc"}}, "abc"},
		{"blank value", []*http.Cookie{{Name: "token", Value: ""}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TokenFromCookies(tt.cookies); got != tt.want {
				t.Fatalf("TokenFromCookies() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInsert(t *testing.T) {
	err := AddUser(&User{
		Nick:      "123",
		Uid:       "123",
		Token:     "123444444444444444444444444",
		LoginTime: 1031312,
	})
	if err != nil {
		log.Errorln(err.Error())
	}
}

func TestQuery(t *testing.T) {
	users, err := Query()
	if err != nil {
		log.Errorln(err.Error())
		return
	}
	log.Infoln(users[0].Uid)
}

func TestFind(t *testing.T) {
	user := Find("123")
	log.Infoln(user.Nick)
}
