package stats

import "github.com/gin-gonic/gin"

// RegisterStubRoutes registers placeholder endpoints for future statistics so the frontend avoids 404s.
func RegisterStubRoutes(r *gin.Engine) {
	// All return simple empty payloads (200) to prevent client errors.
	r.GET("/users/:id/clinical-cases-count", func(c *gin.Context) { c.JSON(200, gin.H{"count": 0}) })
	r.GET("/user/:id/total-tests", func(c *gin.Context) { c.JSON(200, gin.H{"total": 0}) })
	r.GET("/user/:id/test-progress", func(c *gin.Context) { c.JSON(200, gin.H{"progress": []any{}}) })
	r.GET("/user/:id/most-studied-category", func(c *gin.Context) { c.JSON(200, gin.H{"category": nil}) })
	r.GET("/chats/:id", func(c *gin.Context) { c.JSON(200, gin.H{"messages": []any{}}) })
}
