package stats

import (
	"ema-backend/migrations"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
)

// RegisterStubRoutes registers statistics endpoints
func RegisterStubRoutes(r *gin.Engine) {
	r.GET("/users/:id/clinical-cases-count", getClinicalCasesStats)
	r.GET("/user/:id/total-tests", getTotalTestsStats)
	r.GET("/user/:id/test-progress", getTestProgress)
	r.GET("/user/:id/most-studied-category", getMostStudiedCategory)
	r.GET("/chats/:id", getChatsStats)
}

func getClinicalCasesStats(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Param("id"))

	sub, err := migrations.GetActiveSubscriptionForUser(userID)
	if err != nil || sub == nil {
		c.JSON(200, gin.H{"total": 0, "used": 0, "remaining": 0})
		return
	}

	plan := sub["subscription_plan"].(map[string]interface{})
	planLimit := plan["clinical_cases"].(int)
	remaining := sub["clinical_cases"].(int)
	used := planLimit - remaining

	log.Printf("[STATS] Clinical Cases - UserID=%d Plan=%d Remaining=%d Used=%d", userID, planLimit, remaining, used)

	c.JSON(200, gin.H{
		"total":     planLimit,
		"used":      used,
		"remaining": remaining,
	})
}

func getTotalTestsStats(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Param("id"))

	sub, err := migrations.GetActiveSubscriptionForUser(userID)
	if err != nil || sub == nil {
		c.JSON(200, gin.H{"total": 0, "used": 0, "remaining": 0})
		return
	}

	plan := sub["subscription_plan"].(map[string]interface{})
	planLimit := plan["questionnaires"].(int)
	remaining := sub["questionnaires"].(int)
	used := planLimit - remaining

	log.Printf("[STATS] Tests/Questionnaires - UserID=%d Plan=%d Remaining=%d Used=%d", userID, planLimit, remaining, used)

	c.JSON(200, gin.H{
		"total":     planLimit,
		"used":      used,
		"remaining": remaining,
	})
}

func getChatsStats(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Param("id"))

	sub, err := migrations.GetActiveSubscriptionForUser(userID)
	if err != nil || sub == nil {
		c.JSON(200, gin.H{"total": 0, "used": 0, "remaining": 0})
		return
	}

	plan := sub["subscription_plan"].(map[string]interface{})
	planLimit := plan["consultations"].(int)
	remaining := sub["consultations"].(int)
	used := planLimit - remaining

	log.Printf("[STATS] Chats/Consultations - UserID=%d Plan=%d Remaining=%d Used=%d", userID, planLimit, remaining, used)

	c.JSON(200, gin.H{
		"total":     planLimit,
		"used":      used,
		"remaining": remaining,
	})
}

func getTestProgress(c *gin.Context) {
	// Por ahora vacío - se puede implementar más adelante con una tabla de historial
	c.JSON(200, gin.H{"progress": []any{}})
}

func getMostStudiedCategory(c *gin.Context) {
	// Por ahora vacío - se puede implementar más adelante con una tabla de historial
	c.JSON(200, gin.H{"category": nil})
}
