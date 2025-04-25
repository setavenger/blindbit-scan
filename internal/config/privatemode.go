package config

var (
	UnlockChan = make(chan string)
)

func init() {
	UnlockChan = make(chan string)
}

func Unlock() {
	close(UnlockChan)
}
