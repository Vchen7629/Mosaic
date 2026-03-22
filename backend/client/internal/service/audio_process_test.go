//go:build unit

package service_test

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"log/slog"
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"mosaic-client.com/internal/service"
	"mosaic-client.com/internal/test"

	at "mosaic-client.com/gen/audio_transcription"
)

const testBatchSize = 40000 // sampleRate (16000) * batchDuration (2.5), matches batchSize in audio_grpc_requests.go

// encodeFloat32 converts float32 samples to a base64 string using little-endian
// IEEE 754 encoding, matching the format expected by ProcessAudio.
func encodeFloat32(samples []float32) string {
	buf := make([]byte, len(samples)*4)
	for i, s := range samples {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(s))
	}
	return base64.StdEncoding.EncodeToString(buf)
}

// makeSamples returns n samples all set to value.
func makeSamples(n int, value float32) []float32 {
	s := make([]float32, n)
	for i := range s {
		s[i] = value
	}
	return s
}

// drainBuffer flushes any leftover samples from the package-level buffer so
// tests don't bleed state into one another.
func drainBuffer(t *testing.T) {
	t.Helper()
	_ = service.FlushAudio(slog.Default(), context.Background(), "1", &test.MockAudioClient{})
}

func TestProcessAudio(t *testing.T) {
	t.Run("Returns decode error on invalid base64", func(t *testing.T) {
		t.Cleanup(func() { drainBuffer(t) })
		client := &test.MockAudioClient{}

		err := service.ProcessAudio(slog.Default(), "not-valid-base64!!!", "1", client)

		assert.ErrorContains(t, err, "audio decode error", "invalid base64 should return a decode error")
		assert.Equal(t, 0, client.TranscribeCalls, "gRPC should not be called on decode error")
	})

	t.Run("Returns nil for silent audio without calling gRPC", func(t *testing.T) {
		t.Cleanup(func() { drainBuffer(t) })
		client := &test.MockAudioClient{}

		// 0.01 is below silenceThreshold (0.02)
		silent := encodeFloat32(makeSamples(100, 0.01))
		err := service.ProcessAudio(slog.Default(), silent, "1", client)

		assert.NoError(t, err, "silent audio should return nil")
		assert.Equal(t, 0, client.TranscribeCalls, "gRPC should not be called for silent audio")
	})

	t.Run("Returns nil for too loud audio without calling gRPC", func(t *testing.T) {
		t.Cleanup(func() { drainBuffer(t) })
		client := &test.MockAudioClient{}

		// 0.8 is above loudThreshold (0.5)
		loud := encodeFloat32(makeSamples(100, 0.8))
		err := service.ProcessAudio(slog.Default(), loud, "1", client)

		assert.NoError(t, err, "too loud audio should return nil")
		assert.Equal(t, 0, client.TranscribeCalls, "gRPC should not be called for too loud audio")
	})

	t.Run("Does not call gRPC when buffer is below batch size", func(t *testing.T) {
		t.Cleanup(func() { drainBuffer(t) })
		client := &test.MockAudioClient{}

		// 100 samples is well below batchSize (40000)
		audio := encodeFloat32(makeSamples(100, 0.1))
		err := service.ProcessAudio(slog.Default(), audio, "1", client)

		assert.NoError(t, err, "valid audio below batch size should return nil")
		assert.Equal(t, 0, client.TranscribeCalls, "gRPC should not be called before batch size is reached")
	})

	t.Run("Calls TranscribeAudio once when buffer reaches batch size", func(t *testing.T) {
		t.Cleanup(func() { drainBuffer(t) })
		client := &test.MockAudioClient{}

		audio := encodeFloat32(makeSamples(testBatchSize, 0.1))
		err := service.ProcessAudio(slog.Default(), audio, "1", client)

		assert.NoError(t, err, "valid audio at batch size should return nil")
		assert.Equal(t, 1, client.TranscribeCalls, "TranscribeAudio should be called exactly once at batch size")
	})

	t.Run("Sends correct profile ID in TranscribeAudio request", func(t *testing.T) {
		t.Cleanup(func() { drainBuffer(t) })

		var capturedID int32
		client := &test.MockAudioClient{
			TranscribeFunc: func(req *at.TranscribeAudioRequest) (*at.TranscribeAudioResponse, error) {
				capturedID = req.ProfileId
				return &at.TranscribeAudioResponse{Success: true}, nil
			},
		}

		audio := encodeFloat32(makeSamples(testBatchSize, 0.1))
		err := service.ProcessAudio(slog.Default(), audio, "42", client)

		assert.NoError(t, err)
		assert.Equal(t, int32(42), capturedID, "profile ID should be forwarded to TranscribeAudio")
	})

	t.Run("Propagates non-retryable gRPC error from TranscribeAudio", func(t *testing.T) {
		t.Cleanup(func() { drainBuffer(t) })

		client := &test.MockAudioClient{
			TranscribeFunc: func(_ *at.TranscribeAudioRequest) (*at.TranscribeAudioResponse, error) {
				return nil, status.Error(codes.Internal, "internal error")
			},
		}

		audio := encodeFloat32(makeSamples(testBatchSize, 0.1))
		err := service.ProcessAudio(slog.Default(), audio, "1", client)

		assert.ErrorContains(t, err, "error processing audio", "gRPC error should be propagated")
	})

	t.Run("Concurrent writes do not produce data races", func(t *testing.T) {
		t.Cleanup(func() { drainBuffer(t) })
		client := &test.MockAudioClient{}

		// Small payload each call so no batch fires and complicates the assertion.
		audio := encodeFloat32(makeSamples(100, 0.1))

		const goroutines = 20
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for range goroutines {
			go func() {
				defer wg.Done()
				err := service.ProcessAudio(slog.Default(), audio, "1", client)
				assert.NoError(t, err)
			}()
		}

		wg.Wait()
	})
}

