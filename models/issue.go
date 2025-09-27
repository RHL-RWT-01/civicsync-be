package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// IssueCategory enum
type IssueCategory string

const (
	Road        IssueCategory = "Road"
	Water       IssueCategory = "Water"
	Sanitation  IssueCategory = "Sanitation"
	Electricity IssueCategory = "Electricity"
	Other       IssueCategory = "Other"
)

// IssueStatus enum
type IssueStatus string

const (
	Pending    IssueStatus = "Pending"
	InProgress IssueStatus = "In Progress"
	Resolved   IssueStatus = "Resolved"
)

// Issue represents a civic issue reported by a user
type Issue struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	Category    IssueCategory      `bson:"category" json:"category"`
	Location    string             `bson:"location" json:"location"`
	ImageURL    *string            `bson:"imageUrl,omitempty" json:"imageUrl,omitempty"`
	Status      IssueStatus        `bson:"status" json:"status"`
	CreatedBy   primitive.ObjectID `bson:"createdBy" json:"createdBy"`
	Longitude   *float64           `bson:"longitude,omitempty" json:"longitude,omitempty"`
	Latitude    *float64           `bson:"latitude,omitempty" json:"latitude,omitempty"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
}
