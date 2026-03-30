package service

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	fd "mosaic-client.com/gen/face_detection"
)

// Helper method to convert an embedding string to []float32
func ParseEmbedding(embString string) ([]float32, error) {
	embString = strings.TrimSpace(embString)
	embString = strings.TrimPrefix(embString, "[")
	embString = strings.TrimSuffix(embString, "]")

	parts := strings.Split(embString, ",")
	res := make([]float32, len(parts))
	for i, p := range parts {
		val, err := strconv.ParseFloat(strings.TrimSpace(p), 32)
		if err != nil {
			return nil, fmt.Errorf("invalid embedding val at index %d: %w", i, err)
		}
		res[i] = float32(val)
	}
	return res, nil
}

// iterates over detected faces and writes the appropriate json response to frontend for each one
func processFaceResults(logger *slog.Logger, faces []*fd.FaceResult, conn *SafeConn) {
	for _, faceData := range faces {
		if !faceData.IsKnown {
			err := conn.WriteJSON(UnknownVisitorResponse{
				Type:          "new_visitor_register",
				FaceEmbedding: faceData.FaceEmbedding,
			})
			if err != nil {
				logger.Error(
					"error sending request to register new visitor face to frontend",
					"err", err,
				)
			}
			// only send data to the frontend if the visitor hasnt
			// been sent to the frontend.
		} else if !HasSeenVisitor(faceData.VisitorId) {
			logger.Debug("[ProcessVisitorFace] This visitor briefing hasnt been sent to frontend, sending")
			err := conn.WriteJSON(KnownVisitorResponse{
				Type:        "visitor_briefing_response",
				VisitorName: faceData.Name,
				Briefing:    faceData.Briefing,
				VisitorID:   faceData.VisitorId,
			})
			if err != nil {
				logger.Error("error sending visitor briefing to frontend", "err", err)
			}

			AddSeenVisitor(faceData.VisitorId)
		}
	}
}
