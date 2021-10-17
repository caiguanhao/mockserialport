package mockserialport_test

import (
	"testing"
	"time"

	"github.com/caiguanhao/mockserialport"
	"go.bug.st/serial"
)

const (
	inputFile  = "ttyIN"
	outputFile = "ttyOUT"
	baudrate   = 57600
)

func TestSerialPort(t *testing.T) {
	t.Log("Starting mock")
	mock := startMock()
	defer mock.Terminate()
	t.Log("Started mock")
	port, err := serial.Open(inputFile, &serial.Mode{
		BaudRate: baudrate,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer port.Close()

	cases := map[string]string{
		"hello": "world",
		"foo":   "bar",
	}
	for send, expected := range cases {
		test(t, port, send, expected)
	}
}

func test(t *testing.T, port mockserialport.Port, send, expected string) {
	n, err := port.Write([]byte(send))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Sent %v bytes: %s", n, send)

	ch := make(chan []byte)
	go func() {
		buff := make([]byte, 20)
		n, err = port.Read(buff)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Received %v bytes: %s", n, buff)
		ch <- buff[:n]
	}()

	select {
	case b := <-ch:
		actual := string(b)
		if actual == expected {
			t.Log("Received:", expected)
		} else {
			t.Errorf("it should receive %s instead of: %s", expected, actual)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout")
	}
}

func startMock() *mockserialport.Mock {
	opts := &mockserialport.Options{
		InputFile:  inputFile,
		OutputFile: outputFile,
		BaudRate:   baudrate,
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
	mock := mockserialport.New(opts)
	if err := mock.StartSocat(); err != nil {
		panic(err)
	}
	go mock.Read()
	return mock
}
