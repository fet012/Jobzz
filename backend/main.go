package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"jobz/config"
	"jobz/handlers"
	"jobz/routes"
	"jobz/middlewares"
)

func main() {
	godotenv.Load()

	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET is not set")
	}
	config.Connect()
	ensureIndexes()

	r := gin.Default()
	r.Use(middlewares.CORS())
	routes.SetupRoutes(r, config.DB)

	srv := &http.Server{
		Addr:    ":" + os.Getenv("PORT"),
		Handler: r,
	}

	go func() {
		log.Println("server listening on", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	handlers.StartExpiryWorker()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("server stopped cleanly")
}

func ensureIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := config.DB.Collection("employers").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Fatalf("employers index: %v", err)
	}

	_, err = config.DB.Collection("candidates").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Fatalf("candidates index: %v", err)
	}

	_, err = config.DB.Collection("applications").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "job_id", Value: 1}, {Key: "candidate_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Fatalf("applications unique index: %v", err)
	}

	_, err = config.DB.Collection("applications").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "candidate_id", Value: 1}},
	})
	if err != nil {
		log.Fatalf("applications candidate index: %v", err)
	}

	_, err = config.DB.Collection("jobs").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "status", Value: 1}},
	})
	if err != nil {
		log.Fatalf("jobs status index: %v", err)
	}

	log.Println("indexes ensured")
}