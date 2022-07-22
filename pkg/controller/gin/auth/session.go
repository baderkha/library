package auth

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/baderkha/library/pkg/conditional"
	"github.com/baderkha/library/pkg/controller/response"
	"github.com/baderkha/library/pkg/store/entity"
	"github.com/baderkha/library/pkg/store/repository"
	"github.com/badoux/checkmail"
	"github.com/davecgh/go-spew/spew"
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
	errorNotFoundAccount              = errors.New("account not found")
	errRepo                           = errors.New("could not transact with repository")
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
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.NewError(err))
			return
		}

	} else {
		session.New()
		session.ExpiresAt = time.Now().Add(c.AccountSessionDuration)
		session.AccountID = accountID

		err := c.SRepo.Create(session)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.NewError(err))
			return
		}
	}
	spew.Dump("writing cookie")
	spew.Dump(c.CookieName)
	spew.Dump(session.ID)
	spew.Dump(os.Getenv("IS_LOCAL"))
	ctx.SetCookie(
		c.CookieName,
		session.ID,
		60*60*24,
		"/",
		c.Domain,
		conditional.Ternary(os.Getenv("IS_LOCAL") == "TRUE", false, true),
		conditional.Ternary(os.Getenv("IS_LOCAL") == "TRUE", false, true),
	)

}

func (c *SessionAuthGinController) DeleteCookie(ctx *gin.Context) {
	ctx.SetCookie(
		c.CookieName,
		"",
		0,
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
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.NewError(err))
		return
	}
	acc, err := c.Arepo.GetById(info.UserName)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(errUnauthorized))
		return
	}
	if c.validatePassword(acc.Password, info.Password) != nil {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(errUnauthorized))
		return
	}
	c.SerializeSession(acc.ID, ctx, nil)
	ctx.String(http.StatusOK, "logged in")
}

func (c *SessionAuthGinController) IsPasswordSafe(p string) error {
	return passwordvalidator.Validate(p, MinEntropy)
}

func (s *SessionAuthGinController) sendAccountInfo(ctx *gin.Context, accID, sesId string) {
	ctx.Set("account_id", accID)
	ctx.Set("session_id", sesId)
}

func (c *SessionAuthGinController) logout(ctx *gin.Context) {
	sesId, isFoundSes := ctx.Get("session_id")
	if isFoundSes {
		c.SRepo.DeleteById(sesId.(string))
	}
	c.DeleteCookie(ctx)
	ctx.JSON(200, "logged out")
}

func (c *SessionAuthGinController) validateAccount(acc *entity.Account) error {
	err := c.IsPasswordSafe(acc.Password)
	if err != nil {
		return fmt.Errorf("validation error : password  : %v", response.NewError(err))
	}
	err = checkmail.ValidateFormat(acc.Email)
	if err != nil {
		return fmt.Errorf("validation error : email : %v", response.NewError(err))
	}
	if !regexUserId.Match([]byte(acc.ID)) {
		return fmt.Errorf("validation error : account id : %v", response.NewError(errorAccountMustBeValid))
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
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.NewError(err))
		return
	}

	err = c.validateAccount(&acc)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.NewError(err))
		return
	}

	if c.Arepo.DoesAccountExist(acc.ID, acc.Email) {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.NewError(err))
		return
	}
	acc = *(c.genHashAuth(&acc))
	err = c.Arepo.Create(&acc)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.NewError(err))
		return
	}

	ctx.JSON(http.StatusCreated, c.toAccount(&acc, false))

}

func (c *SessionAuthGinController) getAccountInfo(ctx *gin.Context) {
	id := ctx.Param("id")
	ac, err := c.Arepo.GetById(id)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, response.NewError(errorNotFoundAccount))
		return
	}
	ctx.JSON(200, c.toAccount(ac, true))
}

func (c *SessionAuthGinController) changePassword(ctx *gin.Context) {
	type pwd struct {
		OldPassword string `json:"old_pwd" binding:"required"`
		NewPassword string `json:"new_pwd" binding:"required"`
	}

	accId, _ := ctx.Get("account_id")
	var p pwd

	err := ctx.ShouldBindJSON(&p)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.NewError(err))
		return
	}

	acc, err := c.Arepo.GetById(accId.(string))
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.NewError(errRepo))
		return
	}

	err = c.validatePassword(acc.Password, p.OldPassword)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(errUnauthorized))
		return
	}

	err = c.IsPasswordSafe(p.NewPassword)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.NewError(err))
		return
	}
	acc.Password = p.NewPassword
	acc = c.genHashAuth(acc)
	err = c.Arepo.Update(acc)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.NewError(errRepo))
		return
	}
	ctx.JSON(http.StatusOK, c.toAccount(acc, false))

}

func (c *SessionAuthGinController) deleteAccount(ctx *gin.Context) {
	id, _ := ctx.Get("account_id")
	err := c.Arepo.DeleteById(id.(string))
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.NewError(errRepo))
		return
	}
	ctx.JSON(200, map[string]interface{}{"message": fmt.Sprintf("Deleted Account %s", id)})
}

func (c *SessionAuthGinController) validate(ctx *gin.Context) {
	sessionId, _ := ctx.Cookie(c.CookieName)
	if sessionId != "" {
		session, err := c.SRepo.GetById(sessionId)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(errUnauthorized))
			return
		}
		// send the account id
		c.sendAccountInfo(ctx, session.AccountID, sessionId)
		// refresh cookie
		c.SerializeSession(session.AccountID, ctx, session)
		ctx.Next()
		return
	}
	ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(errUnauthorized))
}

func (c *SessionAuthGinController) GetAuthMiddleWare() gin.HandlerFunc {
	return c.validate
}

func (c *SessionAuthGinController) pwdStrength(ctx *gin.Context) {
	type pwd struct {
		Password string `json:"password" binding:"required"`
	}
	var p pwd
	err := ctx.ShouldBindJSON(&p)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.NewError(err))
		return
	}
	err = c.IsPasswordSafe(p.Password)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, map[string]interface{}{"message": err.Error()})
		return
	}
	ctx.JSON(200, "ok")
}

func (c *SessionAuthGinController) ApplyRoutes(e *gin.Engine) *gin.Engine {
	grp := e.Group(c.URLPathPrefix)
	{
		grp.POST("login", c.login)
		grp.POST("logout", c.validate, c.logout)
		grp.POST("password-strength/check", c.pwdStrength)

		grp.POST("accounts", c.newAccount)
		grp.PUT("accounts/_password", c.validate, c.changePassword)
		grp.DELETE("accounts", c.validate, c.deleteAccount)
		grp.GET("accounts/:id", c.getAccountInfo)
	}
	return e
}

func NewGinSessionAuthGorm(db *gorm.DB, basePathRoute string, cookieName string, loginExpiryTime time.Duration, domain string) *SessionAuthGinController {
	return &SessionAuthGinController{
		URLPathPrefix:          basePathRoute,
		AccountSessionDuration: loginExpiryTime,
		Domain:                 domain,
		Arepo: &repository.AccountGorm{
			CrudGorm: repository.CrudGorm[entity.Account]{
				DB:         db,
				PrimaryKey: "id",
				Table:      (&entity.Account{}).TableName(),
				Parser:     nil,
				Sorter:     nil,
			},
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
