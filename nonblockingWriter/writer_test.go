package nonblockingWriter

import (
	"os"
	"testing"
)

func TestBufWriter_Write(t *testing.T) {
	tcs := map[string]struct {
		cap        int
		data       []byte
		expectData []byte
	}{
		"Normal": {
			cap:        5,
			data:       []byte("hello"),
			expectData: []byte("hello"),
		},
		"Empty": {
			cap:        5,
			data:       []byte(""),
			expectData: []byte(""),
		},
		"Overflow": {
			cap:        3,
			data:       []byte("hello"),
			expectData: []byte("hello"),
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			tmp, _ := os.CreateTemp("", "")
			b := NewBufWriter(tmp, tc.cap)
			defer os.Remove(tmp.Name())
			defer close(b)
			b.Write(tc.data)

			// Check written data
			out, _ := os.ReadFile(tmp.Name())
			if string(out) != string(tc.expectData) {
				t.Fatalf("expected '%s', got '%s'", string(tc.expectData), string(out))
			}

			// Check returned length
			n, _ := b.Write(tc.data)
			if n != len(tc.data) {
				t.Fatalf("expected '%d', got '%d'", len(tc.data), n)
			}
		})
	}
}
