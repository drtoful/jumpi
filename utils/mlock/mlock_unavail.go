// +build android nacl netbsd plan9 windows

package mlock

func init() {
	supported = false
}

func lockMemory() error {
	return nil
}
