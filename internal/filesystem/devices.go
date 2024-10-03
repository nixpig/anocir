package filesystem

type Device struct {
	Source string
	Target string
	Fstype string
	Flags  uintptr
	Data   string
}

var (
	AllDevices           = "a"
	BlockDevice          = "b"
	CharDevice           = "c"
	UnbufferedCharDevice = "u"
	FifoDevice           = "p"
)
