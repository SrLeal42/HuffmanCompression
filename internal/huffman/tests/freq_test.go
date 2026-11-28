package huffman_test

import (
	"bytes"
	"errors"
	"hash/crc32"
	"io"
	"math"
	"testing"
	"testing/iotest"

	"huffcompressor/internal/huffman"
)

func TestScanReaderEmpty(t *testing.T) {
	res, err := huffman.ScanReader(bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("ScanReader(empty) returned unexpected error: %v", err)
	}

	if res.OriginalSize != 0 {
		t.Errorf("OriginalSize = %d, want 0", res.OriginalSize)
	}
	if res.SymbolCount != 0 {
		t.Errorf("SymbolCount = %d, want 0", res.SymbolCount)
	}
	if res.CRC32 != 0 {
		t.Errorf("CRC32 = %#x, want 0x0", res.CRC32)
	}
	for b, count := range res.Freqs {
		if count != 0 {
			t.Fatalf("Freqs[%d] = %d, want 0", b, count)
		}
	}

	entropy := huffman.ShannonEntropy(res.Freqs, res.OriginalSize)
	if math.IsNaN(entropy) || entropy != 0.0 {
		t.Errorf("ShannonEntropy = %v, want 0.0", entropy)
	}
	if minBytes := huffman.TheoreticalMinBytes(entropy, res.OriginalSize); minBytes != 0.0 {
		t.Errorf("TheoreticalMinBytes = %v, want 0.0", minBytes)
	}
}

func TestScanReaderSingleRepeatedSymbol(t *testing.T) {
	payload := bytes.Repeat([]byte{'A'}, 1000)

	res, err := huffman.ScanReader(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("ScanReader returned unexpected error: %v", err)
	}

	if res.OriginalSize != 1000 {
		t.Errorf("OriginalSize = %d, want 1000", res.OriginalSize)
	}
	if res.SymbolCount != 1 {
		t.Errorf("SymbolCount = %d, want 1", res.SymbolCount)
	}
	if res.Freqs['A'] != 1000 {
		t.Errorf("Freqs['A'] = %d, want 1000", res.Freqs['A'])
	}
	if wantCRC := crc32.ChecksumIEEE(payload); res.CRC32 != wantCRC {
		t.Errorf("CRC32 = %#x, want %#x", res.CRC32, wantCRC)
	}

	entropy := huffman.ShannonEntropy(res.Freqs, res.OriginalSize)
	if entropy != 0.0 || math.Signbit(entropy) {
		t.Errorf("ShannonEntropy = %v (signbit=%v), want +0.0", entropy, math.Signbit(entropy))
	}
	if minBytes := huffman.TheoreticalMinBytes(entropy, res.OriginalSize); minBytes != 0.0 {
		t.Errorf("TheoreticalMinBytes = %v, want 0.0", minBytes)
	}
}

func TestShannonEntropyExactDistributions(t *testing.T) {
	const eps = 1e-9

	allBytes := make([]byte, 0, 1024)
	for rep := 0; rep < 4; rep++ {
		for b := 0; b < 256; b++ {
			allBytes = append(allBytes, byte(b))
		}
	}

	fourSymbols := make([]byte, 0, 1000)
	for _, s := range []byte{'A', 'B', 'C', 'D'} {
		fourSymbols = append(fourSymbols, bytes.Repeat([]byte{s}, 250)...)
	}

	twoSymbols := append(bytes.Repeat([]byte{'0'}, 500), bytes.Repeat([]byte{'1'}, 500)...)

	tests := []struct {
		name            string
		payload         []byte
		wantSymbolCount uint16
		wantEntropy     float64
		wantMinBytes    float64
	}{
		{
			name:            "2_equiprobable_symbols",
			payload:         twoSymbols,
			wantSymbolCount: 2,
			wantEntropy:     1.0,
			wantMinBytes:    125.0,
		},
		{
			name:            "4_equiprobable_symbols",
			payload:         fourSymbols,
			wantSymbolCount: 4,
			wantEntropy:     2.0,
			wantMinBytes:    250.0,
		},
		{
			name:            "256_equiprobable_symbols",
			payload:         allBytes,
			wantSymbolCount: 256,
			wantEntropy:     8.0,
			wantMinBytes:    1024.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := huffman.ScanReader(bytes.NewReader(tc.payload))
			if err != nil {
				t.Fatalf("ScanReader returned unexpected error: %v", err)
			}
			if res.SymbolCount != tc.wantSymbolCount {
				t.Errorf("SymbolCount = %d, want %d", res.SymbolCount, tc.wantSymbolCount)
			}

			entropy := huffman.ShannonEntropy(res.Freqs, res.OriginalSize)
			if math.Abs(entropy-tc.wantEntropy) > eps {
				t.Errorf("ShannonEntropy = %.12f, want %.12f", entropy, tc.wantEntropy)
			}

			minBytes := huffman.TheoreticalMinBytes(entropy, res.OriginalSize)
			if math.Abs(minBytes-tc.wantMinBytes) > eps {
				t.Errorf("TheoreticalMinBytes = %.12f, want %.12f", minBytes, tc.wantMinBytes)
			}
		})
	}
}

func TestScanReaderMultiChunkAndFragmentedReads(t *testing.T) {
	const size = 150 * 1024 // > 2 * DefaultBufferSize
	payload := make([]byte, size)
	var wantFreqs [256]uint64
	for i := range payload {
		b := byte((i * 31) & 0xFF)
		payload[i] = b
		wantFreqs[b]++
	}
	wantCRC := crc32.ChecksumIEEE(payload)

	readers := []struct {
		name string
		r    func() io.Reader
	}{
		{name: "direct", r: func() io.Reader { return bytes.NewReader(payload) }},
		{name: "one_byte_reader", r: func() io.Reader { return iotest.OneByteReader(bytes.NewReader(payload)) }},
		{name: "data_err_reader", r: func() io.Reader { return iotest.DataErrReader(bytes.NewReader(payload)) }},
	}

	for _, tc := range readers {
		t.Run(tc.name, func(t *testing.T) {
			res, err := huffman.ScanReader(tc.r())
			if err != nil {
				t.Fatalf("ScanReader returned unexpected error: %v", err)
			}
			if res.OriginalSize != size {
				t.Errorf("OriginalSize = %d, want %d", res.OriginalSize, size)
			}
			if res.CRC32 != wantCRC {
				t.Errorf("CRC32 = %#x, want %#x", res.CRC32, wantCRC)
			}
			if res.SymbolCount != 256 {
				t.Errorf("SymbolCount = %d, want 256", res.SymbolCount)
			}
			if res.Freqs != wantFreqs {
				t.Errorf("Freqs mismatch against expected counts")
			}
		})
	}
}

func TestScanReaderErrorPropagation(t *testing.T) {
	wantErr := errors.New("disk error")
	if _, err := huffman.ScanReader(iotest.ErrReader(wantErr)); !errors.Is(err, wantErr) {
		t.Errorf("ScanReader error = %v, want %v", err, wantErr)
	}
	if _, err := huffman.ScanReader(nil); err == nil {
		t.Errorf("ScanReader(nil) = nil, want non-nil error")
	}
}

func TestTheoreticalMinBytesEdgeCases(t *testing.T) {
	if got := huffman.TheoreticalMinBytes(4.0, 0); got != 0.0 {
		t.Errorf("TheoreticalMinBytes(4.0, 0) = %v, want 0.0", got)
	}
	if got := huffman.TheoreticalMinBytes(-1.0, 100); got != 0.0 {
		t.Errorf("TheoreticalMinBytes(-1.0, 100) = %v, want 0.0", got)
	}
}