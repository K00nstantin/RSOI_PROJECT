package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"RSOI_PROJECT/internal/auth"
	"RSOI_PROJECT/internal/kafka"
	"RSOI_PROJECT/internal/models"
)

func main() {
	// Переменные окружения
	kafkaBrokers := []string{os.Getenv("KAFKA_BROKERS")}
	if len(kafkaBrokers) == 0 || kafkaBrokers[0] == "" {
		kafkaBrokers = []string{"localhost:9092"}
	}
	kafkaTopic := os.Getenv("KAFKA_TOPIC")
	if kafkaTopic == "" {
		kafkaTopic = "library-events"
	}
	kafkaGroupID := "stats-group"

	// ИСПРАВЛЕНО: используем IDP_URL (как в docker-compose), а не IDENTITY_SERVICE_URL
	idpURL := os.Getenv("IDP_URL")

	if idpURL == "" {
		idpURL = "http://identity:8090"
	}

	// Хранилище событий (в памяти)
	var eventsStore []models.Event

	// Инициализация консьюмера
	consumerGroup, err := kafka.NewConsumerGroup(kafkaBrokers, kafkaGroupID, kafkaTopic)
	if err != nil {
		log.Fatalf("Failed to create consumer group: %v", err)
	}
	defer consumerGroup.Close()

	// Запуск консьюмера в фоне
	go kafka.Consume(consumerGroup, kafkaTopic, &eventsStore)

	// Инициализация аутентификации (загрузка JWKS)
	authCfg := auth.NewConfig()
	if err := authCfg.LoadJWKS(idpURL); err != nil {
		log.Fatalf("Failed to load JWKS: %v", err)
	}

	// Gin роутер
	r := gin.Default()
	r.Use(authCfg.AuthMiddleware()) // все запросы требуют токен

	// Эндпоинт для отчёта (только Admin)
	r.GET("/api/v1/stats/report", func(c *gin.Context) {
		getReport(c, &eventsStore)
	})

	// Health check
	r.GET("/manage/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "up"})
	})

	// Запуск HTTP сервера
	srv := &http.Server{
		Addr:    ":8091",
		Handler: r,
	}

	go func() {
		log.Println("Statistics service starting on :8091")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down statistics service...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
}
