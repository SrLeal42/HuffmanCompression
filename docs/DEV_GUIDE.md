## Fase 0 — Ambiente e Fundação
**Objetivo:** Garantir toolchain funcional (Go + `cgo` + Fyne) antes de escrever o núcleo.

- [x] **0.1 Validação de ambiente:** Confirmar versão do Go e compilador C (`gcc` via MSYS2/MinGW-w64 no Windows).
- [x] **0.2 Setup do projeto:** Inicializar `go.mod`, criar árvore de diretórios (`internal/huffman/`, `ui/`) e validar a versão do Fyne (confirmar suporte a `fyne.Do`).
- [x] **0.3 Smoke test Fyne:** Compilar e abrir uma janela mínima em `main.go` para absorver o primeiro build lento do `cgo`.

**Critério de pronto:** `go build ./...` compila sem erros e a janela mínima abre e fecha normalmente.

---

## Fase 1 — Estruturas Fundamentais do Núcleo (`internal/huffman`)
**Objetivo:** Implementar e provar matematicamente a construção da árvore e geração de códigos sem dependência de I/O de formato ou GUI.

- [x] **1.1 Frequências e Entropia** (`freq.go`, `entropy.go`, `freq_test.go`)
  - Contagem via `io.Reader` + `bufio.Reader` em `[256]uint64` (sem `map`), computando `OriginalSize` e `CRC32` (IEEE) na mesma passagem.
  - Cálculo da Entropia de Shannon (`H`) e tamanho teórico mínimo (`H * size / 8`).
- [x] **1.2 Min Heap Determinístico** (`heap.go`, `heap_test.go`)
  - Min Heap escrito do zero ordenando por `(freq, seq)`.
  - Testes unitários garantindo ordem estrita e desempate determinístico por `seq`.
- [x] **1.3 Árvore de Huffman e Gerador de Códigos** (`tree.go`, `codes.go`, `tree_test.go`)
  - Construção da árvore inserindo folhas em ordem crescente de byte (`0..255`) com `seq` monotônico; 1º pop -> esquerda (`0`), 2º pop -> direita (`1`).
  - Estrutura `Code` com suporte a comprimentos `> 64` bits (slice de bytes MSB-first) + *fast path* em `uint64` para `<= 64` bits.
  - Casos especiais: 0 símbolos e 1 símbolo único (código de 0 bits).
  - Testes: propriedade *prefix-free*, igualdade de Kraft ($\sum 2^{-L_i} = 1$), determinismo e teste sintético de Fibonacci (~80 símbolos, código > 64 bits).

**Critério de pronto:** `go test ./internal/huffman/...` passando em todos os testes unitários da Fase 1.

---

## Fase 2 — Bit I/O e Formato `.huff` (`HUF1`)
**Objetivo:** Garantir serialização binária exata em nível de bit e validação defensiva de cabeçalhos.

- [ ] **2.1 BitWriter e BitReader** (`bitio.go`, `bitio_test.go`)
  - `BitWriter` com acumulador de 64 bits sobre `bufio.Writer`, ordem MSB-first e `Flush()` completando o último byte com zeros.
  - `BitReader` sobre `bufio.Reader` retornando `io.ErrUnexpectedEOF` em leitura truncada.
  - Testes de ida e volta com sequências de comprimentos variados (incluindo `> 64` bits) e verificação do padding.
- [ ] **2.2 Cabeçalho `.huff` (`HUF1`)** (`format.go`, `format_test.go`)
  - Escrita e leitura little-endian + `binary.PutUvarint` / `binary.ReadUvarint`.
  - Validações obrigatórias: Magic `HUF1`, `SymbolCount <= 256`, símbolos em ordem estritamente crescente (sem duplicados), frequência > 0, soma das frequências == `OriginalSize` (com checagem de overflow), `Filename` não vazio e sanitizado contra *path traversal* (`filepath.Base`, rejeitando separadores, `.` e `..`).
  - Testes negativos cobrindo cada regra de rejeição.

**Critério de pronto:** Todos os testes de Bit I/O e de cabeçalhos válidos/malformados passando.

---

