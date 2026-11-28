package huffman

// Code representa o código binário de comprimento variável atribuído a um símbolo.
type Code struct {
	Bits    []byte // Bits empacotados MSB-first: bit i está em Bits[i/8] na posição 7-(i%8)
	Fast    uint64 // Caminho rápido: código nos BitLen bits menos significativos (válido se BitLen <= 64)
	BitLen  uint16 // Comprimento do código em bits (0 para árvore de folha única ou símbolo ausente)
	Present bool   // true se o símbolo existe na árvore
}

// CodeTable mapeia cada possível byte (0x00 a 0xFF) para seu respectivo Code.
type CodeTable [256]Code

// GenerateCodes percorre a árvore de Huffman em profundidade (DFS) e retorna a tabela
// de códigos para todos os símbolos presentes. Se root == nil, retorna uma CodeTable zerada.
func GenerateCodes(root *Node) CodeTable {
	var table CodeTable
	if root == nil {
		return table
	}

	if root.IsLeaf {
		table[root.Symbol] = Code{
			BitLen:  0,
			Present: true,
		}
		return table
	}

	path := make([]uint8, 0, 256)
	collectCodes(root, path, &table)
	return table
}

func collectCodes(node *Node, path []uint8, table *CodeTable) {
	if node == nil {
		return
	}
	if node.IsLeaf {
		table[node.Symbol] = buildCode(path)
		return
	}

	collectCodes(node.Left, append(path, 0), table)
	collectCodes(node.Right, append(path, 1), table)
}

func buildCode(path []uint8) Code {
	bitLen := len(path)
	packed := make([]byte, (bitLen+7)/8)
	var fast uint64

	for i, bit := range path {
		if bit == 1 {
			packed[i/8] |= 1 << (7 - (i % 8))
		}
		if bitLen <= 64 {
			fast = (fast << 1) | uint64(bit)
		}
	}

	return Code{
		Bits:    packed,
		Fast:    fast,
		BitLen:  uint16(bitLen),
		Present: true,
	}
}

// BitAt retorna o valor do bit (0 ou 1) na posição index (0 <= index < int(c.BitLen)).
// Entra em panic se index estiver fora do intervalo [0, c.BitLen).
func (c Code) BitAt(index int) uint8 {
	if index < 0 || index >= int(c.BitLen) {
		panic("huffman: bit index out of range")
	}
	return (c.Bits[index/8] >> (7 - (index % 8))) & 1
}

// String retorna a representação binária textual do código (ex.: "0101").
// Para símbolo único na árvore (Present && BitLen == 0), retorna "0 (implícito)" conforme a SPEC.
// Para símbolo ausente (!Present), retorna "".
func (c Code) String() string {
	if !c.Present {
		return ""
	}
	if c.BitLen == 0 {
		return "0 (implícito)"
	}

	buf := make([]byte, c.BitLen)
	for i := 0; i < int(c.BitLen); i++ {
		buf[i] = '0' + c.BitAt(i)
	}
	return string(buf)
}