package web

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	userkey = "user"
)

func login(c *gin.Context) {
	session := sessions.Default(c)

	username := c.PostForm("username")
	passwd := c.PostForm("passwd")

	// Validate form input
	if strings.Trim(username, " ") == "" || strings.Trim(passwd, " ") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username/password can't be empty"})
		return
	}

	// Check for username and password match, usually from a database
	if username != "hello" || passwd != "world" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
		return
	}

	// Save the username in the session
	session.Set(userkey, username) // In real world usage you'd set this to the users ID

	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}

	host = &c.Request.Host

	c.Redirect(http.StatusSeeOther, "/")
}

func logout(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(userkey)
	if user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session token"})
		return
	}
	session.Delete(userkey)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	c.Redirect(http.StatusFound, "/login")
}

// AuthRequired is a simple middleware to check the session
func AuthRequired(c *gin.Context) {
	if (c.Request.URL.String() == "/login") ||
		strings.HasPrefix(c.Request.URL.String(), "/assets") {
		c.Next()
		return
	}

	session := sessions.Default(c)
	user := session.Get(userkey)
	if user == nil {
		// Abort the request with the appropriate error code
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		c.Abort()
		return
	}
	// Continue down the chain to handler etc
	c.Next()
}

func loginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{})
}