## Fase 3 — Encoder, Decoder e Robustez (Marco do Núcleo)
**Objetivo:** Fechar o motor completo de compressão/descompressão de arquivos reais com garantia de integridade.

- [ ] **3.1 Encoder e Decoder** (`encoder.go`, `decoder.go`)
  - `Encode`: 2 passagens com `bufio`, callback de progresso `func(done, total int64)`, retorno de struct de estatísticas (tamanhos, tempos, entropia, tabela de símbolos/códigos para a GUI).
  - `Decode`: leitura/validação do cabeçalho, reconstrução da árvore, decodificação até `OriginalSize` bytes (tratando 0 símbolos e 1 símbolo sem payload), escrita em arquivo temporário no diretório de destino, checagem de `CRC32` e `os.Rename` atômico (removendo o temporário em caso de erro).
- [ ] **3.2 Testes Round-Trip, Corrupção e Fuzzing** (`codec_test.go`, `fuzz_test.go`)
  - Round-trip: arquivo vazio, 1 byte, 1 símbolo repetido, 256 símbolos, dados aleatórios, texto.
  - Arquivos inválidos: magic errado, truncamento, bit-flip (erro de CRC), tabela fora de ordem, soma incorreta, nome `../`.
  - Fuzzing nativo: `FuzzRoundTrip` (`decode(encode(x)) == x`) e `FuzzDecode` (nunca entra em `panic` nem consome memória excessiva).

**Critério de pronto (Ponto de Controle):** Núcleo 100% funcional, `go test -race ./internal/huffman/...` verde e fuzzing executado sem falhas.

---

## Fase 4 — Interface Gráfica Essencial (`ui/` e `main.go`)
**Objetivo:** Expor o núcleo via Fyne de forma responsiva e didática.

- [ ] **4.1 Fluxo Principal e Assincronismo** (`ui/app.go`, `main.go`)
  - Seleção de arquivo de entrada, arquivo `.huff` e diretório/arquivo de destino.
  - Botões Comprimir e Descomprimir rodando em goroutine, barra de progresso atualizada via `fyne.Do`, bloqueio de controles durante operação.
  - Proteção contra sobrescrita: impedir sobrescrever a origem e pedir confirmação quando o destino já existir.
  - Exibição clara de erros (arquivo inválido, CRC incorreto, I/O).
- [ ] **4.2 Métricas e Tabela de Símbolos** (`ui/metrics.go`, `ui/table.go`)
  - Painel de resultados: tamanho original, tamanho comprimido, redução %, tempo, velocidade (MB/s), Entropia de Shannon e tamanho teórico mínimo vs. real.
  - `widget.Table`: byte (hex + caractere imprimível), frequência, comprimento do código e código binário (exibindo `"0 (implícito)"` quando houver 1 único símbolo), com ordenação por frequência.

**Critério de pronto:** Aplicação desktop executa compressão/descompressão fim a fim sem travar a UI, com métricas e tabela funcionais.

---

## Fase 5 — Experimentos e Documentação
**Objetivo:** Consolidar o estudo prático comparando compressão real vs. entropia.

- [ ] **5.1 Bateria de experimentos:** Comprimir e descomprimir TXT, BMP, WAV, PNG/JPG, MP3, binário/ZIP, bytes aleatórios, byte único repetido, arquivo vazio e recompressão de `.huff`.
- [ ] **5.2 `README.md`:** Documentar arquitetura, instruções de build (incluindo `cgo`/MinGW no Windows), uso e tabela completa com os resultados dos experimentos.

**Critério de pronto:** Todos os requisitos da "Definição de pronto" (Seção 13 da SPEC) atendidos.

---

## Fase 6 — Bônus e Extensões (Opcional / Pós-MVP)
**Objetivo:** Evoluções incrementais após a entrega principal estar estável.

- [ ] **6.1 Visualização da Árvore:** Exportação Graphviz DOT -> renderização parcial no Canvas Fyne com `container.Scroll` -> subárvores recolhíveis -> zoom.
- [ ] **6.2 Extensões de Formato/Performance:** Huffman canônico (`HUF2`), modo *stored* (fallback quando comprimido > original), lookup table na decodificação, Huffman adaptativo.