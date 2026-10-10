package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"jobz/config"
	"jobz/middlewares"
	"jobz/models"
)

type CreateJobInput struct {
	Title       string    `json:"title" binding:"required"`
	Description string    `json:"description" binding:"required"`
	Location    string    `json:"location" binding:"required"`
	SalaryMin   int       `json:"salary_min" binding:"required,gte=0"`
	SalaryMax   int       `json:"salary_max" binding:"required,gte=0"`
	JobType     string    `json:"job_type" binding:"required,oneof=remote onsite hybrid"`
	Deadline    time.Time `json:"deadline" binding:"required"`
}

type UpdateJobInput struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	SalaryMin   int       `json:"salary_min"`
	SalaryMax   int       `json:"salary_max"`
	JobType     string    `json:"job_type" binding:"omitempty,oneof=remote onsite hybrid"`
	Deadline    time.Time `json:"deadline"`
}

// CreateJob — employer only.
// POST /api/employer/jobs
func CreateJob(c *gin.Context) {
	employerIDStr := c.GetString(middlewares.CtxUserID)
	employerID, err := primitive.ObjectIDFromHex(employerIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	var input CreateJobInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.SalaryMax < input.SalaryMin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "salary_max must be greater than or equal to salary_min"})
		return
	}

	if input.Deadline.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Deadline must be in the future"})
		return
	}

	now := time.Now()
	job := models.Job{
		Title:       strings.TrimSpace(input.Title),
		Description: strings.TrimSpace(input.Description),
		Location:    strings.TrimSpace(input.Location),
		SalaryMin:   input.SalaryMin,
		SalaryMax:   input.SalaryMax,
		JobType:     input.JobType,
		Deadline:    input.Deadline,
		EmployerID:  employerID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	result, err := config.DB.Collection("jobs").InsertOne(context.TODO(), job)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create job"})
		return
	}

	job.ID = result.InsertedID.(primitive.ObjectID)
	c.JSON(http.StatusCreated, gin.H{"job": job})
}

// ListJobs — public. Supports search + filters.
// GET /api/jobs?location=Lagos&job_type=remote&min_salary=500000&q=engineer
func ListJobs(c *gin.Context) {
	filter := bson.M{}

	if loc := c.Query("location"); loc != "" {
		filter["location"] = bson.M{"$regex": loc, "$options": "i"}
	}
	if jt := c.Query("job_type"); jt != "" {
		filter["job_type"] = jt
	}
	if minS := c.Query("min_salary"); minS != "" {
		if n, err := strconv.Atoi(minS); err == nil {
			filter["salary_max"] = bson.M{"$gte": n}
		}
	}
	if q := c.Query("q"); q != "" {
		filter["$or"] = []bson.M{
			{"title": bson.M{"$regex": q, "$options": "i"}},
			{"description": bson.M{"$regex": q, "$options": "i"}},
		}
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := config.DB.Collection("jobs").Find(context.TODO(), filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch jobs"})
		return
	}
	defer cursor.Close(context.TODO())

	jobs := []models.Job{}
	if err := cursor.All(context.TODO(), &jobs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not parse jobs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"jobs": jobs, "count": len(jobs)})
}

// GetJob — public.
// GET /api/jobs/:id
func GetJob(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	var job models.Job
	err = config.DB.Collection("jobs").FindOne(context.TODO(), bson.M{"_id": id}).Decode(&job)
	if err == mongo.ErrNoDocuments {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"job": job})
}

// UpdateJob — employer only, and only the employer who posted it.
// PUT /api/employer/jobs/:id
func UpdateJob(c *gin.Context) {
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

	var existing models.Job
	err = config.DB.Collection("jobs").FindOne(context.TODO(), bson.M{"_id": jobID}).Decode(&existing)
	if err == mongo.ErrNoDocuments {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// BUSINESS RULE: only the employer who posted the job can modify it.
	if existing.EmployerID != employerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only modify jobs you posted"})
		return
	}

	var input UpdateJobInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := bson.M{"updated_at": time.Now()}
	if input.Title != ""       { update["title"] = strings.TrimSpace(input.Title) }
	if input.Description != "" { update["description"] = strings.TrimSpace(input.Description) }
	if input.Location != ""    { update["location"] = strings.TrimSpace(input.Location) }
	if input.SalaryMin != 0    { update["salary_min"] = input.SalaryMin }
	if input.SalaryMax != 0    { update["salary_max"] = input.SalaryMax }
	if input.JobType != ""     { update["job_type"] = input.JobType }
	if !input.Deadline.IsZero() { update["deadline"] = input.Deadline }

	_, err = config.DB.Collection("jobs").UpdateOne(
		context.TODO(),
		bson.M{"_id": jobID},
		bson.M{"$set": update},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update job"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job updated"})
}

// DeleteJob — employer only, and only the employer who posted it.
// DELETE /api/employer/jobs/:id
func DeleteJob(c *gin.Context) {
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

	var existing models.Job
	err = config.DB.Collection("jobs").FindOne(context.TODO(), bson.M{"_id": jobID}).Decode(&existing)
	if err == mongo.ErrNoDocuments {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if existing.EmployerID != employerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete jobs you posted"})
		return
	}

	_, err = config.DB.Collection("jobs").DeleteOne(context.TODO(), bson.M{"_id": jobID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete job"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job deleted"})
}