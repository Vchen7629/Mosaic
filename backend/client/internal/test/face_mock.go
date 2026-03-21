package test

import (
	"context"
	"sync"

	"google.golang.org/grpc"

	fd "mosaic-client.com/gen/face_detection"
)

// MockFaceClient is a configurable test double for fd.FaceDetectionServiceClient.
// Set the Func fields to override the default success responses.
type MockFaceClient struct {
	Mu sync.Mutex

	ProcessVisitorFacesFunc  func(*fd.ProcessVisitorFacesRequest) (*fd.ProcessVisitorFacesResponse, error)
	RegisterVisitorFaceFunc  func(*fd.RegisterVisitorFaceRequest) (*fd.RegisterVisitorFaceResponse, error)
	SyncProfileFunc          func(*fd.SyncProfileRequest) (*fd.SyncProfileResponse, error)
	RegisterProfileFaceFunc  func(*fd.RegisterProfileFaceRequest) (*fd.RegisterProfileFaceResponse, error)
}

func (m *MockFaceClient) ProcessVisitorFaces(
	_ context.Context,
	req *fd.ProcessVisitorFacesRequest,
	_ ...grpc.CallOption,
) (*fd.ProcessVisitorFacesResponse, error) {
	m.Mu.Lock()
	fn := m.ProcessVisitorFacesFunc
	m.Mu.Unlock()

	if fn != nil {
		return fn(req)
	}
	return &fd.ProcessVisitorFacesResponse{FaceDetected: true, Success: true}, nil
}

func (m *MockFaceClient) RegisterVisitorFace(
	_ context.Context,
	req *fd.RegisterVisitorFaceRequest,
	_ ...grpc.CallOption,
) (*fd.RegisterVisitorFaceResponse, error) {
	m.Mu.Lock()
	fn := m.RegisterVisitorFaceFunc
	m.Mu.Unlock()

	if fn != nil {
		return fn(req)
	}
	return &fd.RegisterVisitorFaceResponse{Success: true}, nil
}

func (m *MockFaceClient) SyncProfile(
	_ context.Context,
	req *fd.SyncProfileRequest,
	_ ...grpc.CallOption,
) (*fd.SyncProfileResponse, error) {
	m.Mu.Lock()
	fn := m.SyncProfileFunc
	m.Mu.Unlock()

	if fn != nil {
		return fn(req)
	}
	return &fd.SyncProfileResponse{FaceDetected: true, Success: true}, nil
}

func (m *MockFaceClient) RegisterProfileFace(
	_ context.Context,
	req *fd.RegisterProfileFaceRequest,
	_ ...grpc.CallOption,
) (*fd.RegisterProfileFaceResponse, error) {
	m.Mu.Lock()
	fn := m.RegisterProfileFaceFunc
	m.Mu.Unlock()

	if fn != nil {
		return fn(req)
	}
	return &fd.RegisterProfileFaceResponse{Success: true}, nil
}
