package main

import (
	"context"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	mu      sync.Mutex
	devices map[uuid.UUID]Device
}

func NewHandler() *Handler {
	return &Handler{devices: map[uuid.UUID]Device{}}
}

func (h *Handler) RegisterDevice(ctx context.Context, request RegisterDeviceRequestObject) (RegisterDeviceResponseObject, error) {
	device := Device{
		DeviceUuid:   uuid.New(),
		Name:         request.Body.Name,
		SerialNumber: request.Body.SerialNumber,
	}

	h.mu.Lock()
	h.devices[device.DeviceUuid] = device
	h.mu.Unlock()

	return RegisterDevice201JSONResponse{DeviceUuid: device.DeviceUuid}, nil
}

func (h *Handler) GetDevice(ctx context.Context, request GetDeviceRequestObject) (GetDeviceResponseObject, error) {
	h.mu.Lock()
	device, ok := h.devices[request.DeviceUuid]
	h.mu.Unlock()

	if !ok {
		return GetDevice404JSONResponse{Message: "device not found"}, nil
	}

	return GetDevice200JSONResponse(device), nil
}

func main() {
	e := echo.New()

	handler := NewHandler()
	RegisterHandlers(e, NewStrictHandler(handler, nil))

	log.Fatal(e.Start(":8080"))
}
