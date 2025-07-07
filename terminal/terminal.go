package terminal

import (
	"fmt"
	"golang.org/x/text/encoding/simplifiedchinese"
	"io"
	"os/exec"
	"sync"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

type Terminal struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	stderr     io.ReadCloser
	outputChan chan []byte
	closeChan  chan struct{}
	closeOnce  sync.Once
	wg         sync.WaitGroup
	closed     bool
	mutex      sync.Mutex
	encoding   encoding.Encoding
}

func NewTerminal(shell string) (*Terminal, error) {
	cmd := exec.Command(shell, "-NoLogo", "-NoProfile", "-Command", "-")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe failed: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe failed: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe failed: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("command start failed: %w", err)
	}

	t := &Terminal{
		cmd:        cmd,
		stdin:      stdin,
		stdout:     stdout,
		stderr:     stderr,
		outputChan: make(chan []byte, 1024),
		closeChan:  make(chan struct{}),
	}

	err = t.SetEncoding("GBK")
	if err != nil {
		return nil, fmt.Errorf("set encoding failed: %w", err)
	}

	//_, err = t.Write([]byte("chcp 65001"))
	//if err != nil {
	//	fmt.Errorf("change encoding failed: %w", err)
	//	return nil, err
	//}
	t.wg.Add(1)
	go t.handleOutput()

	return t, nil
}

func (t *Terminal) Write(data []byte) (int, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.closed {
		return 0, io.ErrClosedPipe
	}
	return t.stdin.Write(data)
}

func (t *Terminal) Read() <-chan []byte {
	return t.outputChan
}

func (t *Terminal) Close() {
	t.closeOnce.Do(func() {
		t.mutex.Lock()
		defer t.mutex.Unlock()

		close(t.closeChan)
		t.closed = true

		t.stdin.Close()
		t.cmd.Process.Kill()
		t.cmd.Wait()

		t.wg.Wait()
		close(t.outputChan)
	})
}

func (t *Terminal) handleOutput() {
	defer t.wg.Done()

	// 使用io.MultiReader保证顺序一致性
	reader := io.MultiReader(t.stdout, t.stderr)

	for {
		select {
		case <-t.closeChan:
			return
		default:
			buf := make([]byte, 4096)

			translatedReader := transform.NewReader(reader, t.encoding.NewDecoder())
			n, err := translatedReader.Read(buf)

			if n > 0 {
				t.outputChan <- buf[:n]
			}
			if err != nil {
				if err != io.EOF {
					t.outputChan <- []byte("\r\n\x1b[31mTerminal error: " + err.Error() + "\x1b[0m")
				}
				return
			}
		}
	}
}

func (t *Terminal) ExitCode() int {
	if t.cmd.ProcessState == nil {
		return -1
	}
	return t.cmd.ProcessState.ExitCode()
}

func (t *Terminal) SetEncoding(encName string) error {
	switch encName {
	case "UTF-8":
		t.encoding = encoding.Nop
	case "GBK":
		t.encoding = simplifiedchinese.GBK
	case "GB2312":
		t.encoding = simplifiedchinese.GB18030
	case "Latin1":
		t.encoding = charmap.Windows1252
	default:
		return fmt.Errorf("不支持的编码: %s", encName)
	}
	return nil
}
