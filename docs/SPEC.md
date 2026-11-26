# Huffman Compressor — Especificação do Projeto (v2)

*Versão revisada para um projeto de estudo de 1–2 dias · Go + Fyne · formato `.huff`*

## 1. Objetivo

Desenvolver uma aplicação desktop capaz de comprimir e descomprimir arquivos de qualquer formato utilizando uma implementação própria do algoritmo de Huffman.

O projeto tem como principais objetivos:

- Estudar e implementar o algoritmo de compressão de Huffman.
- Trabalhar diretamente com dados binários e manipulação de bits.
- Desenvolver um formato próprio de arquivo comprimido.
- Compreender estruturas de dados, especialmente árvores binárias e filas de prioridade.
- Desenvolver uma interface gráfica simples para usar a ferramenta e visualizar o funcionamento do algoritmo.
- Comparar os resultados da compressão com diferentes tipos de arquivos.

**Prazo estimado:** 1–2 dias.

**Princípio de escopo:** a aplicação é desenvolvida completa, sem versões intermediárias específicas para texto ou outros formatos (o compressor sempre trabalha com bytes). Porém, o desenvolvimento segue camadas: primeiro o núcleo do algoritmo (com testes), depois o formato, depois a interface. A visualização gráfica da árvore é um item de bônus e não bloqueia a entrega (ver seção 11).

---

## 2. Tecnologias

| Componente | Tecnologia |
| --- | --- |
| Linguagem | Go |
| Interface gráfica | Fyne |
| Manipulação de arquivos | Biblioteca padrão (`os`, `bufio`, `io`, `path/filepath`) |
| Serialização do cabeçalho | `encoding/binary` (little-endian, `PutUvarint`) |
| Checksum | `hash/crc32` |
| Testes | `testing` do Go, incluindo fuzzing nativo (`go test -fuzz`) |
| Formato de saída | `.huff` |
| Algoritmo | Huffman estático, implementado do zero |

### Justificativa

Go oferece uma boa combinação entre manipulação de dados binários, operações bitwise, estruturas de dados e facilidade de desenvolvimento. Fyne permite desenvolver uma aplicação desktop inteiramente em Go, incluindo desenho personalizado pelo Canvas.

### Observações sobre o Fyne

- O Fyne usa cgo, então é necessário ter um compilador C instalado (no Windows, por exemplo, MinGW-w64 via MSYS2). Vale configurar isso e compilar um “hello world” antes de qualquer outra coisa, porque o primeiro build é lento.
- Operações pesadas (comprimir, descomprimir) devem rodar em goroutine. Atualizações de interface vindas de goroutines devem usar `fyne.Do` nas versões recentes do Fyne (2.6 em diante). Confirmar na documentação da versão utilizada.
- O `container.Scroll` permite rolagem, mas o Fyne não oferece zoom pronto. Zoom é tratado como extensão opcional.

---

## 3. Estrutura do projeto

```
huffcompressor/
├── go.mod
├── main.go                  # ponto de entrada (inicia a GUI)
├── internal/
│   └── huffman/             # núcleo, sem dependência de GUI
│       ├── freq.go          # contagem de frequências
│       ├── heap.go          # Min Heap
│       ├── tree.go          # construção da árvore
│       ├── codes.go         # geração dos códigos
│       ├── bitio.go         # BitWriter e BitReader
│       ├── format.go        # cabeçalho .huff (leitura, escrita, validação)
│       ├── encoder.go       # compressão
│       ├── decoder.go       # descompressão
│       ├── entropy.go       # entropia de Shannon (métrica de estudo)
│       └── *_test.go        # testes unitários e fuzz
└── ui/                      # interface Fyne
```

O pacote `huffman` não importa nada de Fyne. Isso permite testar o algoritmo sem abrir a interface e deixa a GUI como uma camada fina por cima.

---

## 4. Funcionamento do algoritmo

O compressor trabalha diretamente com bytes, independentemente do formato ou extensão do arquivo original. Cada byte (0 a 255) é um símbolo independente.

### Processo de compressão

