package server

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/capyflow/allspark-go/logx"

	vortex "github.com/capyflow/vortexv3"
	vpkg "github.com/capyflow/vortexv3/pkg"
	vhttp "github.com/capyflow/vortexv3/server/http"

	"Vortex_Refinery/internal/bus"
	"Vortex_Refinery/pkg/types"
)

type AdapterServer struct {
	ctx       context.Context
	port      int
	eventBus  *bus.EventBus
	e         *vortex.VortexEngine
}

func NewAdapterServer(ctx context.Context, port int, eb *bus.EventBus) *AdapterServer {
	return &AdapterServer{
		ctx:      ctx,
		port:     port,
		eventBus: eb,
	}
}

func (s *AdapterServer) Start() {
	rootGroup := vhttp.NewRootGroup("/")
	s.prepareRouters(rootGroup)

	e := vortex.NewVortexEngine(s.ctx,
		vortex.WithPort(s.port),
		vortex.WithEnableProtocol([]string{vpkg.HTTP}),
		vortex.WithHttpRouterRootGroup(rootGroup),
		vortex.WithJwtOption(nil),
	)
	s.e = e
	e.Start()
}

func (s *AdapterServer) prepareRouters(root *vhttp.VortexHttpRouterGroup) {
	root.AddRouter([]string{"POST"}, "/webhook", s.HandleWebhook, vhttp.WithSkipJwtVerify())
	root.AddRouter([]string{"GET"}, "/health", s.HandleHealth, vhttp.WithSkipJwtVerify())
}

func (s *AdapterServer) HandleWebhook(ctx *vhttp.Context) error {
	var event types.Event
	if err := ctx.UnmarshalBody(&event); err != nil {
		var payload map[string]interface{}
		if err := ctx.UnmarshalBody(&payload); err != nil {
			return ctx.JsonResponse(vhttp.Codes.BadRequest, map[string]string{"error": "invalid json"})
		}
		event.Payload = payload
	}

	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	gctx := ctx.Context()
	if err := s.eventBus.PushEvent(gctx, &event); err != nil {
		logx.Errorf("AdapterServer|HandleWebhook|PushEvent|Error|%v", err)
		return ctx.JsonResponse(vhttp.Codes.InternalError, map[string]string{"error": "failed to process event"})
	}

	return ctx.JsonResponse(vhttp.Code{Code: 202, MsgKey: "accepted"}, map[string]string{
		"event_id": event.EventID,
		"status":   "accepted",
	})
}

func (s *AdapterServer) HandleHealth(ctx *vhttp.Context) error {
	return ctx.JsonResponse(vhttp.Codes.Success, map[string]string{"status": "healthy"})
}

func (s *AdapterServer) Shutdown(ctx context.Context) error {
	logx.Infof("AdapterServer|Shutdown|Info|shutting down")
	return nil
}
