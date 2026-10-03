package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

// --- WriteFrame ---

func TestWriteFrame_RoundTrip(t *testing.T) {
	cases := [][]byte{
		[]byte("hello"),
		[]byte(""),
		[]byte("a"),
		bytes.Repeat([]byte("x"), 1024),
		bytes.Repeat([]byte("y"), MaxFrameSize),
	}
	for _, payload := range cases {
		var buf bytes.Buffer
		if err := WriteFrame(&buf, payload); err != nil {
			t.Fatalf("WriteFrame len=%d: %v", len(payload), err)
		}
		got, err := ReadFrame(&buf)
		if err != nil {
			t.Fatalf("ReadFrame len=%d: %v", len(payload), err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("roundtrip mismatch len=%d: got %d bytes", len(payload), len(got))
		}
	}
}

func TestWriteFrame_OverLimit(t *testing.T) {
	payload := make([]byte, MaxFrameSize+1)
	var buf bytes.Buffer
	err := WriteFrame(&buf, payload)
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("expected ErrFrameTooLarge, got %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("WriteFrame wrote %d bytes before failing — should write nothing", buf.Len())
	}
}

func TestWriteFrame_ExactLimit(t *testing.T) {
	// Ровно MaxFrameSize — граница, должна пройти.
	payload := make([]byte, MaxFrameSize)
	var buf bytes.Buffer
	if err := WriteFrame(&buf, payload); err != nil {
		t.Fatalf("MaxFrameSize should be allowed: %v", err)
	}
}

// --- ReadFrame ---

func TestReadFrame_EmptyReader(t *testing.T) {
	_, err := ReadFrame(bytes.NewReader(nil))
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestReadFrame_TruncatedHeader(t *testing.T) {
	// Заголовок — 4 байта. Даём 2.
	_, err := ReadFrame(bytes.NewReader([]byte{0x00, 0x01}))
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected io.ErrUnexpectedEOF, got %v", err)
	}
}

func TestReadFrame_TruncatedBody(t *testing.T) {
	// length=10, а даём 3 байта payload
	var buf bytes.Buffer
	hdr := make([]byte, 4)
	binary.BigEndian.PutUint32(hdr, 10)
	buf.Write(hdr)
	buf.Write([]byte("abc"))
	_, err := ReadFrame(&buf)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected io.ErrUnexpectedEOF, got %v", err)
	}
}

// TestReadFrame_OverLimit — атакующий отправитель объявляет огромный length.
// Функция должна отвергнуть до аллокации, не уйти в OOM.
func TestReadFrame_OverLimit(t *testing.T) {
	var buf bytes.Buffer
	hdr := make([]byte, 4)
	// length = 4 GB — гигантский, наивный make([]byte, n) положит процесс.
	binary.BigEndian.PutUint32(hdr, 0xFFFFFFFF)
	buf.Write(hdr)
	_, err := ReadFrame(&buf)
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("expected ErrFrameTooLarge, got %v", err)
	}
}

func TestReadFrame_JustOverLimit(t *testing.T) {
	var buf bytes.Buffer
	hdr := make([]byte, 4)
	binary.BigEndian.PutUint32(hdr, MaxFrameSize+1)
	buf.Write(hdr)
	_, err := ReadFrame(&buf)
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("expected ErrFrameTooLarge at MaxFrameSize+1, got %v", err)
	}
}

func TestReadFrame_ZeroLength(t *testing.T) {
	// Пустой payload — валидный фрейм, не ошибка.
	var buf bytes.Buffer
	if err := WriteFrame(&buf, []byte{}); err != nil {
		t.Fatalf("WriteFrame empty: %v", err)
	}
	got, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame empty: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty payload, got %d bytes", len(got))
	}
}

// TestFrame_MultipleInStream — три фрейма подряд в одном буфере,
// читаются последовательно до конца.
func TestFrame_MultipleInStream(t *testing.T) {
	var buf bytes.Buffer
	payloads := [][]byte{
		[]byte("first"),
		[]byte("second-frame"),
		[]byte("3"),
	}
	for _, p := range payloads {
		if err := WriteFrame(&buf, p); err != nil {
			t.Fatalf("WriteFrame: %v", err)
		}
	}
	for i, want := range payloads {
		got, err := ReadFrame(&buf)
		if err != nil {
			t.Fatalf("frame %d: ReadFrame: %v", i, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("frame %d: got %q, want %q", i, got, want)
		}
	}
	// После трёх — пусто.
	if _, err := ReadFrame(&buf); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF after last frame, got %v", err)
	}
}

// TestWriteFrame_PropagatesWriterError — ошибка writer'а не должна теряться.
type failWriter struct{ after int }

func (w *failWriter) Write(p []byte) (int, error) {
	if w.after <= 0 {
		return 0, errors.New("boom")
	}
	if len(p) > w.after {
		n := w.after
		w.after = 0
		return n, errors.New("boom")
	}
	w.after -= len(p)
	return len(p), nil
}

func TestWriteFrame_PropagatesWriterError(t *testing.T) {
	err := WriteFrame(&failWriter{after: 0}, []byte("x"))
	if err == nil {
		t.Fatalf("expected write error to propagate")
	}
}
