package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Employer struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Company   string             `bson:"company" json:"company"`
	Email     string             `bson:"email" json:"email"`
	Password  string             `bson:"password" json:"-"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}