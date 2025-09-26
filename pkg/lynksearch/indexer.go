package lynksearch

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lynkdb/lynksearch/pkg/tokenizer"
)

/**

 */

const (
	maxIndexDocumentNum int = 100000

	maxTokensInDocument int = 65535
	indexBlockSize      int = 32 << 20
)

type indexer struct {
	mu sync.RWMutex

	dir string

	tokenMux      sync.RWMutex
	tokenIndex    map[string]*tokenEntry
	tokenIntIndex map[uint32]*tokenEntry
	tokenArray    []*tokenEntry

	docMux      sync.RWMutex
	docIndex    map[string]*docMetaEntry
	docIntIndex map[uint32]*docMetaEntry
	docNum      int64

	tokenDocBlockFile         *os.File
	tokenDocBlockBuffer       *bufio.Writer
	tokenDocBlockBufferOffset int

	blockSize int

	tokenDocReader *os.File
}

type docMetaEntry struct {
	Id       uint32
	UniId    string
	TokenNum uint32
}

type tokenEntry struct {
	Id    uint32
	Value string

	DocNum        uint32
	DocListOffset uint32

	// offset, length
	docBlockOffset [][2]uint32

	// doc-Id, token-freq, token-num
	docBlockBuffer [][3]uint32
}

func (it *indexer) index(docid string, textFields []string) error {

	var (
		tokens    = tokenizer.Tokenize(strings.Join(textFields, ","))
		tokenFreq = make(map[string]uint32)
	)

	// 计算词频和文档长度
	for _, token := range tokens {
		tokenFreq[token]++
	}

	it.docMux.Lock()
	docMeta, ok := it.docIndex[docid]
	if !ok {
		docMeta = &docMetaEntry{
			Id:    uint32(len(it.docIndex) + 1),
			UniId: docid,
		}
		it.docIndex[docid] = docMeta
		it.docIntIndex[docMeta.Id] = docMeta

		it.docNum += 1
	}
	it.docMux.Unlock()

	for _, v := range textFields {
		it.blockSize += len(v)
	}

	if len(tokens) > maxTokensInDocument {
		docMeta.TokenNum = uint32(maxTokensInDocument / 64)
	} else {
		docMeta.TokenNum = uint32(len(tokens) / 64)
	}

	it.tokenMux.Lock()
	for tokenValue, freq := range tokenFreq {

		if freq > 255 {
			freq = 255
		}

		token, ok := it.tokenIndex[tokenValue]
		if !ok {
			token = &tokenEntry{
				Id:    uint32(len(it.tokenArray)) + 1,
				Value: tokenValue,
			}
			it.tokenIntIndex[token.Id] = token
			it.tokenIndex[token.Value] = token
			it.tokenArray = append(it.tokenArray, token)
		}

		token.docBlockBuffer = append(token.docBlockBuffer, [3]uint32{
			docMeta.Id, freq, docMeta.TokenNum,
		})
	}
	it.tokenMux.Unlock()

	if it.blockSize >= indexBlockSize {
		if err := it.flushBlock(); err != nil {
			return err
		}
		it.blockSize = 0
	}

	return nil
}

