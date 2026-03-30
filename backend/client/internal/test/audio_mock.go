package test

import (
	"context"
	"sync"

	"google.golang.org/grpc"

	at "mosaic-client.com/gen/audio_transcription"
)

// MockAudioClient is a configurable test double for at.AudioTranscriptionServiceClient.
// Set TranscribeFunc/SaveFunc to override the default success responses.
type MockAudioClient struct {
	Mu              sync.Mutex
	TranscribeCalls int
	SaveCalls       int

	TranscribeFunc func(*at.TranscribeAudioRequest) (*at.TranscribeAudioResponse, error)
	SaveFunc       func(*at.SaveTranscriptRequest) (*at.SaveTranscriptResponse, error)
}

func (m *MockAudioClient) TranscribeAudio(
	_ context.Context,
	req *at.TranscribeAudioRequest,
	_ ...grpc.CallOption,
) (*at.TranscribeAudioResponse, error) {
	m.Mu.Lock()
	m.TranscribeCalls++
	fn := m.TranscribeFunc
	m.Mu.Unlock()

	if fn != nil {
		return fn(req)
	}
	return &at.TranscribeAudioResponse{Success: true}, nil
}

func (m *MockAudioClient) SaveTranscript(
	_ context.Context,
	req *at.SaveTranscriptRequest,
	_ ...grpc.CallOption,
) (*at.SaveTranscriptResponse, error) {
	m.Mu.Lock()
	m.SaveCalls++
	fn := m.SaveFunc
	m.Mu.Unlock()

	if fn != nil {
		return fn(req)
	}
	return &at.SaveTranscriptResponse{Success: true}, nil
}
