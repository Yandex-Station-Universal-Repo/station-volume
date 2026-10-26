package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fixtureField(n uint64, b []byte) []byte {
	v := binary.AppendUvarint(nil, n<<3|2)
	v = binary.AppendUvarint(v, uint64(len(b)))
	return append(v, b...)
}
func fixtureVolume(percent float32, muted bool) []byte {
	v := []byte{53}
	v = binary.LittleEndian.AppendUint32(v, math.Float32bits(percent))
	if muted {
		v = append(v, 24, 1)
	}
	return fixtureField(198, fixtureField(1, fixtureField(11, v)))
}
func fixtureSend(c net.Conn, b []byte) error {
	packet := binary.LittleEndian.AppendUint32(nil, uint32(len(b)))
	packet = append(packet, b...)
	_, e := c.Write(packet)
	return e
}
func fixtureListen(t *testing.T) net.Listener {
	t.Helper()
	l, e := net.Listen("tcp4", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { l.Close() })
	return l
}
func fixtureConfig(t *testing.T, state, control net.Listener) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "quasar.cfg")
	s := state.Addr().(*net.TCPAddr).Port
	c := control.Addr().(*net.TCPAddr).Port
	if e := os.WriteFile(path, []byte(fmt.Sprintf(`{"stereo_pair":{"port":%d},"iohub_clients":{"port":%d}}`, s, c)), 0600); e != nil {
		t.Fatal(e)
	}
	return path
}
func TestSetFrameSchemaGolden(t *testing.T) {
	// Independently transcribed schema fixture: field252 IOControl88, VolumeSet254,
	// PercentLevel50.0f, Source=Button1. No credentials or arbitrary directives.
	want, e := hex.DecodeString("33000000e20f2dfa052a0a280a260a09646972656374697665120a766f6c756d655f73657420017a070d000048421801a001fe01e80f58")
	if e != nil {
		t.Fatal(e)
	}
	if got := setFrame(50); !bytes.Equal(got, want) {
		t.Fatalf("got %x want %x", got, want)
	}
}
func TestVolumeDecodesWrappedMutedAndRejectsInvalid(t *testing.T) {
	for _, wrapped := range []bool{false, true} {
		for _, mute := range []bool{false, true} {
			b := fixtureVolume(70, mute)
			if wrapped {
				body, e := bytesField(b, 198)
				if e != nil {
					t.Fatal(e)
				}
				b = fixtureField(252, body)
				b = append(b, 232, 15, 198, 1)
			}
			v, ok, e := volume(b)
			want := float64(70)
			if mute {
				want = 0
			}
			if e != nil || !ok || v != want {
				t.Fatal(v, ok, e)
			}
		}
	}
	for _, v := range []float32{float32(math.NaN()), float32(math.Inf(1)), -1, 101} {
		if _, _, e := volume(fixtureVolume(v, false)); e == nil {
			t.Fatal(v)
		}
	}
	for _, bad := range [][]byte{{0}, {255}, {10, 99, 0}} {
		if _, _, e := volume(bad); e == nil {
			t.Fatal("malformed protobuf accepted")
		}
	}
}
func TestSetWaitsForNativeAcknowledgement(t *testing.T) {
	state, control := fixtureListen(t), fixtureListen(t)
	path := fixtureConfig(t, state, control)
	done := make(chan error, 1)
	go func() {
		s, e := state.Accept()
		if e != nil {
			done <- e
			return
		}
		defer s.Close()
		_ = s.SetDeadline(time.Now().Add(5 * time.Second))
		if e = fixtureSend(s, fixtureVolume(70, false)); e != nil {
			done <- e
			return
		}
		c, e := control.Accept()
		if e != nil {
			done <- e
			return
		}
		defer c.Close()
		_ = c.SetDeadline(time.Now().Add(5 * time.Second))
		var header [4]byte
		if _, e = io.ReadFull(c, header[:]); e != nil {
			done <- e
			return
		}
		b := make([]byte, binary.LittleEndian.Uint32(header[:]))
		if _, e = io.ReadFull(c, b); e != nil {
			done <- e
			return
		}
		if !bytes.Equal(b, setFrame(65)[4:]) {
			done <- fmt.Errorf("wrong quantised directive")
			return
		}
		done <- fixtureSend(s, fixtureVolume(65, false))
	}()
	if e := runWith([]string{"set", "67"}, path, io.Discard); e != nil {
		t.Fatal(e)
	}
	if e := <-done; e != nil {
		t.Fatal(e)
	}
}
func TestMalformedStateNeverWrites(t *testing.T) {
	state, control := fixtureListen(t), fixtureListen(t)
	path := fixtureConfig(t, state, control)
	go func() {
		c, e := state.Accept()
		if e == nil {
			defer c.Close()
			_, _ = c.Write([]byte{255, 255, 255, 255})
		}
	}()
	if e := runWith([]string{"set", "50"}, path, io.Discard); e == nil {
		t.Fatal("invalid native state accepted")
	}
	_ = control.(*net.TCPListener).SetDeadline(time.Now().Add(50 * time.Millisecond))
	if c, e := control.Accept(); e == nil {
		c.Close()
		t.Fatal("wrote before valid state")
	}
}
func TestBadArgumentsRejectedWithoutConfig(t *testing.T) {
	for _, args := range [][]string{nil, {"get", "extra"}, {"watch", "extra"}, {"set"}, {"set", "50", "--force"}, {"arbitrary"}} {
		if e := runWith(args, "/nonexistent-volume-fixture", io.Discard); e == nil {
			t.Fatal(args)
		}
	}
}
