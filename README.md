# mockserialport

Easily create a virtual serial port command line program for test or development.

[Docs](https://pkg.go.dev/github.com/caiguanhao/mockserialport)

Depends on the `socat` executable. You must install `socat` first.

This also works on Android if you have [Android's socat](https://github.com/jakev/android-binaries/blob/master/socat).

Make sure you have permissions to create device file. For example, you can
specify socat extra opts like this: `user=1001,group=1001,mode=666`.

## Usage

```go
import "github.com/caiguanhao/mockserialport"
import "go.bug.st/serial"

opts := &mockserialport.Options{
	InputFile:  "ttyIN",
	OutputFile: "ttyOUT",
	PidFile:    "socat.pid",
	SocatPath:  "/usr/local/bin/socat",
	BaudRate:   57600,
	ExtraOpts:  "",
	Verbose:    true,
	Open: func(path string, baudrate int) (mockserialport.Port, error) {
		return serial.Open(path, &serial.Mode{
			BaudRate: baudrate,
		})
	},
	Process: func(mock *mockserialport.Mock, input []byte) []byte {
		switch string(input) {
		case "hello":
			mock.Write([]byte("world"))
		case "foo":
			mock.Write([]byte("bar"))
		}
		return nil
	},
}
opts.SetFlags(flag.CommandLine)
flag.Parse()
mock := mockserialport.New(opts)
if err := mock.Start(); err != nil {
	panic(err)
}
```
