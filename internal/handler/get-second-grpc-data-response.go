package handler

import (
	"context"
	"fmt"

	cam_pt "github.com/j7248855-web/orsa-vision-grpc-third/gen/cam"
	"github.com/jmoiron/sqlx"
)

type Server struct {
	cam_pt.UnimplementedCameraControlServer
	DB *sqlx.DB
}

// Отправка ответ на третий микросервис в виде IP камеры и других метаданных
func (s *Server) StartStreamSession(ctx context.Context, req *cam_pt.StreamConfig) (*cam_pt.StreamConfig, error) {
	busID := req.GetBusId()

	var ip, deviceType string
	err := s.DB.QueryRow("SELECT rtsp_link, type FROM devices WHERE bus_id = $1", busID).Scan(&ip, &deviceType)
	if err != nil {
		return nil, fmt.Errorf("device not found")
	}

	return &cam_pt.StreamConfig{
		BusId:     busID,
		CameraIp:  ip,
		SessionId: req.SessionId,
	}, nil
}
