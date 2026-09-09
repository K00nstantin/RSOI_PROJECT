package main

import (
	"RSOI_PROJECT/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func getReport(c *gin.Context, events *[]models.Event) {
	role, exists := c.Get("role")
	if !exists || role != "Admin" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin role required"})
		return
	}

	breakdown := make(map[string]int)
	for _, e := range *events {
		breakdown[e.Action]++
	}

	c.JSON(http.StatusOK, gin.H{
		"total_events": len(*events),
		"breakdown":    breakdown,
	})
}
