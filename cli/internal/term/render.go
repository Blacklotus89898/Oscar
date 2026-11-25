package term

import (
	"strconv"
	"strings"

	"github.com/hinshun/vt10x"
)

// Glyph attribute bits, mirroring vt10x's unexported constants.
const (
	attrReverse = 1 << iota
	attrUnderline
	attrBold
	attrGfx
	attrItalic
	attrBlink
)

func glyphRune(g vt10x.Glyph) rune {
	if g.Char == 0 {
		return ' '
	}
	return g.Char
}

type style struct {
	fg, bg vt10x.Color
	mode   int16
}

// sgr is the escape sequence that selects st from a reset state.
func (st style) sgr() string {
	codes := []string{"0"}
	m := st.mode
	if m&attrBold != 0 {
		codes = append(codes, "1")
	}
	if m&attrItalic != 0 {
		codes = append(codes, "3")
	}
	if m&attrUnderline != 0 {
		codes = append(codes, "4")
	}
	if m&attrBlink != 0 {
		codes = append(codes, "5")
	}
	if m&attrReverse != 0 {
		codes = append(codes, "7")
	}
	codes = append(codes, colorCodes(st.fg, false)...)
	codes = append(codes, colorCodes(st.bg, true)...)
	return "\x1b[" + strings.Join(codes, ";") + "m"
}

func colorCodes(c vt10x.Color, bg bool) []string {
	base := 30
	if bg {
		base = 40
	}
	switch {
	case c >= vt10x.DefaultFG: // defaults (and the cursor colour)
		return nil
	case c < 8:
		return []string{strconv.Itoa(base + int(c))}
	case c < 16:
		return []string{strconv.Itoa(base + 60 + int(c) - 8)}
	case c < 256:
		return []string{strconv.Itoa(base + 8), "5", strconv.Itoa(int(c))}
	default:
		return []string{strconv.Itoa(base + 8), "2", strconv.Itoa(int(c >> 16 & 0xff)), strconv.Itoa(int(c >> 8 & 0xff)), strconv.Itoa(int(c & 0xff))}
	}
}

// render draws the emulator's screen; the caller holds its lock.
func render(v vt10x.View, showCursor bool) string {
	cols, rows := v.Size()
	cur := v.Cursor()
	cursorOn := showCursor && v.CursorVisible()
	var b strings.Builder
	b.Grow(cols * rows * 2)
	for y := 0; y < rows; y++ {
		var last style
		first := true
		for x := 0; x < cols; x++ {
			g := v.Cell(x, y)
			st := style{g.FG, g.BG, g.Mode &^ (attrGfx | 1<<6)}
			if cursorOn && x == cur.X && y == cur.Y {
				st.mode ^= attrReverse
			}
			if first || st != last {
				b.WriteString(st.sgr())
				last, first = st, false
			}
			b.WriteRune(glyphRune(g))
		}
		b.WriteString("\x1b[0m")
		if y < rows-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
