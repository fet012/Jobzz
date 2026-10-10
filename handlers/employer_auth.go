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

// Input shapes for the two endpoints.
// These are separate from models.Employer because the API accepts
// a subset of fields — never the ID, never CreatedAt, etc.

type RegisterEmployerInput struct {
	Company  string `json:"company" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginEmployerInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RegisterEmployer creates a new employer account.
// POST /api/auth/employer/register
func RegisterEmployer(c *gin.Context) {
	var input RegisterEmployerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	collection := config.DB.Collection("employers")

	// Check if email already exists
	var existing models.Employer
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

	employer := models.Employer{
		Company:   strings.TrimSpace(input.Company),
		Email:     input.Email,
		Password:  hashed,
		CreatedAt: time.Now(),
	}

	result, err := collection.InsertOne(context.TODO(), employer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create account"})
		return
	}

	// Set the generated ID back on the struct so the response has it
	employer.ID = result.InsertedID.(primitive.ObjectID)

	// Sign a token immediately so the client is logged in after registering
	token, err := utils.SignToken(employer.ID.Hex(), "employer")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not sign token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Account created",
		"token":   token,
		"employer": gin.H{
			"id":      employer.ID.Hex(),
			"company": employer.Company,
			"email":   employer.Email,
		},
	})
}

// LoginEmployer authenticates an employer and returns a JWT.
// POST /api/auth/employer/login
func LoginEmployer(c *gin.Context) {
	var input LoginEmployerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	var employer models.Employer
	err := config.DB.Collection("employers").
		FindOne(context.TODO(), bson.M{"email": input.Email}).
		Decode(&employer)

	if err == mongo.ErrNoDocuments {
		
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if err := utils.CheckPassword(input.Password, employer.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	token, err := utils.SignToken(employer.ID.Hex(), "employer")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not sign token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"employer": gin.H{
			"id":      employer.ID.Hex(),
			"company": employer.Company,
			"email":   employer.Email,
		},
	})
} 