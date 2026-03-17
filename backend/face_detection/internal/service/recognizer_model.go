package service

import (
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

func (p *RecognizerPool) Acquire() *face.Recognizer    { return <-p.recPool }
func (p *RecognizerPool) Release(rec *face.Recognizer) { p.recPool <- rec }
func (p *RecognizerPool) Close() {
	close(p.recPool)
	for rec := range p.recPool {
		rec.Close()
	}
}
