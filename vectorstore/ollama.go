package vector

import (
	"context"
	"fmt"
	"time"

	"github.com/tiamxu/kit/log"
)

func (s *ModelService) Initialize(ctx context.Context) error {
	start := time.Now()
	defer func() {
		log.Printf("Model initialization completed in %v", time.Since(start))
	}()

	// Initialize LLM and Embedder
	llm, embedder, err := s.initModels()
	if err != nil {
		return fmt.Errorf("model initialization failed: %w", err)
	}

	// Initialize Vector Store
	store, err := s.initVectorStore(ctx, embedder)
	if err != nil {
		return fmt.Errorf("vector store initialization failed: %w", err)
	}

	s.llm = llm
	s.embedder = embedder
	s.store = store

	return nil
}
