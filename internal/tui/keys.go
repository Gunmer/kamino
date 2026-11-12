package tui

import (
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

// KeyKind identifies a key press.
type KeyKind string

const (
	KeyRune      KeyKind = "rune"
	KeyEnter     KeyKind = "enter"
	KeyEsc       KeyKind = "esc"
	KeyTab       KeyKind = "tab"
	KeyShiftTab  KeyKind = "shift_tab"
	KeyBackspace KeyKind = "backspace"
	KeyUp        KeyKind = "up"
	KeyDown      KeyKind = "down"
	KeyLeft      KeyKind = "left"
	KeyRight     KeyKind = "right"
	KeyF1        KeyKind = "f1"
	KeyF2        KeyKind = "f2"
	KeyF3        KeyKind = "f3"
	KeyF4        KeyKind = "f4"
	KeyCtrlC     KeyKind = "ctrl_c"
	KeyCtrlS     KeyKind = "ctrl_s"
	KeyUnknown   KeyKind = "unknown"
)

// Key is one decoded key press.
type Key struct {
	Kind KeyKind
	Rune rune
}

type keyReader struct {
	fd      int
	pending []byte
}

func newKeyReader(fd int) *keyReader { return &keyReader{fd: fd} }

func (r *keyReader) nextByte() (byte, error) {
	if len(r.pending) > 0 {
		value := r.pending[0]
		r.pending = r.pending[1:]
		return value, nil
	}
	buf := make([]byte, 1)
	for {
		n, err := unix.Read(r.fd, buf)
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			return 0, err
		}
		if n == 1 {
			return buf[0], nil
		}
	}
}

func (r *keyReader) peekAvailable(timeoutMS int) []byte {
	fds := []unix.PollFd{{Fd: int32(r.fd), Events: unix.POLLIN}}
	count, err := unix.Poll(fds, timeoutMS)
	if err != nil || count == 0 {
		return nil
	}
	buf := make([]byte, 8)
	n, err := unix.Read(r.fd, buf)
	if err != nil || n <= 0 {
		return nil
	}
	return buf[:n]
}

// ReadKey decodes one key press.
func (r *keyReader) ReadKey() (Key, error) {
	first, err := r.nextByte()
	if err != nil {
		return Key{}, err
	}
	switch first {
	case 0x03:
		return Key{Kind: KeyCtrlC}, nil
	case 0x13:
		return Key{Kind: KeyCtrlS}, nil
	case '\r', '\n':
		return Key{Kind: KeyEnter}, nil
	case '\t':
		return Key{Kind: KeyTab}, nil
	case 0x7f, 0x08:
		return Key{Kind: KeyBackspace}, nil
	case 0x1b:
		rest := r.peekAvailable(15)
		if len(rest) == 0 {
			return Key{Kind: KeyEsc}, nil
		}
		r.pending = append(rest, r.pending...)
		return r.decodeEscape()
	}
	if first < utf8.RuneSelf {
		return Key{Kind: KeyRune, Rune: rune(first)}, nil
	}
	width := 2
	if first >= 0xF0 {
		width = 4
	} else if first >= 0xE0 {
		width = 3
	}
	buf := []byte{first}
	for len(buf) < width {
		next, err := r.nextByte()
		if err != nil {
			return Key{Kind: KeyUnknown}, nil
		}
		buf = append(buf, next)
	}
	value, _ := utf8.DecodeRune(buf)
	return Key{Kind: KeyRune, Rune: value}, nil
}

func (r *keyReader) decodeEscape() (Key, error) {
	second, err := r.nextByte()
	if err != nil {
		return Key{Kind: KeyEsc}, nil
	}
	if second != '[' && second != 'O' {
		return Key{Kind: KeyUnknown}, nil
	}
	third, err := r.nextByte()
	if err != nil {
		return Key{Kind: KeyUnknown}, nil
	}
	switch third {
	case 'A':
		return Key{Kind: KeyUp}, nil
	case 'B':
		return Key{Kind: KeyDown}, nil
	case 'C':
		return Key{Kind: KeyRight}, nil
	case 'D':
		return Key{Kind: KeyLeft}, nil
	case 'Z':
		return Key{Kind: KeyShiftTab}, nil
	case 'P':
		return Key{Kind: KeyF1}, nil
	case 'Q':
		return Key{Kind: KeyF2}, nil
	case 'R':
		return Key{Kind: KeyF3}, nil
	case 'S':
		return Key{Kind: KeyF4}, nil
	case '1', '2', '3', '4':
		number := string(third)
		fourth, err := r.nextByte()
		if err != nil {
			return Key{Kind: KeyUnknown}, nil
		}
		if fourth >= '0' && fourth <= '9' {
			number += string(fourth)
			_, _ = r.nextByte()
			switch number {
			case "11":
				return Key{Kind: KeyF1}, nil
			case "12":
				return Key{Kind: KeyF2}, nil
			case "13":
				return Key{Kind: KeyF3}, nil
			case "14":
				return Key{Kind: KeyF4}, nil
			}
		}
	}
	return Key{Kind: KeyUnknown}, nil
}
