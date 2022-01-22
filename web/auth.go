package web

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	userkey  = "authorized_user"
	loginKey = "login_msg"
)

func leftLoginMsg(c *gin.Context, msg string) {
	session := sessions.Default(c)
	session.Set(loginKey, msg)
	session.Save()
}

func login(c *gin.Context) {
	session := sessions.Default(c)

	username := c.PostForm("username")
	passwd := c.PostForm("passwd")

	// Validate form input
	if strings.Trim(username, " ") == "" || strings.Trim(passwd, " ") == "" {
		leftLoginMsg(c, "User name or password cannot be empty")
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Check for username and password match, usually from a database
	if !ValidateUser([]byte(username), []byte(passwd)) {
		leftLoginMsg(c, "Username/password does not match")
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Save the username in the session
	session.Set(userkey, username)

	if err := session.Save(); err != nil {
		leftLoginMsg(c, "Failed to save session data")
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	host = &c.Request.Host
	c.Redirect(http.StatusSeeOther, "/")
}

func logout(c *gin.Context) {
	session := sessions.Default(c)

	user := session.Get(userkey)
	if user != nil {
		session.Delete(userkey)
		session.Save()
	}

	leftLoginMsg(c, "Welcome to WiTTY")
	c.Redirect(http.StatusFound, "/login")
}

// AuthRequired is a simple middleware to check the session
func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(userkey)

	if user == nil {
		leftLoginMsg(c, "Not authorized, login first")
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		c.Abort()
		return
	}

	c.Next()
}

func loginPage(c *gin.Context) {
	session := sessions.Default(c)
	msg := session.Get(loginKey)

	if msg == nil {
		msg = "Login first"
	}

	c.HTML(http.StatusOK, "login.html", gin.H{"msg": msg})
}
