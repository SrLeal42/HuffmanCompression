package huffman_test

import (
	"testing"

	"huffcompressor/internal/huffman"
)

func TestMinHeapEmptyAndNilInputs(t *testing.T) {
	h := huffman.NewMinHeap(-5)

	if h.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", h.Len())
	}
	if h.Peek() != nil {
		t.Fatalf("Peek() = %+v, want nil", h.Peek())
	}
	if h.Pop() != nil {
		t.Fatalf("Pop() = %+v, want nil", h.Pop())
	}

	h.Push(nil)
	if h.Len() != 0 || h.Peek() != nil || h.Pop() != nil {
		t.Fatalf("Push(nil) modified empty heap: Len=%d", h.Len())
	}
}

func TestMinHeapOrderByFreq(t *testing.T) {
	h := huffman.NewMinHeap(6)
	freqs := []uint64{50, 10, 40, 5, 20, 1}
	for i, f := range freqs {
		h.Push(&huffman.Node{
			Symbol: byte(i),
			Freq:   f,
			Seq:    uint32(i),
			IsLeaf: true,
		})
	}

	want := []uint64{1, 5, 10, 20, 40, 50}
	for i, w := range want {
		if peek := h.Peek(); peek == nil || peek.Freq != w {
			t.Fatalf("step %d: Peek().Freq = %v, want %d", i, peek, w)
		}
		got := h.Pop()
		if got == nil || got.Freq != w {
			t.Fatalf("step %d: Pop().Freq = %v, want %d", i, got, w)
		}
	}

	if h.Len() != 0 || h.Pop() != nil {
		t.Fatalf("expected heap to be empty after popping all elements")
	}
}

func TestMinHeapTieBreakBySeq(t *testing.T) {
	h := huffman.NewMinHeap(5)
	seqs := []uint32{4, 1, 3, 0, 2}
	for _, seq := range seqs {
		h.Push(&huffman.Node{
			Symbol: byte(seq),
			Freq:   10,
			Seq:    seq,
			IsLeaf: true,
		})
	}

	for wantSeq := uint32(0); wantSeq < 5; wantSeq++ {
		got := h.Pop()
		if got == nil {
			t.Fatalf("Pop() returned nil at Seq %d", wantSeq)
		}
		if got.Freq != 10 || got.Seq != wantSeq {
			t.Fatalf("Pop() = (Freq=%d, Seq=%d), want (Freq=10, Seq=%d)", got.Freq, got.Seq, wantSeq)
		}
	}
}

func TestMinHeapHuffmanMergeSimulation(t *testing.T) {
	h := huffman.NewMinHeap(3)
	h.Push(&huffman.Node{Symbol: 'A', Freq: 2, Seq: 0, IsLeaf: true})
	h.Push(&huffman.Node{Symbol: 'B', Freq: 2, Seq: 1, IsLeaf: true})
	h.Push(&huffman.Node{Symbol: 'C', Freq: 4, Seq: 2, IsLeaf: true})

	left := h.Pop()
	right := h.Pop()
	if left == nil || right == nil || left.Seq != 0 || right.Seq != 1 {
		t.Fatalf("unexpected initial pops: left=%+v right=%+v", left, right)
	}

	h.Push(&huffman.Node{
		Freq:   left.Freq + right.Freq,
		Seq:    3,
		IsLeaf: false,
		Left:   left,
		Right:  right,
	})

	next := h.Pop()
	if next == nil || !next.IsLeaf || next.Symbol != 'C' || next.Seq != 2 || next.Freq != 4 {
		t.Fatalf("expected leaf 'C' (Freq=4, Seq=2) before internal node, got %+v", next)
	}

	last := h.Pop()
	if last == nil || last.IsLeaf || last.Seq != 3 || last.Freq != 4 {
		t.Fatalf("expected internal node (Freq=4, Seq=3), got %+v", last)
	}
}

func TestMinHeap256Symbols(t *testing.T) {
	const count = 256
	h := huffman.NewMinHeap(count)

	for i := 0; i < count; i++ {
		idx := (i * 73) % count
		h.Push(&huffman.Node{
			Symbol: byte(idx),
			Freq:   uint64((idx % 17) + 1),
			Seq:    uint32(idx),
			IsLeaf: true,
		})
	}

	if h.Len() != count {
		t.Fatalf("Len() = %d, want %d", h.Len(), count)
	}

	var prev *huffman.Node
	for i := 0; i < count; i++ {
		curr := h.Pop()
		if curr == nil {
			t.Fatalf("Pop() returned nil at index %d", i)
		}
		if prev != nil {
			valid := prev.Freq < curr.Freq || (prev.Freq == curr.Freq && prev.Seq < curr.Seq)
			if !valid {
				t.Fatalf("heap order violated at step %d: prev=(%d,%d) curr=(%d,%d)",
					i, prev.Freq, prev.Seq, curr.Freq, curr.Seq)
			}
		}
		prev = curr
	}

	if h.Len() != 0 {
		t.Fatalf("Len() = %d after popping all, want 0", h.Len())
	}
}