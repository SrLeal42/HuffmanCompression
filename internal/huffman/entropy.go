package huffman

import "math"

// ShannonEntropy calcula a entropia de Shannon H em bits por byte (intervalo [0.0, 8.0]).
// Retorna 0.0 se totalSize == 0 ou se houver no máximo 1 símbolo com frequência > 0.
func ShannonEntropy(freqs [256]uint64, totalSize uint64) float64 {
	if totalSize == 0 {
		return 0.0
	}

	total := float64(totalSize)
	var entropy float64
	for _, count := range freqs {
		if count == 0 {
			continue
		}
		p := float64(count) / total
		entropy -= p * math.Log2(p)
	}

	if entropy <= 0 {
		return 0.0
	}
	return entropy
}

// TheoreticalMinBytes calcula o tamanho teórico mínimo em bytes (H * totalSize / 8)
// segundo o limite de entropia de Shannon. Retorna 0.0 se totalSize == 0 ou entropy <= 0.
func TheoreticalMinBytes(entropy float64, totalSize uint64) float64 {
	if totalSize == 0 || entropy <= 0 {
		return 0.0
	}
	return (entropy * float64(totalSize)) / 8.0
}