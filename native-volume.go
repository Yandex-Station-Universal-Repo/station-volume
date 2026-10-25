// Native Station volume over the existing loopback Quasar IPC protocol.
// No credentials, mixer writes, input-event injection, or speech commands.
package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"strconv"
	"time"
)

const configPath = "/system/vendor/quasar/quasar.cfg"

type ports struct {
	State struct{ Port int `json:"port"` } `json:"stereo_pair"`
	Control struct{ Port int `json:"port"` } `json:"iohub_clients"`
}

type stream struct {
	conn net.Conn
	buf  []byte
}

func openStream(port int) (*stream, error) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("native volume connection: %w", err)
	}
	return &stream{conn: conn}, nil
}

func (s *stream) frame() ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(s.conn, header[:]); err != nil {
		return nil, err
	}
	n := int(binary.LittleEndian.Uint32(header[:]))
	if n == 0 || n > 8*1024*1024 {
		return nil, errors.New("invalid native IPC frame length")
	}
	if cap(s.buf) < n {
		s.buf = make([]byte, n)
	} else {
		s.buf = s.buf[:n]
	}
	_, err := io.ReadFull(s.conn, s.buf)
	return s.buf, err
}

// Skip all unrelated fields without decoding or logging their contents.
func next(data []byte, offset *int) (number, wire uint64, value []byte, integer uint64, err error) {
	key, n := binary.Uvarint(data[*offset:])
	if n <= 0 || key>>3 == 0 {
		err = errors.New("invalid native protobuf field")
		return
	}
	*offset += n
	number, wire = key>>3, key&7
	size := 0
	switch wire {
	case 0:
		integer, n = binary.Uvarint(data[*offset:])
		if n <= 0 {
			err = errors.New("invalid native protobuf integer")
			return
		}
		*offset += n
		return
	case 1:
		size = 8
	case 2:
		var length uint64
		length, n = binary.Uvarint(data[*offset:])
		if n <= 0 || length > uint64(len(data)) {
			err = errors.New("invalid native protobuf length")
			return
		}
		*offset += n
		size = int(length)
	case 5:
		size = 4
	default:
		err = errors.New("unsupported native protobuf wire type")
		return
	}
	if size > len(data)-*offset {
		err = errors.New("truncated native protobuf field")
		return
	}
	value = data[*offset : *offset+size]
	*offset += size
	return
}

func bytesField(data []byte, field uint64) ([]byte, error) {
	for offset := 0; offset < len(data); {
		number, wire, value, _, err := next(data, &offset)
		if err != nil {
			return nil, err
		}
		if number == field && wire == 2 {
			return value, nil
		}
	}
	return nil, nil
}

func volume(frame []byte) (float64, bool, error) {
	var body []byte
	var kind uint64
	for offset := 0; offset < len(frame); {
		number, wire, value, integer, err := next(frame, &offset)
		if err != nil {
			return 0, false, err
		}
		switch {
		case number == 198 && wire == 2:
			body, kind = value, 198
		case number == 252 && wire == 2:
			body = value
		case number == 253 && wire == 0:
			kind = integer
		}
	}
	if kind != 198 {
		return 0, false, nil
	}
	state, err := bytesField(body, 1)
	if err != nil {
		return 0, false, err
	}
	state, err = bytesField(state, 11)
	if err != nil || state == nil {
		return 0, false, err
	}
	muted, present := false, false
	var percent float64
	for offset := 0; offset < len(state); {
		number, wire, value, integer, err := next(state, &offset)
		if err != nil {
			return 0, false, err
		}
		if number == 3 && wire == 0 {
			muted = integer != 0
		} else if number == 6 && wire == 5 {
			percent = float64(math.Float32frombits(binary.LittleEndian.Uint32(value)))
			present = true
		}
	}
	if !present {
		return 0, false, nil
	}
	if math.IsNaN(percent) || math.IsInf(percent, 0) || percent < 0 || percent > 100 {
		return 0, false, errors.New("invalid native volume percentage")
	}
	if muted {
		return 0, true, nil
	}
	return percent, true, nil
}

func (s *stream) readVolume() (float64, error) {
	for {
		frame, err := s.frame()
		if err != nil {
			return 0, err
		}
		percent, present, err := volume(frame)
		if err != nil {
			return 0, err
		}
		if present {
			return percent, nil
		}
	}
}