1. Abrir o arquivo original em modo binário.
2. **Primeira passagem:** calcular a frequência de cada byte, o tamanho total e o CRC32 dos dados originais.
3. Construir a árvore de Huffman usando uma fila de prioridade (Min Heap).
4. Gerar os códigos binários de cada símbolo.
5. Escrever o cabeçalho `.huff` (que depende dos resultados da primeira passagem).
6. **Segunda passagem:** reler o arquivo original, codificar cada byte e gravar os bits com o BitWriter.
7. Fazer o flush final, completando o último byte com bits zero.

### Processo de descompressão

1. Abrir o arquivo `.huff`.
2. Ler e validar o cabeçalho (seção 6.3).
3. Reconstruir a árvore de Huffman a partir da tabela de frequências, usando exatamente o mesmo algoritmo e o mesmo desempate da compressão.
4. Decodificar os bits com o BitReader percorrendo a árvore até produzir `Original Size` bytes.
5. Gravar os bytes em um arquivo temporário no diretório de destino.
6. Comparar o CRC32 dos bytes gravados com o do cabeçalho.
7. Se o CRC bater, renomear o temporário para o nome final. Se não bater, apagar o temporário e reportar erro.

### Estratégia de leitura

A abordagem de duas passagens evita carregar o arquivo inteiro na memória: toda leitura e escrita usa `bufio` com buffers fixos. O núcleo recebe um callback opcional de progresso (`func(done, total int64)`) para a interface atualizar uma barra de progresso.

---

## 5. Componentes e decisões de implementação

| Componente | Responsabilidade |
| --- | --- |
| Frequency Counter | Calcular a frequência dos 256 possíveis bytes, o tamanho total e o CRC32. |
| Min Heap | Selecionar os nós de menor frequência. |
| Huffman Tree | Construir a árvore. |
| Code Generator | Gerar os códigos binários dos símbolos. |
| BitWriter | Escrever códigos de tamanho variável agrupando bits em bytes. |
| BitReader | Ler individualmente os bits dos dados comprimidos. |
| Encoder | Orquestrar as duas passagens e a escrita do arquivo. |
| Decoder | Reconstruir os dados originais e validar a integridade. |
| Entropy | Calcular a entropia de Shannon para comparar com o resultado real. |

### 5.1 Frequências

Usar um array fixo `[256]uint64`, nunca um `map`. A iteração sobre `map` em Go tem ordem aleatória, o que quebraria o determinismo da construção da árvore.

### 5.2 Min Heap e desempate determinístico

A regra de desempate precisa ser idêntica na compressão e na descompressão:

- As folhas são inseridas na heap em ordem crescente de byte.
- Cada nó (folha ou interno) recebe um número de sequência `seq` crescente no momento da criação.
- A heap ordena por `(freq, seq)`.
- Ao juntar dois nós, o primeiro removido da heap vira o filho esquerdo (bit `0`) e o segundo vira o filho direito (bit `1`). O nó interno criado recebe o próximo `seq`.

Como o decoder reconstrói a árvore com o mesmo código, o determinismo é garantido por construção. O heap pode ser escrito à mão (cerca de 30 linhas) ou feito com `container/heap`. Como o princípio do projeto é implementar do zero, a sugestão é escrever o próprio heap.

### 5.3 Geração de códigos e comprimentos acima de 64 bits

Os códigos são gerados por percurso em profundidade (DFS) a partir da raiz. Cada código é armazenado como uma estrutura com o comprimento em bits e os bits em um slice de bytes (MSB primeiro), sem depender de um inteiro de 64 bits.

Na prática, um código acima de 64 bits exige frequências próximas de uma sequência de Fibonacci e arquivos na casa de dezenas de terabytes, então nunca vai acontecer com arquivos reais. Ainda assim, o código deve suportar o caso, e ele é testado de forma sintética (seção 10). O decoder não é afetado, pois apenas percorre a árvore.

Otimização opcional: caminho rápido com `uint64` quando o comprimento for 64 bits ou menos, e caminho lento com slice apenas para os raros códigos maiores.

### 5.4 BitWriter e BitReader

