//go:build integration

package test

import (
	"context"
	"sync"

	"google.golang.org/grpc"
	at "mosaic-client.com/gen/audio_transcription"
	fd "mosaic-client.com/gen/face_detection"
)

type StubFaceClient struct {
	Mu                  sync.Mutex
	ProcessVisitorCalls []*fd.ProcessVisitorFacesRequest
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

func (s *StubFaceClient) ProcessProfileFace(
	_ context.Context,
	_ *fd.ProcessProfileFaceRequest,
	_ ...grpc.CallOption,
) (*fd.ProcessProfileFaceResponse, error) {
	return &fd.ProcessProfileFaceResponse{Success: true}, nil
}

func (s *StubFaceClient) RegisterVisitorFace(
	_ context.Context,
	_ *fd.RegisterVisitorFaceRequest,
	_ ...grpc.CallOption,
) (*fd.RegisterVisitorFaceResponse, error) {
	return &fd.RegisterVisitorFaceResponse{Success: true}, nil
}

func (s *StubFaceClient) RegisterProfileFace(
	_ context.Context,
	_ *fd.RegisterProfileFaceRequest,
	_ ...grpc.CallOption,
) (*fd.RegisterProfileFaceResponse, error) {
	return &fd.RegisterProfileFaceResponse{Success: true}, nil
}

type StubAudioClient struct{}

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
	return &at.SaveTranscriptResponse{Success: true}, nil
}