func TestFlushAudio(t *testing.T) {
	t.Run("Returns nil and skips gRPC for audio less than 1 second", func(t *testing.T) {
		t.Cleanup(func() { drainBuffer(t) })
		client := &test.MockAudioClient{}

		// Pre-fill buffer with sub-second audio (below batchSize so ProcessAudio
		// does not send on its own).
		audio := encodeFloat32(makeSamples(16000-1, 0.1))
		assert.NoError(t, service.ProcessAudio(slog.Default(), audio, "1", client))

		err := service.FlushAudio(slog.Default(), context.Background(), "1", client)

		assert.NoError(t, err, "sub-second remaining audio should return nil")
		assert.Equal(t, 0, client.TranscribeCalls, "TranscribeAudio should not be called for sub-second audio")
	})

	t.Run("Calls TranscribeAudio for audio at least 1 second", func(t *testing.T) {
		t.Cleanup(func() { drainBuffer(t) })
		client := &test.MockAudioClient{}

		// Exactly 1 second — below batchSize so ProcessAudio does not send,
		// but FlushAudio should.
		audio := encodeFloat32(makeSamples(16000, 0.1))
		assert.NoError(t, service.ProcessAudio(slog.Default(), audio, "1", client))
		assert.Equal(t, 0, client.TranscribeCalls, "setup: ProcessAudio must not have sent yet")

		err := service.FlushAudio(slog.Default(), context.Background(), "1", client)

		assert.NoError(t, err)
		assert.Equal(t, 1, client.TranscribeCalls, "TranscribeAudio should be called for >= 1s of remaining audio")
	})

	t.Run("Clears buffer so a second flush does not re-send", func(t *testing.T) {
		t.Cleanup(func() { drainBuffer(t) })
		client := &test.MockAudioClient{}

		audio := encodeFloat32(makeSamples(16000, 0.1))
		assert.NoError(t, service.ProcessAudio(slog.Default(), audio, "1", client))
		assert.NoError(t, service.FlushAudio(slog.Default(), context.Background(), "1", client))

		callsAfterFirstFlush := client.TranscribeCalls

		err := service.FlushAudio(slog.Default(), context.Background(), "1", client)

		assert.NoError(t, err)
		assert.Equal(t, callsAfterFirstFlush, client.TranscribeCalls, "second flush on empty buffer should not call TranscribeAudio")
	})

	t.Run("Propagates gRPC error from TranscribeAudio", func(t *testing.T) {
		t.Cleanup(func() { drainBuffer(t) })

		client := &test.MockAudioClient{
			TranscribeFunc: func(_ *at.TranscribeAudioRequest) (*at.TranscribeAudioResponse, error) {
				return nil, status.Error(codes.Unavailable, "unavailable")
			},
		}

		audio := encodeFloat32(makeSamples(16000, 0.1))
		assert.NoError(t, service.ProcessAudio(slog.Default(), audio, "1", client))

		err := service.FlushAudio(slog.Default(), context.Background(), "1", client)

		assert.ErrorContains(t, err, "error flushing audio", "gRPC error should be propagated")
	})

	t.Run("Concurrent ProcessAudio and FlushAudio calls do not produce data races", func(t *testing.T) {
		t.Cleanup(func() { drainBuffer(t) })
		client := &test.MockAudioClient{}
		audio := encodeFloat32(makeSamples(100, 0.1))

		const goroutines = 10
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for range goroutines {
			go func() {
				defer wg.Done()
				_ = service.ProcessAudio(slog.Default(), audio, "1", client)
			}()
		}

		// FlushAudio races against the appending goroutines, real-world
		// scenario when the user clicks "stop recording".
		_ = service.FlushAudio(slog.Default(), context.Background(), "1", client)

		wg.Wait()
	})
}
