package mlock

var supported bool

func Supported() bool {
	return supported
}

func LockMemory() error {
	return lockMemory()
}
