package main

import (
	"path/filepath"
	"testing"
)

func TestBundledDatasetImportsAsRawDocuments(t *testing.T) {
	files, err := datasetFiles(filepath.Join("..", "..", "..", "quality", "data", "ecommerce_agent_dataset"))
	if err != nil || len(files) != 294 {
		t.Fatalf("files=%d err=%v", len(files), err)
	}
	faqCount := 0
	for _, file := range files {
		product, err := readProduct(file)
		if err != nil {
			t.Fatal(err)
		}
		documents := buildKnowledge(product)
		if len(documents) != 1+len(product.RAGKnowledge.OfficialFAQ)+len(product.RAGKnowledge.UserReviews) {
			t.Fatalf("raw knowledge lost for %s", file)
		}
		for _, doc := range documents {
			if doc.ProductID != product.ProductID || doc.ChunkCount != 0 || doc.Content == "" {
				t.Fatalf("invalid raw document: %+v", doc)
			}
		}
		faqCount += len(product.RAGKnowledge.OfficialFAQ)
	}
	if faqCount != 827 {
		t.Fatalf("FAQ pairs=%d", faqCount)
	}
}