- Ordem dos bits: MSB primeiro dentro de cada byte.
- O BitWriter acumula bits em um buffer de 64 bits e descarrega bytes completos em um `bufio.Writer`. Evitar escrever bit a bit direto no disco.
- O BitReader lê byte a byte e devolve um bit por vez. Quando faltam dados antes de completar `Original Size` bytes, retorna `io.ErrUnexpectedEOF`.
- Os bits de preenchimento do último byte são sempre zero e nunca são interpretados, pois o decoder para ao atingir `Original Size`.

---

## 6. Formato próprio de arquivo: `.huff`

### 6.1 Convenções

- Todos os inteiros multi-byte usam little-endian.
- A versão do formato está embutida no Magic Number (`HUF1`). Uma versão futura usaria `HUF2`, e o decoder rejeita qualquer magic desconhecido. Não existe campo de versão separado.
- A extensão `.huff` é uma convenção do projeto. A identificação real é feita pelo Magic Number.

### 6.2 Estrutura (versão 1)

| Campo | Tipo | Descrição |
| --- | --- | --- |
| Magic Number | 4 bytes | `HUF1` |
| Original Size | uint64 | Tamanho original do arquivo em bytes. |
| Filename Length | uint16 | Tamanho do nome original em bytes. |
| Filename | UTF-8 | Nome original, incluindo extensão (somente o nome, sem diretórios). |
| Checksum | uint32 | CRC32 dos dados originais. |
| Symbol Count | uint16 | Quantidade de símbolos diferentes (0 a 256). |
| Frequency Table | variável | Para cada símbolo, em ordem crescente de byte: símbolo (1 byte) + frequência (uvarint). |
| Compressed Data | variável | Bits dos códigos Huffman, agrupados em bytes, MSB primeiro. |

A tabela guarda apenas os símbolos presentes no arquivo. As frequências usam `binary.PutUvarint`, o que reduz bastante o cabeçalho em relação a um `uint64` fixo por entrada. O Symbol Count é `uint16` porque o valor 256 não cabe em um byte.

`Original Size` define quando a decodificação termina, evitando confundir o preenchimento do último byte com dados válidos.

### 6.3 Validações na leitura

O decoder deve rejeitar o arquivo, sem travar nem consumir memória em excesso, quando:

- O Magic Number for diferente de `HUF1`.
- O Symbol Count for maior que 256.
- Os símbolos da tabela não estiverem em ordem estritamente crescente (isso também elimina duplicados).
- Alguma frequência for zero.
- A soma das frequências (com checagem de overflow) for diferente de `Original Size`.
- O nome tiver tamanho zero, ou contiver separadores de caminho, ou for `.` ou `..`. Ao extrair, aplicar `filepath.Base` ao nome armazenado para impedir que um `.huff` malicioso escreva fora da pasta de destino (por exemplo `../../algo`).
- O arquivo terminar antes de `Original Size` bytes serem decodificados.
- O CRC32 final não bater (o arquivo temporário é apagado).

---

## 7. Casos especiais e cuidados

| Caso | Tratamento |
| --- | --- |
| Arquivo vazio | Symbol Count = 0, Original Size = 0, sem dados. CRC32 de entrada vazia é 0. Compressão e descompressão devem funcionar. |
| Um único símbolo distinto | A árvore é uma folha só (código de 0 bits). Symbol Count = 1 e nenhum dado comprimido é gravado. O decoder repete o símbolo `Original Size` vezes. Na tabela da interface, exibir o código como “0 (implícito)”. |
| Arquivo pequeno | Os metadados podem superar a economia, e o resultado pode ser maior que o original. Isso é esperado e deve aparecer nas métricas. |
| Todos os 256 bytes presentes | Cabeçalho no tamanho máximo. Deve funcionar normalmente. |
| Códigos acima de 64 bits | Suportados pela estrutura de código (seção 5.3) e testados de forma sintética. |
| Preenchimento do último byte | Bits zero, nunca interpretados. |
| Frequências iguais | Desempate por `seq` (seção 5.2). |
| Magic Number inválido | Erro claro, sem tentar decodificar. |
| Arquivo corrompido ou incompleto | Validações da seção 6.3, com `io.ErrUnexpectedEOF` e erro de CRC. |
| Sobrescrita acidental | Nunca gravar sobre o arquivo de origem. Se o destino já existir, pedir confirmação na interface. Gravar sempre em arquivo temporário e renomear no final. |

