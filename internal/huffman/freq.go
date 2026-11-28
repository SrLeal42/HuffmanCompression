package huffman

import (
	"errors"
	"hash/crc32"
	"io"
)

// DefaultBufferSize define o tamanho do bloco de leitura em streaming (64 KB).
const DefaultBufferSize = 64 * 1024

// ScanResult agrupa os metadados extraídos na primeira passagem sobre os dados originais.
type ScanResult struct {
	Freqs        [256]uint64 // Contagem de ocorrências de cada byte (0x00 a 0xFF)
	OriginalSize uint64      // Total de bytes lidos
	CRC32        uint32      // Checksum CRC32 (polinômio IEEE) dos dados originais
	SymbolCount  uint16      // Quantidade de símbolos distintos com frequência > 0 (0 a 256)
}

// ScanReader lê um io.Reader até io.EOF em blocos de tamanho fixo, calculando
// a tabela de frequências, o tamanho total, o CRC32 IEEE e a contagem de símbolos distintos
// em uma única passagem e com uso de memória constante O(1).
func ScanReader(r io.Reader) (ScanResult, error) {
	if r == nil {
		return ScanResult{}, errors.New("huffman: nil reader")
	}

	crc := crc32.NewIEEE()
	buf := make([]byte, DefaultBufferSize)
	var res ScanResult

	for {
		n, err := r.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			res.OriginalSize += uint64(n)
			_, _ = crc.Write(chunk)
			for _, b := range chunk {
				res.Freqs[b]++
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return ScanResult{}, err
		}
	}

	res.CRC32 = crc.Sum32()
	for _, count := range res.Freqs {
		if count > 0 {
			res.SymbolCount++
		}
	}

	return res, nil
}