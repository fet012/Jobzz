package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Job struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	Location    string             `bson:"location" json:"location"`
	SalaryMin   int                `bson:"salary_min" json:"salary_min"`
	SalaryMax   int                `bson:"salary_max" json:"salary_max"`
	JobType     string             `bson:"job_type" json:"job_type"`
	Deadline    time.Time          `bson:"deadline" json:"deadline"`
	EmployerID  primitive.ObjectID `bson:"employer_id" json:"employer_id"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}