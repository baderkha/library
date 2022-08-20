package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/baderkha/library/pkg/conditional"
	"github.com/baderkha/library/pkg/controller/gin/auth/sso"
	"github.com/baderkha/library/pkg/controller/response"
	"github.com/baderkha/library/pkg/email"
	"github.com/baderkha/library/pkg/store/entity"
	"github.com/baderkha/library/pkg/store/repository"
	"github.com/badoux/checkmail"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sethvargo/go-password/password"
	passwordvalidator "github.com/wagslane/go-password-validator"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	MinEntropy     = 65
	BcryptCost     = 12
	BcryptCostWeak = 10
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

type emailVerificationObj struct {
	Email string
}

type SessionAuthGinController struct {
	CookieName                         string
	Domain                             string
	FrontEndDomain                     string
	PasswordResetURL                   string
	URLPathPrefix                      string
	AccountSessionDuration             time.Duration
	Verification_PasswordResetDuration time.Duration
	Arepo                              repository.IAccount
	SRepo                              repository.ISession
	Hrepo                              repository.IHashVerificationAccount
	Tx                                 repository.ITransaction
	MailValidation                     email.ISender
	BaseMailConfig                     email.Content
	SSOHandler                         sso.Handler
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

	ctx.SetCookie(
		c.CookieName,
		session.ID,
		int(c.AccountSessionDuration/time.Minute),
		"/",
		c.Domain,
		true,
		true,
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

func (c *SessionAuthGinController) IsLoggedIn(ctx *gin.Context) bool {
	sessionId, _ := ctx.Cookie(c.CookieName)
	if sessionId != "" {
		session, err := c.SRepo.GetById(sessionId)
		if err != nil {
			return false
		}
		return session.ID == sessionId && session.ExpiresAt.Unix() > time.Now().Unix()
	}
	return false
}

func (c *SessionAuthGinController) login(ctx *gin.Context) {
	if !c.IsLoggedIn(ctx) {
		var info loginObj
		err := ctx.BindJSON(&info)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, response.NewError(err))
			return
		}
		acc, err := c.Arepo.GetById(info.UserName)
		if err != nil || acc.IsSSO {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(errUnauthorized))
			return
		}
		if c.validatePassword(acc.Password, info.Password) != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(errUnauthorized))
			return
		}
		c.SerializeSession(acc.ID, ctx, nil)
	}
	ctx.String(http.StatusOK, "LOGGED IN")
}

// once user is login we will use our own session logic
func (c *SessionAuthGinController) loginSSO(ctx *gin.Context) {
	if !c.IsLoggedIn(ctx) {
		acc, err := c.SSOHandler.VerifyUser(ctx.Request.Header)
		if err != nil {
			fmt.Println(err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(errUnauthorized))
			return
		}

		doesAccountExist, dbAccount := c.Arepo.DoesAccountExistByEmail(acc.Email)
		if !doesAccountExist {
			acc.New()
			err := c.Arepo.Create(acc)
			if err != nil {
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.NewError(errRepo))
				return
			}
			dbAccount = acc
		}

		c.SerializeSession(dbAccount.ID, ctx, nil)
	}
	ctx.String(http.StatusOK, "LOGGED IN")
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

func (c *SessionAuthGinController) onNewAccount(acc *entity.Account) error {
	// email to the client
	err := c.sendVerificationEmail(acc, entity.HashVerificationAccountTypeVerify)
	if err != nil {
		return err
	}
	return nil
}

func (c *SessionAuthGinController) sendVerificationEmail(acc *entity.Account, Type string) error {
	uuid, _ := uuid.NewV4()

	pwdB64 := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s_%s", password.MustGenerate(64, 10, 0, false, true), uuid.String())))

	// weak hash since we'll delete it after usage
	hash := sha256.Sum256([]byte(pwdB64))

	// persist model
	hashVerificationForAccount := entity.HashVerificationAccount{
		ID:        string(hash[:]),
		AccountID: acc.ID,
		Email:     acc.Email,
		TTLExpiry: time.Now().Add(c.Verification_PasswordResetDuration),
		Type:      Type,
	}

	err := c.Hrepo.Create(&hashVerificationForAccount)
	if err != nil {
		return err
	}
	config := c.BaseMailConfig
	config.To = acc.Email
	config.ToFriendlyName = acc.Email

	switch Type {
	case entity.HashVerificationAccountTypeResetPass:
		config.Body = fmt.Sprintf(
			`<strong> Please %s with this link  <a href="%s?signature=%s"/> </strong>. 
		If You did not request either a (%s) , ignore this email.`,
			hashVerificationForAccount.Type,
			c.PasswordResetURL,
			pwdB64,
			strings.Join([]string{entity.HashVerificationAccountTypeVerify, entity.HashVerificationAccountTypeResetPass}, " , "),
		)
		break
	case entity.HashVerificationAccountTypeVerify:
		config.Body = fmt.Sprintf(
			`<strong> Please %s with this link  <a href="%s%s%s?signature=%s"/> </strong>. 
		If You did not request either a (%s) , ignore this email.`,
			hashVerificationForAccount.Type,
			c.Domain,
			c.URLPathPrefix,
			"/email/_verify",
			pwdB64,
			strings.Join([]string{entity.HashVerificationAccountTypeVerify, entity.HashVerificationAccountTypeResetPass}, " , "),
		)
		break
	default:
		return errUnauthorized
	}

	return c.MailValidation.SendHTMLEmail(&c.BaseMailConfig)
}

