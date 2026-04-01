package service

import (
	"context"
	"fmt"

	"github.com/Kagami/go-face"
)

type RecognizerPool struct {
	recPool chan *face.Recognizer
}

// Recognizer Pool to handle concurrent requests
func NewRecognizerPool(modelsDir string, size int) (*RecognizerPool, error) {
	ch := make(chan *face.Recognizer, size)

	for range size {
		rec, err := face.NewRecognizer(modelsDir)
		if err != nil {
			return nil, fmt.Errorf("can't init face recognizer: %w", err)
		}
		ch <- rec
	}

	return &RecognizerPool{recPool: ch}, nil
}

func (p *RecognizerPool) Acquire(ctx context.Context) (*face.Recognizer, error) {
	select {
	case rec := <-p.recPool:
		return rec, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (p *RecognizerPool) Release(rec *face.Recognizer) { p.recPool <- rec }
func (p *RecognizerPool) Close() {
	close(p.recPool)
	for rec := range p.recPool {
		rec.Close()
	}
}
