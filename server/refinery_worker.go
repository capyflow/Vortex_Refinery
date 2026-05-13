package server

import (
	"github.com/capyflow/allspark-go/logx"
	vhttp "github.com/capyflow/vortexv3/server/http"

	"Vortex_Refinery/pkg/types"
)

// ListWorkers 获取 Worker 列表
func (s *RefineryServer) ListWorkers(ctx *vhttp.Context) error {
	gctx := ctx.Context()
	coll := s.store.Collection("worker_registry")

	cursor, err := coll.Find(gctx, map[string]interface{}{})
	if err != nil {
		logx.Errorf("RefineryServer|ListWorkers|Find|Error|%v", err)
		return ctx.JsonResponse(vhttp.Codes.InternalError, nil)
	}
	defer cursor.Close(gctx)

	var workers []types.WorkerRegistry
	if err := cursor.All(gctx, &workers); err != nil {
		logx.Errorf("RefineryServer|ListWorkers|All|Error|%v", err)
		return ctx.JsonResponse(vhttp.Codes.InternalError, nil)
	}
	return ctx.JsonResponse(vhttp.Codes.Success, workers)
}