func (c *SessionAuthGinController) sendVerificationEmailRest(ctx *gin.Context) {
	var e emailVerificationObj
	err := ctx.BindJSON(&e)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.NewError(err))
		return
	}
	isExist, acc := c.Arepo.DoesAccountExistByEmail(e.Email)
	if isExist {
		_ = c.sendVerificationEmail(acc, entity.HashVerificationAccountTypeVerify)
	}
	ctx.JSON(http.StatusOK, "ok")
}

func (c *SessionAuthGinController) verifyValidationHashAndGrabAccount(hash string) (acc *entity.Account, isValid bool) {
	hashRes, err := c.Hrepo.GetById(hash)

	isValid = err == nil && !hashRes.HasBeenUsed && hashRes.TTLExpiry.Unix() > time.Now().Unix()

	if isValid {
		acc, err = c.Arepo.GetById(hashRes.AccountID)
		hashRes.HasBeenUsed = true
		_ = c.Hrepo.Update(hashRes)
	}
	return acc, isValid && err == nil

}

func (c *SessionAuthGinController) verifyValidationLink(ctx *gin.Context) {

	var (
		errUpdateDB error
	)

	signature := ctx.Query("signature")
	redirectClient := ctx.Query("redirect") == "TRUE"

	if signature == "" {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(errUnauthorized))
		return
	}

	hash := sha256.Sum256([]byte(signature))
	signature = string(hash[:])

	acc, isValid := c.verifyValidationHashAndGrabAccount(signature)
	if !isValid {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(errUnauthorized))
		return
	}

	acc.IsVerified = true // account is verified and enabled for use

	errUpdateDB = c.Arepo.Update(acc)

	if errUpdateDB != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.NewError(errRepo))
		return
	}

	if redirectClient {
		ctx.Redirect(http.StatusTemporaryRedirect, c.FrontEndDomain)
		return
	}

	ctx.JSON(http.StatusOK, "email_verified")
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
	acc.IsVerified = false

	tx := c.Tx.Begin()

	err = c.Arepo.Create(&acc)
	if err != nil {
		tx.RollBack()
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.NewError(err))
		return
	}

	err = c.onNewAccount(&acc)
	if err != nil {
		tx.RollBack()
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

func (c *SessionAuthGinController) updateAccountPassword(newpassword string, acc *entity.Account, ctx *gin.Context) (err error) {

	err = c.IsPasswordSafe(newpassword)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.NewError(err))
		return
	}
	acc.Password = newpassword
	acc = c.genHashAuth(acc)
	err = c.Arepo.Update(acc)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, response.NewError(errRepo))
		return
	}
	return nil
}

func (c *SessionAuthGinController) changePasswordWithAuthToken(ctx *gin.Context) {
	type pwd struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_pwd" binding:"required"`
	}

	var p pwd

	err := ctx.ShouldBindJSON(&p)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, response.NewError(err))
		return
	}

	acc, isValid := c.verifyValidationHashAndGrabAccount(p.Token)
	if !isValid {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(errUnauthorized))
		return
	}

	c.updateAccountPassword(p.NewPassword, acc, ctx)

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
	c.updateAccountPassword(p.NewPassword, acc, ctx)

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
		grp.POST("login/sso", c.loginSSO)
		grp.POST("logout", c.validate, c.logout)
		grp.POST("password-strength/check", c.pwdStrength)
		grp.POST("/email/_send_verification", c.sendVerificationEmailRest)
		grp.POST("/email/_verify", c.verifyValidationLink)

		grp.POST("accounts", c.newAccount)
		grp.PUT("accounts/_password/reset", c.validate, c.changePassword)
		grp.PUT("accounts/_password/forgotten", c.changePasswordWithAuthToken)
		grp.DELETE("accounts", c.validate, c.deleteAccount)
		grp.GET("accounts/:id", c.getAccountInfo)
	}
	return e
}

// SessionConfig : Sesion Auth Config
type SessionConfig struct {
	// DB : gorm data base this is required
	DB *gorm.DB
	// BasePathRoute : the base uri
	BasePathRoute string
	// CookieName : name of the cooke this is required
	CookieName string
	// LoginExpiryTime : the login expiry time default is 15m
	LoginExpiryTime time.Duration
	// Domain : host domain
	Domain string
	// SSOConfig : sso configuration , if you want /login/sso route working
	SSOConfig sso.Config
	// FrontEndDomain : front end domain for default redirect
	FrontEndDomain string
	// PasswordResetURLFull : password reset forgotten page
	PasswordResetURLFull string
}

func NewGinSessionAuthGorm(s *SessionConfig) *SessionAuthGinController {

	return &SessionAuthGinController{
		CookieName:             s.CookieName,
		URLPathPrefix:          s.BasePathRoute,
		AccountSessionDuration: s.LoginExpiryTime,
		Domain:                 s.Domain,
		Arepo: &repository.AccountGorm{
			CrudGorm: repository.CrudGorm[entity.Account]{
				DB:         s.DB,
				PrimaryKey: "id",
				Table:      (&entity.Account{}).TableName(),
				Parser:     nil,
				Sorter:     nil,
			},
		},
		SRepo: &repository.SessionGorm{
			DB:         s.DB,
			PrimaryKey: "id",
			Table:      (&entity.Session{}).TableName(),
			Parser:     nil,
			Sorter:     nil,
		},
		SSOHandler:       sso.New(s.SSOConfig),
		FrontEndDomain:   s.FrontEndDomain,
		PasswordResetURL: s.PasswordResetURLFull,
	}
}
