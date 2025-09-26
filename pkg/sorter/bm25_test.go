package sorter

import (
	"testing"
)

func Test_Main(t *testing.T) {
	// Example usage
	bm25 := NewBM25(1.2, 0.75, 1000, 200.0) // k1=1.2, b=0.75, N=1000, avgdl=200

	// Example document and query term
	termFreq := 3    // Term appears 3 times in the document
	docLength := 250 // Document length (number of terms)
	docFreq := 100   // Number of documents containing the term

	score1 := bm25.Score(termFreq+1, docLength, docFreq)
	t.Logf("BM25 Score1: %.4f", score1)

	score2 := bm25.Score(termFreq, docLength, docFreq)
	t.Logf("BM25 Score2: %.4f", score2)
}
