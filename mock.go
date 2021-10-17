package mockserialport

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type (
	// Mock contains options and the serial port
	Mock struct {
		Options *Options
		Port    Port
	}

	// Mock options
	Options struct {
		InputFile  string // device file for your program to open for read and write
		OutputFile string // device file for this program to open for read and write
		PidFile    string // file to store the process id of socat
		SocatPath  string // path to the socat executable, defaults to "socat"
		BaudRate   int    // baud rate (1200/2400/4800/9600/19200/38400/57600/115200)
		ExtraOpts  string // extra options of socat
		Verbose    bool   // whether to print log to stderr

		// Function is called to open serial port
		Open func(path string, baudrate int) (Port, error)

		// Function is called when new bytes is read. Return any
		// unprocessed bytes for later use
		Process func(*Mock, []byte) []byte
	}

	// Serial port interface
	Port interface {
		Read(p []byte) (n int, err error)
		Write(p []byte) (n int, err error)
	}

	// See flag.FlagSet
	Flag interface {
		StringVar(*string, string, string, string)
		IntVar(*int, string, int, string)
	}
)

// Create a new Mock.
func New(opts *Options) *Mock {
	return &Mock{
		Options: opts,
	}
}

// Start() executes StartSocat() and Read(), which starts a socat process and
// read data from the virtual serial port.
func (m *Mock) Start() (err error) {
	if err = m.StartSocat(); err != nil {
		return
	}
	i := 0
	for i < 10 {
		_, err = os.Stat(m.Options.OutputFile)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
		i++
	}
	if err != nil {
		return
	}
	err = m.Read()
	return
}

// Kill previous socat process (if any) and then start a socat process and
// write process ID to the pid file.
func (m *Mock) StartSocat() error {
	os.Remove(m.Options.OutputFile)
	pidStr, _ := os.ReadFile(m.Options.PidFile)
	pid, _ := strconv.Atoi(string(pidStr))
	if pid > 0 {
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
			if !strings.Contains(err.Error(), "no such process") {
				if m.Options.Verbose {
					log.Println("error killing existing socat:", err)
				}
			}
		} else {
			if m.Options.Verbose {
				log.Printf("successfully killed existing socat pid=%d", pid)
			}
		}
	}
	socatPath := m.Options.SocatPath
	if socatPath == "" {
		socatPath = "socat"
	}
	cmd := exec.Command(socatPath, m.Options.SocatCommandArgs()...)
	if m.Options.Verbose {
		log.Println("running", cmd)
	}
	if err := cmd.Start(); err != nil {
		if m.Options.Verbose {
			log.Println("error:", err)
		}
		return err
	}
	if m.Options.Verbose {
		log.Printf("started socat pid=%d", cmd.Process.Pid)
	}
	return os.WriteFile(m.Options.PidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0666)
}

// Read and process data from the virtual serial port.
func (m *Mock) Read() error {
	port, err := m.Options.Open(m.Options.OutputFile, m.Options.BaudRate)
	if err != nil {
		return err
	}
	m.Port = port
	if m.Options.Verbose {
		log.Println("reading data from", m.Options.OutputFile)
	}
	buf := make([]byte, 100)
	data := []byte{}
	for {
		n, err := port.Read(buf)
		if err != nil {
			if m.Options.Verbose {
				log.Println("error:", err)
			}
			return err
		}
		if n == 0 {
			break
		}
		buffer := buf[:n]
		if m.Options.Process == nil {
			if m.Options.Verbose {
				log.Printf("<= received % X", buffer)
			}
		} else {
			data = append(data, buffer...)
			data = m.Options.Process(m, data)
		}
	}
	return nil
}

// Write data to the virtual serial port.
func (m *Mock) Write(b []byte) error {
	_, err := m.Port.Write(b)
	if err != nil {
		if m.Options.Verbose {
			log.Println("error:", err)
		}
		return err
	}
	if m.Options.Verbose {
		log.Printf("=> sent     % X", b)
	}
	return nil
}

// Helper method to set up flags for flag set.
func (opts *Options) SetFlags(flag Flag) {
	flag.StringVar(&opts.InputFile, "i", opts.InputFile, "input file")
	flag.StringVar(&opts.OutputFile, "o", opts.OutputFile, "output file")
	flag.StringVar(&opts.PidFile, "pid", opts.PidFile, "pid of socat")
	flag.StringVar(&opts.SocatPath, "socat", opts.SocatPath, "path of socat executable")
	flag.IntVar(&opts.BaudRate, "baudrate", opts.BaudRate, "baud rate")
	flag.StringVar(&opts.ExtraOpts, "opts", opts.ExtraOpts, "extra options for socat")
}

// Return command line arguments of socat.
func (opts *Options) SocatCommandArgs() []string {
	extraOpts := opts.ExtraOpts
	if extraOpts != "" && !strings.HasPrefix(extraOpts, ",") {
		extraOpts = "," + extraOpts
	}
	return []string{
		fmt.Sprintf("pty,raw,echo=0,ispeed=%d,ospeed=%d,link=%s%s", opts.BaudRate, opts.BaudRate, opts.InputFile, extraOpts),
		fmt.Sprintf("pty,raw,echo=0,ispeed=%d,ospeed=%d,link=%s%s", opts.BaudRate, opts.BaudRate, opts.OutputFile, extraOpts),
	}
}
