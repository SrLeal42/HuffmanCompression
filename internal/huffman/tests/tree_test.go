package huffman_test

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"huffcompressor/internal/huffman"
)

func TestBuildTreeAndGenerateCodesEmpty(t *testing.T) {
	var freqs [256]uint64
	root := huffman.BuildTree(freqs)
	if root != nil {
		t.Fatalf("BuildTree(empty) = %+v, want nil", root)
	}

	table := huffman.GenerateCodes(nil)
	for sym, code := range table {
		if code.Present || code.BitLen != 0 || code.String() != "" {
			t.Fatalf("table[%d] = %+v (String=%q), want zeroed absent Code", sym, code, code.String())
		}
	}
}

func TestBuildTreeAndGenerateCodesSingleSymbol(t *testing.T) {
	var freqs [256]uint64
	freqs['Z'] = 500

	root := huffman.BuildTree(freqs)
	if root == nil || !root.IsLeaf || root.Symbol != 'Z' || root.Freq != 500 {
		t.Fatalf("BuildTree(single) = %+v, want leaf 'Z' with Freq=500", root)
	}

	table := huffman.GenerateCodes(root)
	if !table['Z'].Present || table['Z'].BitLen != 0 || table['Z'].String() != "0 (implícito)" {
		t.Fatalf("table['Z'] = %+v (String=%q), want Present=true BitLen=0 '0 (implícito)'",
			table['Z'], table['Z'].String())
	}

	for sym, code := range table {
		if sym == 'Z' {
			continue
		}
		if code.Present || code.String() != "" {
			t.Fatalf("unexpected present symbol %d: %+v", sym, code)
		}
	}
}

func TestGenerateCodesPrefixFreeKraftAndMonotonicity(t *testing.T) {
	var textFreqs [256]uint64
	textFreqs['e'] = 120
	textFreqs['a'] = 90
	textFreqs['t'] = 65
	textFreqs['o'] = 50
	textFreqs['n'] = 40
	textFreqs['r'] = 30
	textFreqs['z'] = 5
	textFreqs['q'] = 2

	var twoFreqs [256]uint64
	twoFreqs['0'] = 10
	twoFreqs['1'] = 30

	var allFreqs [256]uint64
	for b := 0; b < 256; b++ {
		allFreqs[b] = uint64(b + 1)
	}

	cases := []struct {
		name  string
		freqs [256]uint64
	}{
		{name: "two_symbols", freqs: twoFreqs},
		{name: "asymmetric_text", freqs: textFreqs},
		{name: "all_256_symbols", freqs: allFreqs},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := huffman.BuildTree(tc.freqs)
			table := huffman.GenerateCodes(root)

			var presentSyms []byte
			var kraftSum float64
			for b := 0; b < 256; b++ {
				if tc.freqs[b] == 0 {
					if table[b].Present {
						t.Fatalf("absent symbol %d marked Present", b)
					}
					continue
				}
				c := table[b]
				if !c.Present || c.BitLen == 0 {
					t.Fatalf("present symbol %d has invalid code %+v", b, c)
				}
				presentSyms = append(presentSyms, byte(b))
				kraftSum += math.Pow(2, -float64(c.BitLen))

				if c.BitLen <= 64 {
					var fromFast strings.Builder
					for bitIdx := int(c.BitLen) - 1; bitIdx >= 0; bitIdx-- {
						if (c.Fast>>uint(bitIdx))&1 == 1 {
							fromFast.WriteByte('1')
						} else {
							fromFast.WriteByte('0')
						}
					}
					if fromFast.String() != c.String() {
						t.Fatalf("symbol %d Fast mismatch: fast=%q string=%q", b, fromFast.String(), c.String())
					}
				}
			}

			if math.Abs(kraftSum-1.0) > 1e-12 {
				t.Errorf("Kraft sum = %.15f, want 1.0", kraftSum)
			}

			for i, a := range presentSyms {
				codeA := table[a].String()
				for j, b := range presentSyms {
					if i == j {
						continue
					}
					codeB := table[b].String()
					if strings.HasPrefix(codeB, codeA) {
						t.Fatalf("prefix property violated: code(%d)=%q is prefix of code(%d)=%q",
							a, codeA, b, codeB)
					}
					if tc.freqs[a] > tc.freqs[b] && table[a].BitLen > table[b].BitLen {
						t.Fatalf("monotonicity violated: freq(%d)=%d len=%d > freq(%d)=%d len=%d",
							a, tc.freqs[a], table[a].BitLen, b, tc.freqs[b], table[b].BitLen)
					}
				}
			}
		})
	}
}

func TestBuildTreeAndCodesDeterminism(t *testing.T) {
	var freqs [256]uint64
	freqs['a'] = 10
	freqs['b'] = 10
	freqs['c'] = 10
	freqs['d'] = 20
	freqs['e'] = 20
	freqs['f'] = 5
	freqs['g'] = 5

	baseline := huffman.GenerateCodes(huffman.BuildTree(freqs))

	for iter := 0; iter < 50; iter++ {
		got := huffman.GenerateCodes(huffman.BuildTree(freqs))
		for sym := 0; sym < 256; sym++ {
			if got[sym].Present != baseline[sym].Present ||
				got[sym].BitLen != baseline[sym].BitLen ||
				got[sym].Fast != baseline[sym].Fast ||
				!bytes.Equal(got[sym].Bits, baseline[sym].Bits) {
				t.Fatalf("non-deterministic code at iter %d symbol %d: got=%+v want=%+v",
					iter, sym, got[sym], baseline[sym])
			}
		}
	}
}

func TestGenerateCodesFibonacciOver64Bits(t *testing.T) {
	const count = 80
	var freqs [256]uint64
	freqs[0] = 1
	freqs[1] = 1
	for i := 2; i < count; i++ {
		freqs[i] = freqs[i-1] + freqs[i-2]
	}

	root := huffman.BuildTree(freqs)
	table := huffman.GenerateCodes(root)

	var maxLen uint16
	for sym := 0; sym < count; sym++ {
		c := table[sym]
		if !c.Present {
			t.Fatalf("symbol %d not marked Present", sym)
		}
		if c.BitLen > maxLen {
			maxLen = c.BitLen
		}

		curr := root
		for step := 0; step < int(c.BitLen); step++ {
			if curr == nil || curr.IsLeaf {
				t.Fatalf("symbol %d ended prematurely at step %d/%d", sym, step, c.BitLen)
			}
			if c.BitAt(step) == 0 {
				curr = curr.Left
			} else {
				curr = curr.Right
			}
		}
		if curr == nil || !curr.IsLeaf || curr.Symbol != byte(sym) {
			t.Fatalf("tree walk for symbol %d landed on %+v", sym, curr)
		}
	}

	if maxLen != count-1 {
		t.Fatalf("maxLen = %d, want %d (> 64)", maxLen, count-1)
	}
}

func TestCodeBitAtOutOfRangePanics(t *testing.T) {
	var freqs [256]uint64
	freqs['A'] = 1
	freqs['B'] = 2
	table := huffman.GenerateCodes(huffman.BuildTree(freqs))
	code := table['A']

	for _, badIdx := range []int{-1, int(code.BitLen)} {
		func(idx int) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("BitAt(%d) did not panic for BitLen=%d", idx, code.BitLen)
				}
			}()
			_ = code.BitAt(idx)
		}(badIdx)
	}
}