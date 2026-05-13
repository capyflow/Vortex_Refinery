package server

import (
	"time"

	vhttp "github.com/capyflow/vortexv3/server/http"

	"Vortex_Refinery/internal/store"
	"Vortex_Refinery/model"
	"Vortex_Refinery/pkg/types"
)

// GetStats 获取 Dashboard 统计数据
func (s *RefineryServer) GetStats(ctx *vhttp.Context) error {
	gctx := ctx.Context()

	wfStore := store.NewWorkflowStore(s.store)
	instStore := store.NewInstanceStore(s.store)
	workerColl := s.store.Collection("worker_registry")

	// Count plugins (deduplicate across workers)
	plugins := 0
	{
		cursor, _ := workerColl.Find(gctx, map[string]interface{}{})
		var workers []types.WorkerRegistry
		if cursor != nil {
			cursor.All(gctx, &workers)
			cursor.Close(gctx)
		}
		seen := make(map[string]bool)
		for _, worker := range workers {
			for _, p := range worker.Plugins {
				if !seen[p] {
					seen[p] = true
					plugins++
				}
			}
		}
	}

	wfList, _ := wfStore.List(gctx)
	instances, _ := instStore.ListByStatus(gctx, "running")

	// Count today's triggered
	today := time.Now().Truncate(24 * time.Hour)
	running := 0
	for _, i := range instances {
		if !i.CreatedAt.Before(today) {
			running++
		}
	}

	return ctx.JsonResponse(vhttp.Codes.Success, model.StatsResp{
		TotalPlugins:     plugins,
		TotalWorkflows:  len(wfList),
		RunningInstances: len(instances),
		TriggeredToday:   running,
	})
}
