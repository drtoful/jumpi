package jumpi

import (
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/GeertJohan/yubigo"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	ConfigYubikeyAPI = "config:yubikey_api"
)

type AuthenticationHandler interface {
	Verify(username, token string) bool
	Setup(username string, tty *terminal.Terminal) error
}

type yubikeyHandler struct {
	yubiAuth *yubigo.YubiAuth
	store    *Store
	lock     *sync.Mutex
}

func (h *yubikeyHandler) Verify(username, token string) bool {
	h.lock.Lock()
	defer h.lock.Unlock()

	if h.yubiAuth == nil {
		log.Println("yubikey_auth: yubikey authentication is not correctly configured, but user wants to use it")
		return false
	}

	val, err := h.store.Get(BucketUsersConfig, username+"~2fa~config")
	if err != nil || len(val) == 0 {
		log.Printf("yubikey_auth: unable to verify token for '%s': unable to find/load configuration\n", username)
		return false
	}

	_, ok, err := h.yubiAuth.Verify(token)
	return ok && err == nil
}

func (h *yubikeyHandler) Setup(username string, tty *terminal.Terminal) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	if h.yubiAuth == nil {
		tty.Write([]byte("Yubikey Authentication unavailable"))
		return errors.New("yubikey authetnication is not correctly configured, but user wants to use it")
	}

	// get next token code
	token, err := tty.ReadPassword("Enter YubiKey OTP: ")
	if err != nil {
		tty.Write([]byte("Unable to get YubiKey OTP: "))
		tty.Write([]byte(err.Error()))
		return err
	}

	// verify yubikey otp
	_, ok, err := h.yubiAuth.Verify(token)
	if err == nil && ok {
		// save yubikey ID into store
		if err := h.store.Set(BucketUsersConfig, username+"~2fa~config", []byte(token[:12])); err != nil {
			return err
		}

		if err := h.store.Set(BucketUsersConfig, username+"~2fa~kind", []byte("yubikey")); err != nil {
			h.store.Delete(BucketUsersConfig, username+"~2fa~config")
			tty.Write([]byte("unable to activate 'yubikey' two factor authentication\n"))
			return err
		}

		tty.Write([]byte("successfully activated 'yubikey' two factor authentication\n"))
		return nil
	}

	tty.Write([]byte("Unable to verify YubiKey OTP: "))
	tty.Write([]byte(err.Error()))
	return err
}

func startYubikeyAuth(store *Store) AuthenticationHandler {
	handler := &yubikeyHandler{
		store: store,
		lock:  &sync.Mutex{},
	}

	go func() {
		// wait for specific config to become available
		var secret Secret
		secret.ID = ConfigYubikeyAPI

		for {
			err := secret.Load(store)
			if err == nil {
				break
			}

			time.Sleep(time.Second)
		}
		log.Println("yubikey_auth: found 'config:yubikey_api' key in store, starting service")

		if secret.Type != Password {
			log.Println("yubikey_auth: secret has not correct type (expected 'password')")
			return
		}

		svalue, ok := secret.Secret.(string)
		if !ok {
			log.Println("yubikey_auth: unable to convert secret to string")
			return
		}

		pieces := strings.SplitN(svalue, ":", 2)
		if len(pieces) != 2 {
			log.Println("yubikey_auth: 'config:yubikey_api' in wrong format, needs to be \"client_id:api_key\"")
			return
		}

		auth, err := yubigo.NewYubiAuth(pieces[0], pieces[1])
		if err != nil {
			log.Printf("yubikey_auth: unable to create Yubikey authentication service: %s\n", err.Error())
			return
		}

		handler.lock.Lock()
		handler.yubiAuth = auth
		handler.lock.Unlock()

		log.Println("yubikey_auth: authentication service successfully started")
	}()

	return handler
}

type TwoFactorAuth struct {
	store    *Store
	services map[string]AuthenticationHandler
	lock     *sync.Mutex
}

func StartTwoFactorAuthServer(store *Store) (*TwoFactorAuth, error) {
	result := &TwoFactorAuth{
		store:    store,
		services: make(map[string]AuthenticationHandler),
		lock:     &sync.Mutex{},
	}

	go func() {
		// wait for store to unlock
		for {
			if !store.IsLocked() {
				break
			}
			time.Sleep(time.Second)
		}

		log.Println("starting two factor authentication services")
		result.lock.Lock()
		defer result.lock.Unlock()
		result.services["yubikey"] = startYubikeyAuth(store)
	}()

	return result, nil
}

func (h *TwoFactorAuth) Verify(username, token string) bool {
	kind, has := h.HasTwoFactor(username)
	if !has {
		return false
	}

	h.lock.Lock()
	val, ok := h.services[kind]
	h.lock.Unlock()

	if !ok {
		return false
	}

	return val.Verify(username, token)
}

func (h *TwoFactorAuth) Setup(username, kind string, tty *terminal.Terminal) error {
	h.lock.Lock()
	val, ok := h.services[kind]
	h.lock.Unlock()

	if !ok {
		return errors.New("unknown 2fa kind '" + kind + "' for setup")
	}

	return val.Setup(username, tty)
}

func (h *TwoFactorAuth) HasTwoFactor(username string) (string, bool) {
	val, err := h.store.Get(BucketUsersConfig, username+"~2fa~kind")
	if err == nil && len(val) > 0 {
		return string(val), true
	}

	return "", false
}
