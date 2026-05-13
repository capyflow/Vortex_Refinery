//go:build ignore

package main

import (
	"fmt"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	// Test path.Join behavior with /{id}
	// (we can't import path here easily, so inline it)

	// Simulate vortex AddRouter logic:
	// relativePath = "/workflows"
	// routerPath = "/{id}"
	// fullPath = path.Join("/workflows", "/{id}")
	// In Go: path.Join strips leading slashes and joins properly

	// The actual test: does Gin handle /workflows/{id} correctly?
	e := gin.New()
	e.GET("/workflows/{id}", func(c *gin.Context) {
		fmt.Printf("  -> handler called with id=%s\n", c.Param("id"))
		c.JSON(200, gin.H{"id": c.Param("id")})
	})
	e.GET("/workflows", func(c *gin.Context) {
		c.JSON(200, gin.H{"list": true})
	})

	fmt.Println("Registered routes:")
	for _, r := range e.Routes() {
		fmt.Printf("  %s %s\n", r.Method, r.Path)
	}

	tests := []string{"/workflows", "/workflows/abc-123"}
	for _, t := range tests {
		req := httptest.NewRequest("GET", t, nil)
		w := httptest.NewRecorder()
		e.ServeHTTP(w, req)
		fmt.Printf("GET %s -> %d: %s\n", t, w.Code, w.Body.String())
	}
}
