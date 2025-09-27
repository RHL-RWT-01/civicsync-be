package models

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Vote represents a user's vote on an issue
type Vote struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Issue     primitive.ObjectID `bson:"issue" json:"issue"`
	User      primitive.ObjectID `bson:"user" json:"user"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}

// EnsureVoteIndex creates a unique compound index for (issue, user)
func EnsureVoteIndex(collection *mongo.Collection) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "issue", Value: 1}, {Key: "user", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	return err
}
