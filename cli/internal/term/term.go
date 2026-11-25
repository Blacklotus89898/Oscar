// Package term runs a command (ssh, tmux over ssh) in a pseudo-terminal and
// keeps an emulated screen, so the dashboard can draw it in one of its panes.
// Windows uses ConPTY, Linux a Unix pty; vt10x parses the output.
package term

import (
	"errors"
	"io"
	"strings"
	"sync"

	"github.com/hinshun/vt10x"
)

// ptyProc is the OS side: a running process attached to a pseudo-terminal.
type ptyProc interface {
	io.ReadWriter
	Resize(cols, rows int) error
	// Wait blocks until the process exits and reports its exit status.
	Wait() error
	// Close releases the pty; reads return an error afterwards.
	Close() error
}

// Session is one command running in a pseudo-terminal.
type Session struct {
	vt    vt10x.Terminal
	p     ptyProc
	dirty chan struct{}
	done  chan struct{}

	mu        sync.Mutex
	cols      int
	rows      int
	err       error
	closeOnce sync.Once
}

// Start runs argv in a cols×rows pseudo-terminal.
func Start(argv []string, cols, rows int) (*Session, error) {
	return StartIn(argv, "", nil, cols, rows)
}

// StartIn is Start in directory dir ("" = current) with env (KEY=value)
// added to the environment.
func StartIn(argv []string, dir string, env []string, cols, rows int) (*Session, error) {
	if len(argv) == 0 {
		return nil, errors.New("term: empty command")
	}
	cols, rows = max(cols, 10), max(rows, 3)
	p, err := startPTY(argv, dir, env, cols, rows)
	if err != nil {
		return nil, err
	}
	return newSession(p, cols, rows), nil
}

func newSession(p ptyProc, cols, rows int) *Session {
	s := &Session{p: p, cols: cols, rows: rows, dirty: make(chan struct{}, 1), done: make(chan struct{})}
	// Replies the emulator owes the program (cursor position, device
	// attributes) go back into the pty.
	s.vt = vt10x.New(vt10x.WithSize(cols, rows), vt10x.WithWriter(p))
	go s.readLoop()
	go func() {
		err := p.Wait()
		s.mu.Lock()
		s.err = err
		s.mu.Unlock()
		// ConPTY keeps the output pipe open until the console is closed.
		s.Close()
	}()
	return s
}

func (s *Session) readLoop() {
	defer close(s.done)
	buf := make([]byte, 32*1024)
	var carry int // bytes of a split UTF-8 sequence kept from the last read
	for {
		n, err := s.p.Read(buf[carry:])
		if n > 0 {
			end := carry + n
			w, _ := s.vt.Write(buf[:end])
			carry = copy(buf, buf[w:end])
			s.markDirty()
		}
		if err != nil {
			return
		}
	}
}

func (s *Session) markDirty() {
	select {
	case s.dirty <- struct{}{}:
	default:
	}
}

// Dirty receives (coalesced) when the screen changed.
func (s *Session) Dirty() <-chan struct{} { return s.dirty }

// Done is closed once the program has exited and its output is drained.
func (s *Session) Done() <-chan struct{} { return s.done }

// Err is the program's exit status once Done is closed (nil = success).
func (s *Session) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// Write sends keyboard input to the program.
func (s *Session) Write(p []byte) error {
	_, err := s.p.Write(p)
	return err
}

// Resize changes the terminal size seen by the program.
func (s *Session) Resize(cols, rows int) {
	cols, rows = max(cols, 10), max(rows, 3)
	s.mu.Lock()
	same := cols == s.cols && rows == s.rows
	s.cols, s.rows = cols, rows
	s.mu.Unlock()
	if same {
		return
	}
	s.vt.Resize(cols, rows)
	_ = s.p.Resize(cols, rows)
	s.markDirty()
}

// Size is the current terminal size.
func (s *Session) Size() (cols, rows int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cols, s.rows
}

// Close ends the program and releases the pty.
func (s *Session) Close() {
	s.closeOnce.Do(func() { _ = s.p.Close() })
}

// AppCursor reports whether the program asked for application cursor keys.
func (s *Session) AppCursor() bool {
	s.vt.Lock()
	defer s.vt.Unlock()
	return s.vt.Mode()&vt10x.ModeAppCursor != 0
}

// Render draws the screen with ANSI colours; showCursor marks the cursor cell.
func (s *Session) Render(showCursor bool) string {
	s.vt.Lock()
	defer s.vt.Unlock()
	return render(s.vt, showCursor)
}

// Text is the visible screen as plain text, trailing blanks trimmed.
func (s *Session) Text() string {
	s.vt.Lock()
	defer s.vt.Unlock()
	cols, rows := s.vt.Size()
	lines := make([]string, rows)
	for y := 0; y < rows; y++ {
		var b strings.Builder
		for x := 0; x < cols; x++ {
			b.WriteRune(glyphRune(s.vt.Cell(x, y)))
		}
		lines[y] = strings.TrimRight(b.String(), " ")
	}
	return strings.Trim(strings.Join(lines, "\n"), "\n")
}
