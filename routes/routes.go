package routes

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"

	"jobz/handlers"
	"jobz/middlewares"
)

func SetupRoutes(r *gin.Engine, db *mongo.Database) {
	// Public
	r.POST("/api/auth/employer/register", handlers.RegisterEmployer)
	r.POST("/api/auth/employer/login", handlers.LoginEmployer)
	r.POST("/api/auth/candidate/register", handlers.RegisterCandidate)
	r.POST("/api/auth/candidate/login", handlers.LoginCandidate)
	r.GET("/api/jobs", handlers.ListJobs)
	r.GET("/api/jobs/:id", handlers.GetJob)

	// Employer-only
	employer := r.Group("/api/employer")
	employer.Use(middlewares.Auth(), middlewares.EmployerOnly())
	{
		employer.POST("/jobs", handlers.CreateJob)
		employer.PUT("/jobs/:id", handlers.UpdateJob)
		employer.DELETE("/jobs/:id", handlers.DeleteJob)
		employer.GET("/jobs/:id/applications", handlers.GetJobApplications)
		employer.PUT("/applications/:id/status", handlers.UpdateApplicationStatus)
	}

	// Candidate-only
	candidate := r.Group("/api/candidate")
	candidate.Use(middlewares.Auth(), middlewares.CandidateOnly())
	{
		candidate.POST("/jobs/:id/apply", handlers.ApplyToJob)
		candidate.GET("/applications", handlers.GetMyApplications)
	}
}