// Package model
// @Description:
package model

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"

	"github.com/legolasljl/studyclaw/utils"
)

func init() {
	go check()
}

var (
	pushFunc func(id, kind, message string)
)

func SetPush(push func(id, kind, message string)) {
	pushFunc = push
}

// Query
/**
 * @Description: 查询所有未掉线的用户
 * @return []*User
 * @return error
 */
func Query() ([]*User, error) {
	var (
		users  []*User
		result []*User
	)
	_ = engine.Ping()
	err := engine.Where("status=?", 1).Find(&users)
	if err != nil {
		return users, err
	}

	for _, user := range users {
		if ok, _ := utils.CheckUserCookie(user.ToCookies()); ok {
			result = append(result, user)
		} else {
			log.Warningln(user.Nick + "的cookie已失效")
			changeStatus(user.Uid, 0)
			if pushFunc != nil {
				pushFunc(user.PushId, "flush", user.Nick+"的cookie已失效")
			}
		}
	}
	return result, err
}

func changeStatus(uid string, status int) {
	_ = engine.Ping()
	_, err := engine.Table(new(User)).Where("uid=?", uid).Update(map[string]any{"status": status})
	if err != nil {
		log.Errorln("改变status失败" + err.Error())
		return
	}
}

func QueryFailUser() ([]*User, error) {
	var users []*User
	_ = engine.Ping()
	err := engine.Where("status=?", 0).Find(&users)
	if err != nil {
		return users, err
	}
	return users, err
}

// QueryByPushID
/**
 * @Description: 根据推送平台的key查询用户
 * @return []*User
 * @return error
 */
func QueryByPushID(pushID string) ([]*User, error) {
	var (
		users  []*User
		result []*User
	)
	_ = engine.Ping()
	err := engine.Where("status=? and push_id=?", 1, pushID).Find(&users)
	if err != nil {
		return users, err
	}

	for _, user := range users {
		if ok, _ := utils.CheckUserCookie(user.ToCookies()); ok {
			result = append(result, user)
		} else {
			log.Warningln(user.Nick + "的cookie已失效")
			changeStatus(user.Uid, 0)
			if pushFunc != nil {
				pushFunc(user.PushId, "flush", user.Nick+"的cookie已失效")
			}
		}
	}
	return result, err
}

// Find
/**
 * @Description:
 * @param uid
 * @return *User
 */
func Find(uid string) *User {
	u := new(User)
	_, err := engine.Where("uid=?", uid).Get(u)
	if err != nil {
		return nil
	}
	return u
}

// AddUser
/**
 * @Description:
 * @param user
 * @return error
 */
func AddUser(user *User) error {
	_ = engine.Ping()
	count, _ := engine.Where("uid=?", user.Uid).Count(new(User))
	if count < 1 {
		user.Status = 1
		_, err := engine.InsertOne(user)
		if err != nil {
			return err
		}
	} else {
		user.Status = 1
		err := UpdateUser(user)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateUser
/**
 * @Description:
 * @param user
 * @return error
 */
func UpdateUser(user *User) error {
	_, err := engine.Where("uid=?", user.Uid).Update(user)
	if err != nil {
		return err
	}
	return nil
}

// DeleteUser
/* @Description:
 * @param uid
 * @return error
 */
func DeleteUser(uid string) error {
	if err := engine.Ping(); err != nil {
		return fmt.Errorf("database connection lost: %w", err)
	}
	_, err := engine.Where("uid=?", uid).Delete(new(User))
	return err
}

// ToCookies
/**
 * @Description: 获取user的cookie
 * @receiver u
 * @return []*http.Cookie
 */
func (u *User) ToCookies() []*http.Cookie {
	cookie := &http.Cookie{
		Name:     "token",
		Value:    u.Token,
		Path:     "/",
		Domain:   "xuexi.cn",
		Expires:  time.Now().Add(time.Hour * 12),
		Secure:   false,
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
	}
	return []*http.Cookie{cookie}
}

func TokenToCookies(token string) []*http.Cookie {
	cookie := &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		Domain:   "xuexi.cn",
		Expires:  time.Now().Add(time.Hour * 12),
		Secure:   false,
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
	}
	return []*http.Cookie{cookie}
}

func (u *User) ToBrowserCookies() []playwright.OptionalCookie {
	cookie := playwright.OptionalCookie{
		Name:     "token",
		Value:    u.Token,
		Path:     playwright.String("/"),
		Domain:   playwright.String(".xuexi.cn"),
		Expires:  playwright.Float(float64(time.Now().Add(time.Hour * 12).Unix())),
		Secure:   playwright.Bool(false),
		HttpOnly: playwright.Bool(false),
		SameSite: playwright.SameSiteAttributeStrict,
	}
	return []playwright.OptionalCookie{cookie}
}

// authCheckURL keeps a session alive. A healthy session echoes the token with
// Max-Age=43200 (12h); shortly before the server-side limit it issues a
// DIFFERENT token. An expired session returns 200 but no token cookie at all.
const authCheckURL = "https://pc-api.xuexi.cn/open/api/auth/check"

// QueryActive returns enabled accounts WITHOUT any network validation.
// The keepalive job must use this: validating first with score/get marks an
// account as failed at the exact moment its token expires, before auth/check
// ever gets a chance to rotate it.
func QueryActive() ([]*User, error) {
	var users []*User
	_ = engine.Ping()
	err := engine.Where("status=?", 1).Find(&users)
	return users, err
}

// TokenFromCookies extracts the token cookie from a response.
func TokenFromCookies(cookies []*http.Cookie) string {
	for _, c := range cookies {
		if c != nil && c.Name == "token" && c.Value != "" {
			return c.Value
		}
	}
	return ""
}

// renewUserToken refreshes one account. A network/TLS error is never treated
// as an expired account; only a response without a token cookie means expiry.
func renewUserToken(user *User) {
	response, err := utils.GetClient().R().SetCookies(user.ToCookies()...).Get(authCheckURL)
	if err != nil || response == nil {
		log.Warningln("保活请求失败，跳过本次判定: ", err)
		return
	}
	token := TokenFromCookies(response.Cookies())
	if token == "" {
		log.Warningln(user.Nick + "的cookie已失效")
		changeStatus(user.Uid, 0)
		if pushFunc != nil {
			pushFunc(user.PushId, "flush", user.Nick+"的cookie已失效，请重新扫码登录")
		}
		return
	}
	if user.Token != token {
		user.Token = token
		if err := UpdateUser(user); err != nil {
			log.Errorln("保活写入失败: " + err.Error())
			return
		}
		log.Infoln("用户" + user.Nick + "的ck已成功保活cookie")
		return
	}
	log.Debugln("用户" + user.Nick + "的会话有效，服务端已刷新cookie有效期")
}

func check() {
	defer func() {
		err := recover()
		if err != nil {
			log.Errorf("%v 出现错误，%v", "auth check", err)
		}
	}()
	c := cron.New()
	cr := "*/10 * * * *"
	if crEnv, ok := os.LookupEnv("CHECK_ENV"); ok && crEnv != "" {
		cr = crEnv
		log.Infoln("已成功自定义保活cron : " + cr)
	}
	_, err := c.AddFunc(cr, func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("保活任务异常，本次跳过: %v", r)
			}
		}()
		log.Infoln("开始执行保活任务")
		users, err := QueryActive()
		if err != nil {
			log.Errorln("读取用户失败: " + err.Error())
			return
		}
		for _, user := range users {
			renewUserToken(user)
		}
	})
	if err != nil {
		log.Errorln("添加保活任务失败" + err.Error())
		return
	}
	c.Start()
}