func appendVarint(dst []byte, value uint64) []byte {
	return binary.AppendUvarint(dst, value)
}

// Public Quasar schema: IOControl88.external_directives95.directive_list1.
// TDirective has Name2, IgnoreAnswer4, PayloadRaw15 and
// DirectiveType20=VolumeSet254. IsLedSilent8 is left unset so the Station
// shows its volume level on the ring and display. The typed payload is
// PercentLevel1(float32) and Source3=Button1: the Station shows the level
// for button changes without a sound (Other1000 shows nothing, Bluetooth2
// also beeps). All non-volume commands and account fields are absent.
func setFrame(percent int) []byte {
	var typed [7]byte
	typed[0] = 13
	binary.LittleEndian.PutUint32(typed[1:5], math.Float32bits(float32(percent)))
	typed[5], typed[6] = 24, 1
	var directive [64]byte
	d := directive[:0]
	d = append(d, 10, 9)
	d = append(d, "directive"...)
	d = append(d, 18, 10)
	d = append(d, "volume_set"...)
	d = append(d, 32, 1, 122, byte(len(typed)))
	d = append(d, typed[:]...)
	d = appendVarint(d, 20<<3)
	d = appendVarint(d, 254)
	// Quasar's IPC envelope carries the typed payload in field252 and its
	// QuasarMessage field number in field253, as observed on native streams.
	var packet [96]byte
	p := packet[:4]
	ioControlLen := 3 + 2 + 2 + len(d)
	p = appendVarint(p, 252<<3|2)
	p = appendVarint(p, uint64(ioControlLen))
	p = appendVarint(p, 95<<3|2)
	p = appendVarint(p, uint64(4+len(d)))
	p = append(p, 10, byte(2+len(d)), 10, byte(len(d)))
	p = append(p, d...)
	p = appendVarint(p, 253<<3)
	p = appendVarint(p, 88)
	binary.LittleEndian.PutUint32(p[:4], uint32(len(p)-4))
	return p
}

func run() error {
	if len(os.Args) < 2 {
		return errors.New("usage: native-volume get|watch|set PERCENT")
	}
	var cfg ports
	file, err := os.Open(configPath)
	if err != nil {
		return err
	}
	err = json.NewDecoder(file).Decode(&cfg)
	file.Close()
	if err != nil || cfg.State.Port == 0 || cfg.Control.Port == 0 {
		return errors.New("native volume IPC endpoints missing from station config")
	}
	s, err := openStream(cfg.State.Port)
	if err != nil {
		return err
	}
	defer s.conn.Close()
	s.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	initial, err := s.readVolume()
	if err != nil {
		return fmt.Errorf("native volume state: %w", err)
	}
	switch os.Args[1] {
	case "get":
		printVolume(initial)
	case "watch":
		printVolume(initial)
		s.conn.SetReadDeadline(time.Time{})
		last := initial
		for {
			value, err := s.readVolume()
			if err != nil {
				return err
			}
			if math.Abs(value-last) > 0.01 {
				printVolume(value)
				last = value
			}
		}
	case "set":
		if len(os.Args) != 3 {
			return errors.New("set requires a percentage from 0 to 100")
		}
		value, err := strconv.ParseFloat(os.Args[2], 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 100 {
			return errors.New("volume must be a percentage from 0 to 100")
		}
		wanted := int(math.Round(value/5)) * 5
		if math.Abs(float64(wanted)-initial) < 0.01 {
			return nil
		}
		control, err := openStream(cfg.Control.Port)
		if err != nil {
			return err
		}
		defer control.conn.Close()
		control.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		packet := setFrame(wanted)
		for len(packet) > 0 {
			n, writeErr := control.conn.Write(packet)
			if writeErr != nil {
				return fmt.Errorf("native volume write: %w", writeErr)
			}
			packet = packet[n:]
		}
		s.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			current, err := s.readVolume()
			if err != nil {
				return fmt.Errorf("native volume did not acknowledge requested level: %w", err)
			}
			if math.Abs(current-float64(wanted)) < 0.01 {
				return nil
			}
		}
	default:
		return errors.New("usage: native-volume get|watch|set PERCENT")
	}
	return nil
}

func printVolume(percent float64) {
	fmt.Println(strconv.FormatFloat(percent, 'f', -1, 32))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "native-volume:", err)
		os.Exit(1)
	}
}
