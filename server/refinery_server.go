package server

import (
	"context"
	"net/http"
	"reflect"

	"github.com/capyflow/allspark-go/logx"
	"github.com/gin-gonic/gin"

	vortex "github.com/capyflow/vortexv3"
	vpkg "github.com/capyflow/vortexv3/pkg"
	vhttp "github.com/capyflow/vortexv3/server/http"

	"Vortex_Refinery/conf"
	"Vortex_Refinery/internal/store"
)

// RefineryServer is the main server for Vortex Refinery
type RefineryServer struct {
	ctx        context.Context
	port       int
	cfg        *conf.Config
	store      *store.MongoStore
	e          *vortex.VortexEngine
	debugRoute func() // temp debug
}

// NewRefineryServer creates a new RefineryServer
func NewRefineryServer(ctx context.Context, cfg *conf.Config, mongoStore *store.MongoStore) *RefineryServer {
	return &RefineryServer{
		ctx:   ctx,
		port:  int(cfg.Port),
		cfg:   cfg,
		store: mongoStore,
	}
}

// Start 启动 Vortex 引擎
func (s *RefineryServer) Start() {
	rootGroup := vhttp.NewRootGroup("/v1")
	s.PrepareRouters(rootGroup)

	// 如果 port 未指定，从 config 中的 http_addr 解析
	port := s.port
	if port == 0 {
		port = 8088
	}

	e := vortex.NewVortexEngine(s.ctx,
		vortex.WithPort(port),
		vortex.WithEnableProtocol([]string{vpkg.HTTP}),
		vortex.WithHttpRouterRootGroup(rootGroup),
		vortex.WithJwtOption(nil),
	)
	s.e = e

	// Debug route - register directly on the gin engine via reflection
	rv := reflect.ValueOf(e).Elem()
	ginEngineField := rv.FieldByName("ginEngine")
	if ginEngineField.IsValid() && ginEngineField.CanInterface() {
		ginEngine := ginEngineField.Interface().(*gin.Engine)
		ginEngine.GET("/v1/debug/routes", func(c *gin.Context) {
			c.JSON(200, gin.H{"routes": ginEngine.Routes()})
		})
	}

	e.Start()
}

// PrepareRouters 注册所有路由
func (s *RefineryServer) PrepareRouters(rootGroup *vhttp.VortexHttpRouterGroup) {
	s.workflowRouters(rootGroup)
	s.pluginRouters(rootGroup)
	s.instanceRouters(rootGroup)
	s.workerRouters(rootGroup)
	s.statsRouters(rootGroup)
}

// ─── Workflow Routes ───────────────────────────────────────────────────────────

func (s *RefineryServer) workflowRouters(root *vhttp.VortexHttpRouterGroup) {
	wfRoot := root.AddGroup("/workflows")
	wfRoot.AddRouter([]string{http.MethodGet}, "", s.ListWorkflows, vhttp.WithSkipJwtVerify())
	wfRoot.AddRouter([]string{http.MethodPost}, "", s.CreateWorkflow, vhttp.WithSkipJwtVerify())
	wfRoot.AddRouter([]string{http.MethodGet}, "/:id", s.GetWorkflow, vhttp.WithSkipJwtVerify())
	wfRoot.AddRouter([]string{http.MethodPut}, "/:id", s.UpdateWorkflow, vhttp.WithSkipJwtVerify())
	wfRoot.AddRouter([]string{http.MethodDelete}, "/:id", s.DeleteWorkflow, vhttp.WithSkipJwtVerify())
	wfRoot.AddRouter([]string{http.MethodPost}, "/:id/publish", s.PublishWorkflow, vhttp.WithSkipJwtVerify())
}

// ─── Plugin Routes ─────────────────────────────────────────────────────────────

func (s *RefineryServer) pluginRouters(root *vhttp.VortexHttpRouterGroup) {
	pRoot := root.AddGroup("/plugins")
	pRoot.AddRouter([]string{http.MethodGet}, "/", s.ListPlugins, vhttp.WithSkipJwtVerify())
	pRoot.AddRouter([]string{http.MethodGet}, "/:name", s.GetPlugin, vhttp.WithSkipJwtVerify())
}

// ─── Instance Routes ───────────────────────────────────────────────────────────

func (s *RefineryServer) instanceRouters(root *vhttp.VortexHttpRouterGroup) {
	iRoot := root.AddGroup("/instances")
	iRoot.AddRouter([]string{http.MethodGet}, "/", s.ListInstances, vhttp.WithSkipJwtVerify())
	iRoot.AddRouter([]string{http.MethodGet}, "/:id", s.GetInstance, vhttp.WithSkipJwtVerify())
	iRoot.AddRouter([]string{http.MethodPost}, "/:id/retry", s.RetryInstance, vhttp.WithSkipJwtVerify())
	iRoot.AddRouter([]string{http.MethodGet}, "/:id/tasks", s.GetInstanceTasks, vhttp.WithSkipJwtVerify())
}

// ─── Worker Routes ─────────────────────────────────────────────────────────────

func (s *RefineryServer) workerRouters(root *vhttp.VortexHttpRouterGroup) {
	wRoot := root.AddGroup("/workers")
	wRoot.AddRouter([]string{http.MethodGet}, "/", s.ListWorkers, vhttp.WithSkipJwtVerify())
}

// ─── Stats Routes ──────────────────────────────────────────────────────────────

func (s *RefineryServer) statsRouters(root *vhttp.VortexHttpRouterGroup) {
	root.AddRouter([]string{http.MethodGet}, "/stats", s.GetStats, vhttp.WithSkipJwtVerify())
}

// Shutdown 优雅关闭
func (s *RefineryServer) Shutdown(ctx context.Context) error {
	logx.Infof("RefineryServer|Shutdown|Info|shutting down")
	return nil
}
