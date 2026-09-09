package main

import (
	"RSOI_PROJECT/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// getReport возвращает агрегированную статистику (только для Admin)
func getReport(c *gin.Context, events *[]models.Event) {
	// Проверка роли (устанавливается в AuthMiddleware)
	role, exists := c.Get("role")
	if !exists || role != "Admin" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin role required"})
		return
	}

	// Агрегация: количество событий по типу действия
	breakdown := make(map[string]int)
	for _, e := range *events {
		breakdown[e.Action]++
	}

	// Можно добавить более детальный отчёт, например, по пользователям
	c.JSON(http.StatusOK, gin.H{
		"total_events": len(*events),
		"breakdown":    breakdown,
		// "user_activity": ... (дополнительно)
	})
}
