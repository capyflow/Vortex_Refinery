package server

import (
	"time"

	"github.com/capyflow/allspark-go/logx"
	vhttp "github.com/capyflow/vortexv3/server/http"

	"Vortex_Refinery/model"
	"Vortex_Refinery/pkg/types"
)

// ListPlugins 获取插件列表（从 Worker Registry 聚合）
func (s *RefineryServer) ListPlugins(ctx *vhttp.Context) error {
	gctx := ctx.Context()
	coll := s.store.Collection("worker_registry")

	cursor, err := coll.Find(gctx, map[string]interface{}{})
	if err != nil {
		logx.Errorf("RefineryServer|ListPlugins|Find|Error|%v", err)
		return ctx.JsonResponse(vhttp.Codes.InternalError, nil)
	}
	defer cursor.Close(gctx)

	var workers []types.WorkerRegistry
	if err := cursor.All(gctx, &workers); err != nil {
		logx.Errorf("RefineryServer|ListPlugins|All|Error|%v", err)
		return ctx.JsonResponse(vhttp.Codes.InternalError, nil)
	}

	// Deduplicate plugins from all workers
	pluginMap := make(map[string]*model.PluginResp)
	for _, worker := range workers {
		status := "unknown"
		if worker.Status == "online" {
			status = "healthy"
		}
		for _, p := range worker.Plugins {
			if _, exists := pluginMap[p]; !exists {
				pluginMap[p] = &model.PluginResp{
					Name:         p,
					Version:      "1.0.0",
					Status:       status,
					RegisteredAt: worker.RegisteredAt.Format(time.RFC3339),
					NodeTypes:    []string{p},
				}
			}
		}
	}

	plugins := make([]*model.PluginResp, 0, len(pluginMap))
	for _, p := range pluginMap {
		plugins = append(plugins, p)
	}
	return ctx.JsonResponse(vhttp.Codes.Success, plugins)
}

// GetPlugin 获取单个插件详情
func (s *RefineryServer) GetPlugin(ctx *vhttp.Context) error {
	name := ctx.GinContext().Param("name")
	gctx := ctx.Context()
	coll := s.store.Collection("worker_registry")

	cursor, err := coll.Find(gctx, map[string]interface{}{})
	if err != nil {
		logx.Errorf("RefineryServer|GetPlugin|Find|Error|%v", err)
		return ctx.JsonResponse(vhttp.Codes.BadRequest, nil)
	}
	defer cursor.Close(gctx)

	var workers []types.WorkerRegistry
	if err := cursor.All(gctx, &workers); err != nil {
		logx.Errorf("RefineryServer|GetPlugin|All|Error|%v", err)
		return ctx.JsonResponse(vhttp.Codes.BadRequest, nil)
	}

	for _, worker := range workers {
		for _, p := range worker.Plugins {
			if p == name {
				status := "unknown"
				if worker.Status == "online" {
					status = "healthy"
				}
				return ctx.JsonResponse(vhttp.Codes.Success, &model.PluginResp{
					Name:         p,
					Version:      "1.0.0",
					Status:       status,
					RegisteredAt: worker.RegisteredAt.Format(time.RFC3339),
					NodeTypes:    []string{p},
				})
			}
		}
	}
	return ctx.JsonResponse(vhttp.Codes.BadRequest, nil)
}
