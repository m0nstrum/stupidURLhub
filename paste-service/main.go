package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"paste-service/config"
	"paste-service/internal/api"
	"paste-service/internal/cache"
	"paste-service/internal/clients/sluggen"
	"paste-service/internal/clients/tagger"
	"paste-service/internal/model"
	"paste-service/internal/service"
	"paste-service/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfg := config.LoadFromEnv()

	if cfg.Server.TestMode {
		runTestServer(cfg)
	} else {
		runProductionServer(cfg)
	}
}

func runTestServer(cfg *config.Config) {
	log.Println("Запуск сервера в тестовом режиме...")

	gin.SetMode(gin.DebugMode)

	cacheInstance := cache.NewInMemoryCache(cfg.Cache.RefreshTTLOnGet)

	mockTagger := tagger.NewMockClient([]string{"test", "mock"}, nil)
	mockSluggen := sluggen.NewMockClient("test-slug", nil)

	mockRepo := &repository.PasteRepository{
		DB:    nil,
		Cache: cacheInstance,
	}

	pasteService := service.NewPasteService(mockRepo, mockTagger, mockSluggen)

	handler := api.NewHandler(pasteService)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	startServer(srv, cfg.Server.ShutdownTimeout)
}

func runProductionServer(cfg *config.Config) {
	setupLogger()

	db, err := setupDatabase(cfg)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	if err := db.AutoMigrate(&model.Paste{}); err != nil {
		log.Fatalf("Ошибка миграции базы данных: %v", err)
	}

	cacheInstance := setupCache(cfg)

	repo := repository.NewPasteRepository(db, cacheInstance, cfg.Cache.DefaultTTL)

	taggerClient := setupTaggerClient(cfg)
	sluggenClient, err := setupSluggenClient(cfg)
	if err != nil {
		log.Printf("Предупреждение: ошибка подключения к генератору slug: %v", err)
		sluggenClient = sluggen.NewMockClient("generated-slug", nil)
	}
	defer func() {
		if client, ok := sluggenClient.(*sluggen.GRPCClient); ok {
			client.Close()
		}
	}()

	pasteService := service.NewPasteService(repo, taggerClient, sluggenClient)

	handler := api.NewHandler(pasteService)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	startServer(srv, cfg.Server.ShutdownTimeout)
}

func startServer(srv *http.Server, shutdownTimeout time.Duration) {
	go func() {
		log.Printf("Сервер запущен на порту %s", srv.Addr[1:])
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Получен сигнал остановки, завершение работы...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Ошибка при остановке сервера: %v", err)
	}

	log.Println("Сервер остановлен")
}

func setupLogger() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Логгер настроен")
}

func setupDatabase(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, err
	}

	log.Println("Подключение к базе данных установлено")
	return db, nil
}

func setupCache(cfg *config.Config) cache.Cache {
	if cfg.Cache.Type == "redis" {
		// TODO: redis
		log.Println("Redis кэш пока не реализован, используется in-memory кэш")
	}

	log.Println("Используется in-memory кэш")
	return cache.NewInMemoryCache(cfg.Cache.RefreshTTLOnGet)
}

func setupTaggerClient(cfg *config.Config) tagger.TaggerClient {
	taggerConfig := tagger.Config{
		BaseURL:     cfg.Tagger.BaseURL,
		Timeout:     cfg.Tagger.Timeout,
		MaxTextSize: cfg.Tagger.MaxTextSize,
	}

	return tagger.NewHTTPClient(taggerConfig)
}

func setupSluggenClient(cfg *config.Config) (sluggen.SlugClient, error) {
	sluggenConfig := sluggen.Config{
		Address:     cfg.SlugGen.Address,
		Timeout:     cfg.SlugGen.Timeout,
		MaxTextSize: cfg.SlugGen.MaxTextSize,
	}

	client, err := sluggen.NewGRPCClient(sluggenConfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}
