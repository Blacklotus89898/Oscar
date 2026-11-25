//go:build windows

package term

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/UserExistsError/conpty"
)

type winPTY struct{ c *conpty.ConPty }

func startPTY(argv []string, dir string, env []string, cols, rows int) (ptyProc, error) {
	exe, err := exec.LookPath(argv[0])
	if err != nil {
		return nil, err
	}
	parts := make([]string, len(argv))
	parts[0] = syscall.EscapeArg(exe)
	for i, a := range argv[1:] {
		parts[i+1] = syscall.EscapeArg(a)
	}
	opts := []conpty.ConPtyOption{conpty.ConPtyDimensions(cols, rows), conpty.ConPtyEnv(append(os.Environ(), env...))}
	if dir != "" {
		opts = append(opts, conpty.ConPtyWorkDir(dir))
	}
	c, err := conpty.Start(strings.Join(parts, " "), opts...)
	if err != nil {
		return nil, err
	}
	return &winPTY{c}, nil
}

func (p *winPTY) Read(b []byte) (int, error)  { return p.c.Read(b) }
func (p *winPTY) Write(b []byte) (int, error) { return p.c.Write(b) }
func (p *winPTY) Resize(cols, rows int) error { return p.c.Resize(cols, rows) }
func (p *winPTY) Close() error                { return p.c.Close() }

func (p *winPTY) Wait() error {
	code, err := p.c.Wait(context.Background())
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("exit status %d", code)
	}
	return nil
}
