package huffman

// BuildTree constrói a árvore de Huffman de forma determinística a partir das frequências.
// Regras:
//   - Símbolos com freq > 0 são inseridos em ordem crescente de byte (0..255) com Seq monotônico.
//   - Se nenhum símbolo tiver freq > 0 (arquivo vazio), retorna nil.
//   - Se apenas 1 símbolo tiver freq > 0, retorna a própria folha como raiz.
//   - Caso contrário, combina repetidamente os dois menores nós da MinHeap:
//     1º Pop -> Left (bit 0), 2º Pop -> Right (bit 1), atribuindo o próximo Seq ao nó interno.
func BuildTree(freqs [256]uint64) *Node {
	h := NewMinHeap(256)
	var seq uint32

	for b := 0; b < 256; b++ {
		if freqs[b] == 0 {
			continue
		}
		h.Push(&Node{
			Symbol: byte(b),
			Freq:   freqs[b],
			Seq:    seq,
			IsLeaf: true,
		})
		seq++
	}

	if h.Len() == 0 {
		return nil
	}

	for h.Len() > 1 {
		left := h.Pop()
		right := h.Pop()
		h.Push(&Node{
			Freq:   left.Freq + right.Freq,
			Seq:    seq,
			IsLeaf: false,
			Left:   left,
			Right:  right,
		})
		seq++
	}

	return h.Pop()
}