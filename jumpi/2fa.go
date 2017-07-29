package jumpi

import (
	"golang.org/x/crypto/ssh/terminal"
)

type Config interface {
	Handle(tty *terminal.Terminal) error
}

type ConfigYubikey struct {
	Username string
}

func (c *ConfigYubikey) Handle(tty *terminal.Terminal) error {
	// get next token code
	token, err := tty.ReadPassword("Enter YubiKey OTP: ")
	if err != nil {
		tty.Write([]byte("Unable to get YubiKey OTP: "))
		tty.Write([]byte(err.Error()))
		return err
	}

	tty.Write([]byte(token))

	return nil
}