---

## 8. Interface gráfica

Interface desktop simples com Fyne. O objetivo é disponibilizar as funcionalidades principais e permitir enxergar o funcionamento do algoritmo, não criar algo sofisticado.

### 8.1 Essencial (necessário para considerar o projeto pronto)

**Seleção de arquivos**

- Selecionar o arquivo a comprimir.
- Selecionar um arquivo `.huff` para descomprimir.
- Escolher o destino dos arquivos gerados.

**Operações**

- Botão de comprimir e botão de descomprimir.
- Operações em goroutine, com barra de progresso e interface sem travar.
- Mensagens de erro claras (arquivo inválido, CRC incorreto, destino já existente).

**Resultados**

- Tamanho original e tamanho comprimido.
- Percentual de redução.
- Tempo de compressão e de descompressão.
- Velocidade de processamento em MB/s.
- Entropia de Shannon e o tamanho teórico mínimo, ao lado do tamanho real.

**Tabela de símbolos** (`widget.Table`)

- Byte (em hexadecimal e, quando imprimível, o caractere), frequência, comprimento do código e código binário.
- Ordenável por frequência.

### 8.2 Bônus: visualização gráfica da árvore

A árvore de um arquivo binário pode ter até 511 nós, e desenhar tudo com objetos de Canvas é pesado no Fyne. Estratégias, da mais simples para a mais completa:

1. Exportar a árvore em formato Graphviz DOT (cerca de 20 linhas de código) e abrir em uma ferramenta externa.
2. Desenhar na interface apenas os primeiros N níveis, ou apenas os símbolos mais frequentes, dentro de um `container.Scroll`.
3. Subárvores recolhíveis (clicar em um nó para expandir ou recolher).
4. Zoom (extensão opcional, pois o Fyne não oferece pronto).

A árvore deve mostrar a frequência de cada nó e permitir ver o caminho binário até cada folha.

---

## 9. Métricas e experimentos

### Fórmulas

- **Taxa de compressão:** `(tamanho comprimido / tamanho original) × 100`
- **Redução de tamanho:** `100 − taxa de compressão`
- **Velocidade:** `tamanho original em MB / tempo em segundos`
- **Entropia de Shannon:** `H = − Σ p(b) · log2 p(b)`, em bits por byte, onde `p(b)` é a frequência relativa do byte `b`.
- **Tamanho teórico mínimo:** `H × tamanho original / 8` bytes.

O tamanho comprimido é sempre o tamanho total do arquivo `.huff`, incluindo o cabeçalho. A diferença entre o tamanho real e o teórico mostra o custo do cabeçalho e a perda por trabalhar com códigos de comprimento inteiro.

### Experimentos sugeridos

| Tipo | Expectativa |
| --- | --- |
| TXT | Boa redução (distribuição de bytes bem desigual). |
| BMP | Depende da imagem, geralmente alguma redução. |
| WAV | Redução modesta. |
| PNG e JPG | Pouca ou nenhuma redução, pois já estão comprimidos. |
| MP3 | Pouca ou nenhuma redução. |
| Outros binários (executáveis, `.zip`) | Varia. |
| Arquivo de bytes aleatórios | Sem redução, e o resultado fica maior por causa do cabeçalho. |
| Arquivo com um único byte repetido | Redução quase total. |
| Arquivo vazio | Apenas o cabeçalho. |
| `.huff` comprimido novamente | Compressão sucessiva: sem ganho esperado, pois os dados já estão próximos da entropia máxima. |

Registrar os resultados (tamanhos, tempos, MB/s, entropia) em uma tabela no README do projeto.

---

## 10. Testes

O núcleo é testado antes de existir interface.

**Unitários**

