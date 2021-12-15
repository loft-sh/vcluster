package log

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/log/terminal"

	"github.com/mgutz/ansi"
)

const waitInterval = time.Millisecond * 150

var tty = terminal.SetupTTY(os.Stdin, os.Stdout)

type loadingText struct {
	Stream         io.Writer
	Message        string
	StartTimestamp int64

	loadingRune int
	isShown     bool
	stopChan    chan bool
}

func (l *loadingText) Start() {
	l.isShown = false
	l.StartTimestamp = time.Now().UnixNano()

	if l.stopChan == nil {
		l.stopChan = make(chan bool)
	}

	go func() {
		l.render()

		for {
			select {
			case <-l.stopChan:
				return
			case <-time.After(waitInterval):
				l.render()
			}
		}
	}()
}

func (l *loadingText) getLoadingChar() string {
	var loadingChar string
	var max int

	if runtime.GOOS == "darwin" {
		switch l.loadingRune {
		case 0:
			loadingChar = "⠋"
		case 1:
			loadingChar = "⠙"
		case 2:
			loadingChar = "⠹"
		case 3:
			loadingChar = "⠸"
		case 4:
			loadingChar = "⠼"
		case 5:
			loadingChar = "⠴"
		case 6:
			loadingChar = "⠦"
		case 7:
			loadingChar = "⠧"
		case 8:
			loadingChar = "⠇"
		case 9:
			loadingChar = "⠏"
		}

		max = 9
	} else {
		switch l.loadingRune {
		case 0:
			loadingChar = "|"
		case 1:
			loadingChar = "/"
		case 2:
			loadingChar = "-"
		case 3:
			loadingChar = "\\"
		}

		max = 3
	}

	l.loadingRune++

	if l.loadingRune > max {
		l.loadingRune = 0
	}

	return loadingChar
}

func (l *loadingText) render() {
	if !l.isShown {
		l.isShown = true
	} else {
		_, _ = l.Stream.Write([]byte("\r"))
	}
	messagePrefix := []byte("[wait] ")

	_, _ = l.Stream.Write([]byte(ansi.Color(string(messagePrefix), "cyan+b")))

	timeElapsed := fmt.Sprintf("%d", (time.Now().UnixNano()-l.StartTimestamp)/int64(time.Second))
	message := []byte(l.getLoadingChar() + " " + l.Message)
	messageSuffix := " (" + timeElapsed + "s)"
	prefixLength := len(messagePrefix)
	suffixLength := len(messageSuffix)

	terminalSize := tty.GetSize()
	if terminalSize != nil && uint16(prefixLength+len(message)+suffixLength) > terminalSize.Width {
		dots := []byte("...")

		maxMessageLength := int(terminalSize.Width) - (prefixLength + suffixLength + len(dots))
		if maxMessageLength > 0 {
			message = append(message[:maxMessageLength], dots...)
		}
	}

	message = append(message, messageSuffix...)
	_, _ = l.Stream.Write(message)
}

func (l *loadingText) Stop() {
	l.stopChan <- true
	_, _ = l.Stream.Write([]byte("\r"))

	messageLength := len(l.Message) + 20

	for i := 0; i < messageLength; i++ {
		_, _ = l.Stream.Write([]byte(" "))
	}

	_, _ = l.Stream.Write([]byte("\r"))
}
