package term

import (
	tea "github.com/charmbracelet/bubbletea"
)

// KeyBytes encodes a key press the way an xterm would send it to the program.
// appCursor selects application cursor keys (vim, less and tmux ask for them).
func KeyBytes(k tea.KeyMsg, appCursor bool) []byte {
	var out []byte
	switch {
	case k.Type == tea.KeyRunes:
		out = []byte(string(k.Runes))
	case k.Type == tea.KeySpace:
		out = []byte{' '}
	case k.Type >= 0 && k.Type <= 31, k.Type == 127:
		// Control characters: ctrl+a..z, enter (\r), tab, esc, backspace (DEL).
		out = []byte{byte(k.Type)}
	default:
		seq, ok := special(k.Type, appCursor)
		if !ok {
			return nil
		}
		out = []byte(seq)
	}
	if k.Alt {
		out = append([]byte{0x1b}, out...)
	}
	return out
}

func special(t tea.KeyType, appCursor bool) (string, bool) {
	arrow := func(c byte) string {
		if appCursor {
			return "\x1bO" + string(c)
		}
		return "\x1b[" + string(c)
	}
	switch t {
	case tea.KeyUp:
		return arrow('A'), true
	case tea.KeyDown:
		return arrow('B'), true
	case tea.KeyRight:
		return arrow('C'), true
	case tea.KeyLeft:
		return arrow('D'), true
	case tea.KeyHome:
		return arrow('H'), true
	case tea.KeyEnd:
		return arrow('F'), true
	case tea.KeyShiftTab:
		return "\x1b[Z", true
	case tea.KeyPgUp:
		return "\x1b[5~", true
	case tea.KeyPgDown:
		return "\x1b[6~", true
	case tea.KeyDelete:
		return "\x1b[3~", true
	case tea.KeyInsert:
		return "\x1b[2~", true
	case tea.KeyCtrlUp:
		return "\x1b[1;5A", true
	case tea.KeyCtrlDown:
		return "\x1b[1;5B", true
	case tea.KeyCtrlRight:
		return "\x1b[1;5C", true
	case tea.KeyCtrlLeft:
		return "\x1b[1;5D", true
	case tea.KeyShiftUp:
		return "\x1b[1;2A", true
	case tea.KeyShiftDown:
		return "\x1b[1;2B", true
	case tea.KeyShiftRight:
		return "\x1b[1;2C", true
	case tea.KeyShiftLeft:
		return "\x1b[1;2D", true
	case tea.KeyF1:
		return "\x1bOP", true
	case tea.KeyF2:
		return "\x1bOQ", true
	case tea.KeyF3:
		return "\x1bOR", true
	case tea.KeyF4:
		return "\x1bOS", true
	case tea.KeyF5:
		return "\x1b[15~", true
	case tea.KeyF6:
		return "\x1b[17~", true
	case tea.KeyF7:
		return "\x1b[18~", true
	case tea.KeyF8:
		return "\x1b[19~", true
	case tea.KeyF9:
		return "\x1b[20~", true
	case tea.KeyF10:
		return "\x1b[21~", true
	case tea.KeyF11:
		return "\x1b[23~", true
	case tea.KeyF12:
		return "\x1b[24~", true
	}
	return "", false
}
