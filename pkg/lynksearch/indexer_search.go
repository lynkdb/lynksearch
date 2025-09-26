package lynksearch

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"

	"github.com/lynkdb/lynkapi/go/lynkapi"
	"github.com/lynkdb/lynksearch/pkg/sorter"
	"github.com/lynkdb/lynksearch/pkg/tokenizer"
)

func (it *indexer) search(q string) *lynkapi.DataResult {

	type hitDoc struct {
		id       uint32
		uniid    string
		tokenSeq int
		tokenNum int
		score    float64
		hitScore float64
	}

	var (
		rs = lynkapi.NewDataResult()

		tokenHits = []*tokenEntry{}
		tokenDup  = map[uint32]bool{}

		ghits = []*hitDoc{}
		ghiti = map[uint32]*hitDoc{}

		tokens = tokenizer.Tokenize(q)
	)
	fmt.Println("tokens", tokens)

	it.tokenMux.RLock()
	for _, v := range tokens {
		token, ok := it.tokenIndex[v]
		if !ok || token.DocNum < 1 || tokenDup[token.Id] {
			continue
		}
		tokenDup[token.Id] = true

		tokenHits = append(tokenHits, token)
	}
	it.tokenMux.RUnlock()

	fmt.Println("token hits", tokenDup)

	sort.Slice(tokenHits, func(i, j int) bool {
		return tokenHits[i].DocNum < tokenHits[j].DocNum
	})

	var (
		limit    = 10
		batchNum = 128

		avgDocLength = 500.0
		bm25         = sorter.NewBM25(1.2, 0.75, 1000000, avgDocLength) // k1=1.2, b=0.75, N=1000, avgdl=200

	)

	it.mu.Lock()
	defer it.mu.Unlock()

	if it.tokenDocReader == nil {
		fpd, err := os.Open(it.dir + "/main.lstd")
		if err != nil {
			rs.Status = lynkapi.NewServiceStatusServerError(err.Error())
			return rs
		}
		// st, err := fpd.Stat()
		// if err != nil {
		// 	rs.Status = lynkapi.NewServiceStatusServerError(err.Error())
		// 	return rs
		// }
		it.tokenDocReader = fpd
	}

	for tokenI, token := range tokenHits {

		var (
			offset = token.DocListOffset
			value  = make([]byte, 8*batchNum)

			hits = []*hitDoc{}
			hiti = map[uint32]*hitDoc{}
		)

		fmt.Println("\nquery token", token.Value, "doc-list-offset", token.DocListOffset, "doc-num", token.DocNum)

		// for hit := 0; hit < token.DocNum && hit < (limit*min(10, len(tokenHits))); {
		for hit := uint32(0); hit < token.DocNum; {

			n, err := it.tokenDocReader.ReadAt(value, int64(offset))
			if n == 0 && err != nil {
				fmt.Println("it.tokenDocReader.ReadAt", err)
				break
			}

			batchOffset := 0

			for b := 0; b < min(batchNum, int(token.DocNum-hit)); b++ {

				hit += 1

				// doc-seq
				docseq, n0 := binary.Uvarint(value[batchOffset:])
				batchOffset += n0

				// doc-token-seq
				tokenSeq, n1 := binary.Uvarint(value[batchOffset:])
				batchOffset += n1

				// doc-token-num
				tokenNum, n2 := binary.Uvarint(value[batchOffset:])
				batchOffset += n2

				fmt.Println(" doc-list", "doc-seq", docseq, "doc-token-seq", tokenSeq, "doc-token-num", tokenNum)

				doc, ok := ghiti[uint32(docseq)]
				if !ok {
					if tokenI > 0 {
						continue
					}

					docMeta, ok := it.docIntIndex[uint32(docseq)]
					if !ok {
						continue
					}

					doc = &hitDoc{
						id:    uint32(docseq),
						uniid: docMeta.UniId,
					}
				}

				hiti[doc.id] = doc
				hits = append(hits, doc)

				doc.tokenSeq += int(tokenSeq)
				doc.tokenNum += int(tokenNum * 64)

				doc.score += bm25.Score(int(tokenSeq), int(tokenNum*64), int(token.DocNum))
				doc.hitScore += 1.0
			}

			offset += uint32(batchOffset)
		}

		ghits = hits
		ghiti = hiti
		fmt.Println(" match token", token.Value, "hits", len(hits), "ghits", len(ghits))
	}

	sort.Slice(ghits, func(i, j int) bool {
		return ghits[i].score > ghits[j].score
	})

	fmt.Println(" search docs", len(ghits))

	if len(ghits) > limit {
		ghits = ghits[:limit]
	}

	for _, v := range ghits {
		rs.Rows = append(rs.Rows, &lynkapi.DataRow{
			Id: v.uniid,
		})
	}

	return rs
}
