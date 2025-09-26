package sorter

import "math"

// BM25 calculates the BM25 score for a document given a query.
type BM25 struct {
	k1    float64 // BM25 parameter k1 (term frequency scaling)
	b     float64 // BM25 parameter b (length normalization)
	N     int     // Total number of documents
	avgdl float64 // Average document length
}

// NewBM25 creates a new BM25 instance.
func NewBM25(k1, b float64, totalDocs int, avgDocLength float64) *BM25 {
	return &BM25{
		k1:    k1,
		b:     b,
		N:     totalDocs,
		avgdl: avgDocLength,
	}
}

// IDF calculates the Inverse Document Frequency for a term.
func (bm *BM25) IDF(docFreq int) float64 {
	numerator := float64(bm.N-docFreq) + 0.5
	denominator := float64(docFreq) + 0.5
	return math.Log(numerator/denominator + 1)
}

// Score calculates the BM25 score for a document.
func (bm *BM25) Score(termFreq, docLength, docFreq int) float64 {
	tf := float64(termFreq)
	dl := float64(docLength)
	idf := bm.IDF(docFreq)

	// BM25 formula
	numerator := tf * (bm.k1 + 1)
	denominator := tf + bm.k1*(1-bm.b+bm.b*(dl/bm.avgdl))
	return idf * numerator / denominator
}
