package main

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/gotd/contrib/bg"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"net/http"
	"regexp"
	"sync"
	"time"
)

var iranMobileRegexp = regexp.MustCompile(`^(0|\+98)\d{10}$`)

type (
	GetCodeRequest struct {
		Phone string `json:"phone"`
	}
	VerifyCodeRequest struct {
		UserKey  string `json:"user_key"`
		Code     string `json:"code"`
		Password string `json:"password"`
	}
	DeleteAccountRequest struct {
		UserKey string `json:"user_key"`
	}

	UserState struct {
		telegramClient *telegram.Client
		passwordChan   chan<- string
		codeChan       chan<- string
		clientClose    bg.StopFunc
		chanCancel     context.CancelFunc
		statusChan     chan error
	}

	AuthHandler struct {
		// userStateMap saves user state between multiple http handlers.
		// it's not very practical but it works!
		// possible leak: user state is only deleted after a user invokes VerifyCode(). so invoking only GetCode()
		// TODO: use in memory cache with expiration
		userStateMap *sync.Map
		store        *Store
		config       *Config
	}

	DeleteHandler struct {
		store  *Store
		config *Config
	}
)

func NewEchoServer(c *Config) *echo.Echo {
	e := echo.New()
	e.Debug = true

	e.Server.ReadTimeout = c.ServerTimeout
	e.Server.WriteTimeout = c.ServerTimeout

	// recover from panics
	//e.Use(middleware.Recover())

	// use header: api-key: SECRET
	e.Use(middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		KeyLookup: "header:api-key",
		Validator: func(auth string, context echo.Context) (bool, error) {
			return auth == c.APIKey, nil
		},
	}))

	return e
}

func RegisterEndpoints(e *echo.Echo, authHandler *AuthHandler, deleteHandler *DeleteHandler) {
	e.POST("/get_code", authHandler.GetCode)
	e.POST("/verify_code", authHandler.VerifyCode)
	e.POST("/delete", deleteHandler.DeleteAccount)
}

// GetCode will do:
// - initialize a client for this user.
// - start Authentication flow.
// - request verification code.
// after that, the authentication flow blocks at Code() function.
// which will be released once it's value is received in a channel from
// Verify() handler.
func (l *AuthHandler) GetCode(c echo.Context) error {
	var req GetCodeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// temporarily disable phone number validation
	//if !iranMobileRegexp.MatchString(req.Phone) {
	//	return echo.NewHTTPError(http.StatusBadRequest, "phone number should be in format +989111111111")
	//}

	userKey := uuid.Must(uuid.NewRandom()).String()
	sessionSaver := &badgerSessionSaver{
		userKey: userKey,
		db:      l.store.db,
	}

	client := telegram.NewClient(l.config.TelegramAppID, l.config.TelegramAppHash, telegram.Options{
		SessionStorage: sessionSaver,
	})

	// we only wait 120 seconds for next request with code && password
	// also telegram connection will be closed after this duration
	authenticatorCtx, cancelFunc := context.WithTimeout(context.Background(), 120*time.Second)

	stopFunc, err := bg.Connect(client, bg.WithContext(authenticatorCtx))
	if err != nil {
		cancelFunc()
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	passwordChan, codeChan, statusChan := make(chan string, 1), make(chan string, 1), make(chan error, 1)
	authenticator := &HTTPAuthenticator{
		phoneNumber:  req.Phone,
		passwordChan: passwordChan,
		codeChan:     codeChan,
	}

	flow := auth.NewFlow(
		authenticator,
		auth.SendCodeOptions{},
	)

	userState := &UserState{
		telegramClient: client,
		passwordChan:   passwordChan,
		codeChan:       codeChan,
		clientClose:    stopFunc,
		chanCancel:     cancelFunc,
		statusChan:     statusChan,
	}

	l.userStateMap.Store(userKey, userState)

	go func() {
		if err = client.Auth().IfNecessary(authenticatorCtx, flow); err != nil {
			logrus.Errorf("failed to start authentication: %s", err.Error())
			cancelFunc()
			if err = stopFunc(); err != nil {
				logrus.Errorf("failed to stop background client: %s", err.Error())
			}
		}

		statusChan <- err
	}()

	return c.JSON(http.StatusOK, map[string]string{
		"message": "authentication mechanism started, you have 120 seconds to send verification code and 2FA password",
		"key":     userKey,
	})
}

// VerifyCode will try and find a login session with `user_key`, if successful,
// it will continue authentication process by writing code and password in UserState channel.
func (l *AuthHandler) VerifyCode(c echo.Context) error {
	var req VerifyCodeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	value, loaded := l.userStateMap.LoadAndDelete(req.UserKey)
	if !loaded {
		return echo.NewHTTPError(http.StatusNotFound, "user key not found in server")
	}

	userState, ok := value.(*UserState)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "invalid type stored in user state map")
	}

	defer userState.chanCancel()
	defer userState.clientClose()

	userState.passwordChan <- req.Password
	userState.codeChan <- req.Code

	statusErr := <-userState.statusChan

	if statusErr != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, statusErr)
	}

	if req.Password != "" {
		if err := l.store.StoreUserPassword(req.UserKey, req.Password); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
	}

	return c.String(http.StatusOK, "authentication finished. use user_key for further calls")
}

func (d *DeleteHandler) DeleteAccount(c echo.Context) error {
	var req DeleteAccountRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	password, err := d.store.GetUserPassword(req.UserKey)
	if err != nil && err != ErrRecordNotFound {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	sessionSaver := &badgerSessionSaver{
		userKey: req.UserKey,
		db:      d.store.db,
	}

	client := telegram.NewClient(d.config.TelegramAppID, d.config.TelegramAppHash, telegram.Options{
		SessionStorage: sessionSaver,
	})

	err = client.Run(c.Request().Context(), func(ctx context.Context) error {
		deleteAccountRequest := &tg.AccountDeleteAccountRequest{}

		if password != "" {
			pass, err := client.API().AccountGetPassword(context.Background())

			passwordSRP, err := auth.PasswordHash([]byte(password), pass.SRPID, pass.SRPB, pass.SecureRandom, pass.CurrentAlgo)
			if err != nil {
				return err
			}

			deleteAccountRequest.Password = passwordSRP
		}

		success, err := client.API().AccountDeleteAccount(ctx, deleteAccountRequest)
		if err != nil {
			return err
		}

		if !success {
			return errors.New("delete account not successful")
		}

		return nil
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	return c.String(http.StatusOK, "Your Account was deleted successfully :)")
}
