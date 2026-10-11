package handlers

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"jobz/config"
	"jobz/models"
)

// StartExpiryWorker runs an expiry sweep immediately, then every hour.
// It runs in a goroutine and returns immediately, so it doesn't block main.
func StartExpiryWorker() {
	go func() {
		// Run once immediately so a fresh deploy doesn't wait an hour.
		runExpiryCheck()

		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			runExpiryCheck()
		}
	}()
}

func runExpiryCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := config.DB.Collection("jobs").UpdateMany(
		ctx,
		bson.M{
			"deadline": bson.M{"$lt": time.Now()},
			"status":   bson.M{"$ne": models.JobStatusExpired},
		},
		bson.M{
			"$set": bson.M{
				"status":     models.JobStatusExpired,
				"updated_at": time.Now(),
			},
		},
	)
	if err != nil {
		log.Printf("expiry worker: %v", err)
		return
	}
	if result.ModifiedCount > 0 {
		log.Printf("expiry worker: marked %d job(s) as expired", result.ModifiedCount)
	}
}