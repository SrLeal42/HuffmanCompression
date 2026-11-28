package huffman

// Node representa um nó (folha ou interno) da árvore de Huffman.
type Node struct {
	Symbol byte   // Valor do byte (0x00 a 0xFF); relevante apenas quando IsLeaf == true
	Freq   uint64 // Frequência acumulada do símbolo ou da subárvore
	Seq    uint32 // Ordem monotônica de criação para desempate determinístico
	IsLeaf bool   // Indica se é um nó folha (símbolo) ou nó interno
	Left   *Node  // Filho esquerdo (caminho de bit 0)
	Right  *Node  // Filho direito (caminho de bit 1)
}

// MinHeap implementa uma fila de prioridade mínima determinística de *Node
// ordenada pela tupla (Freq, Seq).
type MinHeap struct {
	items []*Node
}

// NewMinHeap cria uma MinHeap com capacidade inicial pré-alocada.
func NewMinHeap(capacity int) *MinHeap {
	if capacity < 0 {
		capacity = 0
	}
	return &MinHeap{
		items: make([]*Node, 0, capacity),
	}
}

// Len retorna a quantidade atual de nós na heap.
func (h *MinHeap) Len() int {
	return len(h.items)
}

// Peek retorna o menor nó no topo da heap sem removê-lo, ou nil se estiver vazia.
func (h *MinHeap) Peek() *Node {
	if len(h.items) == 0 {
		return nil
	}
	return h.items[0]
}

// Push insere um nó na heap mantendo a propriedade de ordem em O(log n).
// Ignora chamadas com n == nil.
func (h *MinHeap) Push(n *Node) {
	if n == nil {
		return
	}
	h.items = append(h.items, n)
	h.siftUp(len(h.items) - 1)
}

// Pop remove e retorna o menor nó da heap (menor Freq; em caso de empate, menor Seq)
// em O(log n). Retorna nil se a heap estiver vazia.
func (h *MinHeap) Pop() *Node {
	if len(h.items) == 0 {
		return nil
	}

	top := h.items[0]
	last := len(h.items) - 1
	h.items[0] = h.items[last]
	h.items[last] = nil
	h.items = h.items[:last]

	if len(h.items) > 0 {
		h.siftDown(0)
	}
	return top
}

func lessNode(a, b *Node) bool {
	if a.Freq != b.Freq {
		return a.Freq < b.Freq
	}
	return a.Seq < b.Seq
}

func (h *MinHeap) siftUp(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if !lessNode(h.items[i], h.items[parent]) {
			break
		}
		h.items[i], h.items[parent] = h.items[parent], h.items[i]
		i = parent
	}
}

func (h *MinHeap) siftDown(i int) {
	n := len(h.items)
	for {
		smallest := i
		left := 2*i + 1
		right := 2*i + 2

		if left < n && lessNode(h.items[left], h.items[smallest]) {
			smallest = left
		}
		if right < n && lessNode(h.items[right], h.items[smallest]) {
			smallest = right
		}
		if smallest == i {
			break
		}

		h.items[i], h.items[smallest] = h.items[smallest], h.items[i]
		i = smallest
	}
}