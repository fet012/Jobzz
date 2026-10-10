package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Application struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	JobID       primitive.ObjectID `bson:"job_id" json:"job_id"`
	CandidateID primitive.ObjectID `bson:"candidate_id" json:"candidate_id"`
	Status      string             `bson:"status" json:"status"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

const (
	StatusPending  = "pending"
	StatusReviewed = "reviewed"
	StatusAccepted = "accepted"
	StatusRejected = "rejected"
)