package auth

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/baderkha/library/pkg/conditional"
	"github.com/baderkha/library/pkg/store/entity"
	"github.com/baderkha/library/pkg/store/repository"
	"github.com/badoux/checkmail"
	"github.com/gin-gonic/gin"
	passwordvalidator "github.com/wagslane/go-password-validator"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	MinEntropy = 65
	BcryptCost = 12
)

var (
	regexUserId, _ = regexp.Compile("^[a-zA-Z0-9-_]+$")
)

// error block
var (
	errorAccountUserNameAlreadyExists = errors.New("this account user name / email already exists")
	errorAccountMustBeValid           = errors.New("account must be alphanumeric with no spaces or special characters (you can use underscore or dashes)")
	errUnauthorized                   = errors.New("Unauthorized")
)

type loginObj struct {
	UserName string `json:"user_name" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type SessionAuthGinController struct {
	CookieName             string
	Domain                 string
	URLPathPrefix          string
	AccountSessionDuration time.Duration
	Arepo                  repository.IAccount
	SRepo                  repository.ISession
}

func (c *SessionAuthGinController) toAccount(e *entity.Account, emailRedact bool) *entity.Account {
	e.Password = "*** REDACTED ***"
	e.Email = conditional.Ternary(emailRedact, "*** REDACTED ***", e.Email)
	return e
}

func (c *SessionAuthGinController) SerializeSession(accountID string, ctx *gin.Context, existingSession *entity.Session) {

	session := &entity.Session{}
	if existingSession != nil {
		session = existingSession
		session.ExpiresAt = time.Now().Add(c.AccountSessionDuration)

		err := c.SRepo.Update(session)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

	} else {
		session.New()
		session.ExpiresAt = time.Now().Add(c.AccountSessionDuration)
		session.AccountID = accountID

		err := c.SRepo.Create(session)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}

	ctx.SetCookie(
		c.CookieName,
		session.ID,
		int(c.AccountSessionDuration/time.Second),
		"/",
		c.Domain,
		true,
		true,
	)

}

func (c *SessionAuthGinController) login(ctx *gin.Context) {
	var info loginObj
	err := ctx.BindJSON(&info)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	acc, err := c.Arepo.GetById(info.UserName)
	if err != nil {
		ctx.AbortWithError(http.StatusUnauthorized, errUnauthorized)
		return
	}
	if c.validatePassword(acc.Password, info.Password) != nil {
		ctx.AbortWithError(http.StatusUnauthorized, errUnauthorized)
		return
	}
	c.SerializeSession(acc.ID, ctx, nil)
}

func (c *SessionAuthGinController) IsPasswordSafe(p string) error {
	return passwordvalidator.Validate(p, MinEntropy)
}

func (s *SessionAuthGinController) sendAccountInfo(ctx *gin.Context, accID string) {
	ctx.Set("account_id", accID)
}

func (C *SessionAuthGinController) logout(ctx *gin.Context) {

}

func (c *SessionAuthGinController) validateAccount(acc *entity.Account) error {
	err := c.IsPasswordSafe(acc.Password)
	if err != nil {
		return fmt.Errorf("validation error : password  : %v", err)
	}
	err = checkmail.ValidateFormat(acc.Email)
	if err != nil {
		return fmt.Errorf("validation error : email : %v", err)
	}
	if !regexUserId.Match([]byte(acc.ID)) {
		return fmt.Errorf("validation error : account id : %v", errorAccountMustBeValid)
	}
	return nil
}

func (c *SessionAuthGinController) genHashAuth(e *entity.Account) *entity.Account {
	pByte, _ := bcrypt.GenerateFromPassword([]byte(e.Password), BcryptCost)
	e.Password = string(pByte)
	return e
}

func (c *SessionAuthGinController) validatePassword(hash string, inputPassword string) error {
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(inputPassword)) != nil {
		return errUnauthorized
	}
	return nil
}

func (c *SessionAuthGinController) newAccount(ctx *gin.Context) {
	var acc entity.Account

	err := ctx.BindJSON(&acc)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	err = c.validateAccount(&acc)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if c.Arepo.DoesAccountExist(acc.ID, acc.Email) {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	acc = *(c.genHashAuth(&acc))
	err = c.Arepo.Create(&acc)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusCreated, c.toAccount(&acc, false))

}

func (c *SessionAuthGinController) getAccountInfo(ctx *gin.Context) {

}

func (c *SessionAuthGinController) updateAccount(ctx *gin.Context) {

}

func (c *SessionAuthGinController) deleteAccount(ctx *gin.Context) {

}

func (c *SessionAuthGinController) validate(ctx *gin.Context) {
	sessionId, _ := ctx.Cookie(c.CookieName)
	if sessionId != "" {
		session, err := c.SRepo.GetById(sessionId)
		if err != nil {
			ctx.AbortWithError(http.StatusUnauthorized, errUnauthorized)
			return
		}
		// send the account id
		c.sendAccountInfo(ctx, session.AccountID)
		// refresh cookie
		c.SerializeSession(session.AccountID, ctx, session)
		ctx.Next()
		return
	}
	ctx.AbortWithError(http.StatusUnauthorized, errUnauthorized)
	return
}

func (c *SessionAuthGinController) GetAuthMiddleWare() gin.HandlerFunc {
	return c.validate
}

func (c *SessionAuthGinController) ApplyRoutes(e *gin.Engine) *gin.Engine {
	grp := e.Group(c.URLPathPrefix)
	{
		grp.POST("login", c.login)
		grp.POST("logout", c.validate, c.logout)
		grp.POST("password-strength/check")

		grp.POST("accounts", c.newAccount)
		grp.PATCH("accounts", c.validate, c.updateAccount)
		grp.DELETE("accounts", c.validate, c.deleteAccount)
		grp.GET("accounts/:id", c.validate, c.getAccountInfo)
	}
	return e
}

func NewGinSessionAuthGorm(db *gorm.DB, basePathRoute string, cookieName string, loginExpiryTime time.Duration) *SessionAuthGinController {
	return &SessionAuthGinController{
		URLPathPrefix:          basePathRoute,
		AccountSessionDuration: loginExpiryTime,
		Arepo: &repository.AccountGorm{
			DB:         db,
			PrimaryKey: "id",
			Table:      (&entity.Account{}).TableName(),
			Parser:     nil,
			Sorter:     nil,
		},
		SRepo: &repository.SessionGorm{
			DB:         db,
			PrimaryKey: "id",
			Table:      (&entity.Session{}).TableName(),
			Parser:     nil,
			Sorter:     nil,
		},
	}
}