- Heap: sempre retorna o menor elemento, respeita o desempate por `seq`.
- Árvore e códigos: o conjunto de códigos é livre de prefixo, e a soma de `2^(−comprimento)` das folhas é igual a 1 (desigualdade de Kraft com igualdade).
- Determinismo: a mesma tabela de frequências gera sempre os mesmos códigos.
- BitWriter e BitReader: ida e volta com sequências de comprimentos variados e com preenchimento no final.
- **Códigos acima de 64 bits:** gerar códigos a partir de uma tabela sintética com cerca de 80 símbolos de frequências de Fibonacci (a soma cabe em `uint64`) e verificar que o código mais longo passa de 64 bits e que a decodificação percorrendo a árvore funciona. Não é possível produzir um arquivo real para esse caso.

**Ida e volta (compressão seguida de descompressão deve devolver bytes idênticos)**

- Arquivo vazio, de 1 byte, de um único símbolo repetido, com os 256 bytes, de bytes aleatórios e de texto.

**Fuzzing**

- `FuzzRoundTrip`: para entradas arbitrárias, `decode(encode(x)) == x`. Cobre a maior parte dos casos especiais da seção 7 automaticamente.
- `FuzzDecode`: o decoder recebendo bytes arbitrários nunca deve entrar em pânico nem travar, apenas retornar erro.

**Arquivos inválidos**

- Magic Number errado, arquivo truncado, bit alterado nos dados (deve falhar no CRC), tabela com símbolos fora de ordem, soma de frequências incorreta e nome com `../`.

---

## 11. Plano de execução

Estimativas para orientar a ordem do trabalho, dentro do prazo de 1–2 dias.

| Bloco | Conteúdo | Estimativa |
| --- | --- | --- |
| 0 | Ambiente: Go, compilador C, Fyne com um “hello world” compilando. | 0,5 h |
| 1 | Frequências, heap, árvore, códigos e seus testes. | 3–4 h |
| 2 | BitWriter/BitReader, formato `.huff`, encoder, decoder, testes de ida e volta e fuzz. | 3–4 h |
| 3 | Interface essencial: seleção de arquivos, botões, progresso, métricas e tabela de símbolos. | 3–4 h |
| 4 | Entropia, experimentos da seção 9 e README com os resultados. | 2–3 h |
| Bônus | Visualização da árvore (seção 8.2) e extensões da seção 12. | o que sobrar |

**Ponto de controle:** ao final do bloco 2 o núcleo já deve comprimir e descomprimir qualquer arquivo de forma correta. Se o tempo apertar, a interface essencial (bloco 3) ainda pode ser entregue sem a árvore gráfica.

---

## 12. Extensões opcionais (depois que tudo funcionar)

- **Huffman canônico (`HUF2`):** guardar apenas os comprimentos dos códigos (no máximo 256 bytes) em vez das frequências. O cabeçalho fica menor e o problema do desempate desaparece, pois os códigos são atribuídos por uma regra ordenada.
- **Modo “stored”:** se o resultado comprimido for maior que o original, gravar os dados sem compressão, com um byte de flag no cabeçalho.
- **Decodificação por tabela de consulta (lookup)** em vez de percorrer a árvore bit a bit, para aumentar a velocidade em MB/s.
- **Zoom** na visualização da árvore.
- Processamento por blocos com adaptação ao longo do arquivo (Huffman adaptativo).

---

## 13. Escopo final

O resultado esperado é uma aplicação desktop funcional que consiga:

1. Comprimir arquivos de qualquer formato usando Huffman.
2. Descomprimir arquivos gerados pela própria aplicação.
3. Recuperar o nome e a extensão originais com segurança (sem escrever fora do destino).
4. Verificar a integridade dos dados recuperados com CRC32.
5. Apresentar métricas de compressão e desempenho, incluindo a comparação com a entropia.
6. Exibir a tabela de símbolos (byte, frequência, código) e, se houver tempo, a árvore de Huffman.

**Definição de pronto:** testes e fuzz passando, os experimentos da seção 9 registrados e a interface essencial (seção 8.1) funcionando.

**Princípio do projeto:** implementar o algoritmo de Huffman do zero, sem bibliotecas externas de compressão. Bibliotecas de interface gráfica e recursos auxiliares de leitura, escrita e checksum estão permitidos.

O foco é desenvolver uma implementação compreensível, correta e testada, que permita estudar tanto os aspectos teóricos quanto práticos da compressão de dados.
