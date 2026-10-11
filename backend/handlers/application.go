package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"jobz/config"
	"jobz/middlewares"
	"jobz/models"
)

func ApplyToJob(c *gin.Context) {
	candidateIDStr := c.GetString(middlewares.CtxUserID)
	candidateID, err := primitive.ObjectIDFromHex(candidateIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	jobID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	var job models.Job
	err = config.DB.Collection("jobs").FindOne(context.TODO(), bson.M{"_id": jobID}).Decode(&job)
	if err == mongo.ErrNoDocuments {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Primary check: stored status.
	if job.Status == models.JobStatusExpired {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This job posting has expired"})
		return
	}
	// Fallback: if the worker hasn't run yet, still reject by deadline.
	if time.Now().After(job.Deadline) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This job posting has expired"})
		return
	}

	// Business rule #1: no duplicate applications.
	var existing models.Application
	err = config.DB.Collection("applications").FindOne(
		context.TODO(),
		bson.M{"job_id": jobID, "candidate_id": candidateID},
	).Decode(&existing)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "You have already applied to this job"})
		return
	}
	if err != mongo.ErrNoDocuments {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	now := time.Now()
	application := models.Application{
		JobID:       jobID,
		CandidateID: candidateID,
		Status:      models.StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	result, err := config.DB.Collection("applications").InsertOne(context.TODO(), application)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "You have already applied to this job"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not submit application"})
		return
	}

	application.ID = result.InsertedID.(primitive.ObjectID)
	c.JSON(http.StatusCreated, gin.H{
		"message":     "Application submitted",
		"application": application,
	})
}

func GetMyApplications(c *gin.Context) {
	candidateIDStr := c.GetString(middlewares.CtxUserID)
	candidateID, err := primitive.ObjectIDFromHex(candidateIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	cursor, err := config.DB.Collection("applications").Find(
		context.TODO(),
		bson.M{"candidate_id": candidateID},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch applications"})
		return
	}
	defer cursor.Close(context.TODO())

	applications := []models.Application{}
	if err := cursor.All(context.TODO(), &applications); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not parse applications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"applications": applications, "count": len(applications)})
}

func GetJobApplications(c *gin.Context) {
	employerIDStr := c.GetString(middlewares.CtxUserID)
	employerID, err := primitive.ObjectIDFromHex(employerIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	jobID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	var job models.Job
	err = config.DB.Collection("jobs").FindOne(context.TODO(), bson.M{"_id": jobID}).Decode(&job)
	if err == mongo.ErrNoDocuments {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if job.EmployerID != employerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only view applications for jobs you posted"})
		return
	}

	cursor, err := config.DB.Collection("applications").Find(
		context.TODO(),
		bson.M{"job_id": jobID},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch applications"})
		return
	}
	defer cursor.Close(context.TODO())

	applications := []models.Application{}
	if err := cursor.All(context.TODO(), &applications); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not parse applications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"applications": applications, "count": len(applications)})
}

type UpdateStatusInput struct {
	Status string `json:"status" binding:"required,oneof=pending reviewed accepted rejected"`
}

func UpdateApplicationStatus(c *gin.Context) {
	employerIDStr := c.GetString(middlewares.CtxUserID)
	employerID, err := primitive.ObjectIDFromHex(employerIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	applicationID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application ID"})
		return
	}

	var application models.Application
	err = config.DB.Collection("applications").
		FindOne(context.TODO(), bson.M{"_id": applicationID}).
		Decode(&application)
	if err == mongo.ErrNoDocuments {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var job models.Job
	err = config.DB.Collection("jobs").
		FindOne(context.TODO(), bson.M{"_id": application.JobID}).
		Decode(&job)
	if err == mongo.ErrNoDocuments {
		c.JSON(http.StatusNotFound, gin.H{"error": "Associated job no longer exists"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if job.EmployerID != employerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update applications for jobs you posted"})
		return
	}

	var input UpdateStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = config.DB.Collection("applications").UpdateOne(
		context.TODO(),
		bson.M{"_id": applicationID},
		bson.M{"$set": bson.M{
			"status":     input.Status,
			"updated_at": time.Now(),
		}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated", "status": input.Status})
}