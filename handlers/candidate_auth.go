package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"jobz/config"
	"jobz/models"
	"jobz/utils"
)

type RegisterCandidateInput struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginCandidateInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RegisterCandidate creates a new candidate account.
// POST /api/auth/candidate/register
func RegisterCandidate(c *gin.Context) {
	var input RegisterCandidateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	collection := config.DB.Collection("candidates")

	var existing models.Candidate
	err := collection.FindOne(context.TODO(), bson.M{"email": input.Email}).Decode(&existing)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}
	if err != mongo.ErrNoDocuments {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	hashed, err := utils.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not hash password"})
		return
	}

	candidate := models.Candidate{
		Name:      strings.TrimSpace(input.Name),
		Email:     input.Email,
		Password:  hashed,
		CreatedAt: time.Now(),
	}

	result, err := collection.InsertOne(context.TODO(), candidate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create account"})
		return
	}

	candidate.ID = result.InsertedID.(primitive.ObjectID)

	token, err := utils.SignToken(candidate.ID.Hex(), "candidate")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not sign token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Account created",
		"token":   token,
		"candidate": gin.H{
			"id":    candidate.ID.Hex(),
			"name":  candidate.Name,
			"email": candidate.Email,
		},
	})
}

// LoginCandidate authenticates a candidate and returns a JWT.
// POST /api/auth/candidate/login
func LoginCandidate(c *gin.Context) {
	var input LoginCandidateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	var candidate models.Candidate
	err := config.DB.Collection("candidates").
		FindOne(context.TODO(), bson.M{"email": input.Email}).
		Decode(&candidate)

	if err == mongo.ErrNoDocuments {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if err := utils.CheckPassword(input.Password, candidate.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	token, err := utils.SignToken(candidate.ID.Hex(), "candidate")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not sign token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"candidate": gin.H{
			"id":    candidate.ID.Hex(),
			"name":  candidate.Name,
			"email": candidate.Email,
		},
	})
}