func (it *indexer) flushBlock() error {

	//
	if it.tokenDocBlockFile == nil {

		fp, err := os.OpenFile(it.dir+"/main.lstd.blk", os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return err
		}

		it.tokenDocBlockFile = fp

		it.tokenDocBlockFile.Seek(0, 0)
		it.tokenDocBlockFile.Truncate(0)

		it.tokenDocBlockBuffer = bufio.NewWriter(it.tokenDocBlockFile)
		it.tokenDocBlockBufferOffset = 0
	}

	{
		var (
			offset        = it.tokenDocBlockBufferOffset
			docNum uint32 = 0

			varValue  = make([]byte, 3*8)
			varLength = 0
		)

		for _, t := range it.tokenArray {

			if len(t.docBlockBuffer) == 0 {
				continue
			}

			varLength = 0
			for _, doc := range t.docBlockBuffer {

				// #0 doc-id
				n := binary.PutUvarint(varValue, uint64(doc[0]))

				// #1 token-freq
				n += binary.PutUvarint(varValue[n:], uint64(doc[1]))

				// #2 doc-token-num
				n += binary.PutUvarint(varValue[n:], uint64(doc[2]))

				it.tokenDocBlockBuffer.Write(varValue[:n])

				varLength += n
			}

			{
				t.docBlockOffset = append(t.docBlockOffset, [2]uint32{
					uint32(offset), uint32(varLength),
				})
				t.DocNum += uint32(len(t.docBlockBuffer))
			}

			docNum += uint32(len(t.docBlockBuffer))

			offset += varLength
			it.tokenDocBlockBufferOffset += varLength

			t.docBlockBuffer = nil
		}

		it.tokenDocBlockBuffer.Flush()

		slog.Info("block flush index", "doc", docNum)

	}

	return nil
}

func (it *indexer) flush() error {

	it.mu.Lock()
	defer it.mu.Unlock()

	if err := it.flushBlock(); err != nil {
		return err
	}

	if err := it.flushTokenList(); err != nil {
		return err
	}

	if it.tokenDocBlockFile == nil {
		return nil
	}
	it.tokenDocBlockBuffer.Flush()

	if it.tokenDocReader != nil {
		it.tokenDocReader.Close()
		it.tokenDocReader = nil
	}

	openFile := func(file string) (*os.File, *bufio.Writer, error) {

		fp, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return nil, nil, err
		}

		fp.Seek(0, 0)
		fp.Truncate(0)

		buf := bufio.NewWriter(fp)

		return fp, buf, nil
	}

	t := time.Now()

	ifp, ibuf, err := openFile(fmt.Sprintf("%s/main.lsi", it.dir))
	if err != nil {
		return err
	}
	defer func() {
		ibuf.Flush()
		ifp.Close()
	}()

	tdfp, tdbuf, err := openFile(fmt.Sprintf("%s/main.lstd", it.dir))
	if err != nil {
		return err
	}
	defer func() {
		tdbuf.Flush()
		tdfp.Close()
	}()

	var (
		varValue      = make([]byte, 3*8)
		docNum        = int64(0)
		docListOffset = uint64(0)
	)

	for _, t := range it.tokenArray {

		// index #0 token-id
		n := binary.PutUvarint(varValue, uint64(t.Id))
		ibuf.Write(varValue[:n])

		// index #0 doc-num
		n = binary.PutUvarint(varValue, uint64(t.DocNum))
		ibuf.Write(varValue[:n])

		// index #0 doc-list-offset
		n = binary.PutUvarint(varValue, docListOffset)
		ibuf.Write(varValue[:n])
		t.DocListOffset = uint32(docListOffset)

		for _, blk := range t.docBlockOffset {

			siz := int(blk[1])
			val := make([]byte, siz)

			n, err := it.tokenDocBlockFile.ReadAt(val, int64(blk[0]))
			if err != nil {
				return err
			}
			if n != siz {
				return errors.New("invalid block size")
			}

			tdbuf.Write(val)

			docListOffset += uint64(siz)
		}

		docNum += int64(t.DocNum)
	}

	slog.Info("flush index", "doc", docNum,
		"token", len(it.tokenArray),
		"time", time.Since(t))

	return nil
}

func (it *indexer) flushTokenList() error {
	//

	t := time.Now()

	fp, err := os.OpenFile(it.dir+"/main.lst", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer fp.Close()

	fp.Seek(0, 0)
	fp.Truncate(0)

	// fp.WriteByte(indexVersion)

	for _, t := range it.tokenArray {
		fp.Write([]byte(fmt.Sprintf("%d,%s\n", t.Id, t.Value)))
	}

	slog.Info("flush token list", "num", len(it.tokenArray),
		"time", time.Since(t))

	fp.Sync()

	return nil
}
