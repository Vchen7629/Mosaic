//go:build integration

package test

import (
	"context"
	"sync"

	"google.golang.org/grpc"
	at "mosaic-client.com/gen/audio_transcription"
	cb "mosaic-client.com/gen/conversation_briefing"
	fd "mosaic-client.com/gen/face_detection"
)

type StubFaceClient struct {
	Mu                    sync.Mutex
	ProcessVisitorCalls   []*fd.ProcessVisitorFacesRequest
	SyncProfileCalled     int
	RegisterVisitorCalled int
}

func (s *StubFaceClient) ProcessVisitorFaces(
	_ context.Context,
	in *fd.ProcessVisitorFacesRequest,
	_ ...grpc.CallOption,
) (*fd.ProcessVisitorFacesResponse, error) {
	s.Mu.Lock()
	s.ProcessVisitorCalls = append(s.ProcessVisitorCalls, in)
	s.Mu.Unlock()
	// facedetected: false, service returns early without writing to conn
	return &fd.ProcessVisitorFacesResponse{FaceDetected: false}, nil
}

func (s *StubFaceClient) SyncProfile(
	_ context.Context,
	_ *fd.SyncProfileRequest,
	_ ...grpc.CallOption,
) (*fd.SyncProfileResponse, error) {
	s.Mu.Lock()
	s.SyncProfileCalled++
	s.Mu.Unlock()
	return &fd.SyncProfileResponse{Success: true}, nil
}

func (s *StubFaceClient) RegisterVisitorFace(
	_ context.Context,
	_ *fd.RegisterVisitorFaceRequest,
	_ ...grpc.CallOption,
) (*fd.RegisterVisitorFaceResponse, error) {
	s.Mu.Lock()
	s.RegisterVisitorCalled++
	s.Mu.Unlock()
	return &fd.RegisterVisitorFaceResponse{Success: true}, nil
}

type StubAudioClient struct {
	Mu                   sync.Mutex
	SaveTranscriptCalled int
	SaveTranscriptErr    error
}

func (s *StubAudioClient) TranscribeAudio(
	_ context.Context,
	_ *at.TranscribeAudioRequest,
	_ ...grpc.CallOption,
) (*at.TranscribeAudioResponse, error) {
	return &at.TranscribeAudioResponse{Success: true}, nil
}

func (s *StubAudioClient) SaveTranscript(
	_ context.Context,
	_ *at.SaveTranscriptRequest,
	_ ...grpc.CallOption,
) (*at.SaveTranscriptResponse, error) {
	s.Mu.Lock()
	s.SaveTranscriptCalled++
	err := s.SaveTranscriptErr
	s.Mu.Unlock()
	if err != nil {
		return nil, err
	}
	return &at.SaveTranscriptResponse{Success: true}, nil
}

type StubBriefingClient struct {
	Mu                                 sync.Mutex
	GenerateConversationBriefingCalled int
}

func (s *StubBriefingClient) GenerateConversationBriefing(
	_ context.Context,
	_ *cb.GenerateConversationBriefingRequest,
	_ ...grpc.CallOption,
) (*cb.GenerateConversationBriefingResponse, error) {
	s.Mu.Lock()
	s.GenerateConversationBriefingCalled++
	s.Mu.Unlock()
	return &cb.GenerateConversationBriefingResponse{Success: true}, nil
}
