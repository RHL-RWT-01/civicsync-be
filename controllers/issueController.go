package controllers

import (
	"context"
	"net/http"
	"time"

	"civicsync-be/config"
	"civicsync-be/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var issueCollection *mongo.Collection = config.GetCollection("issues")

// CreateIssue handles the creation of a new issue
func CreateIssue(c *gin.Context) {
	var issue models.Issue
	if err := c.ShouldBindJSON(&issue); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	issue.ID = primitive.NewObjectID()
	issue.CreatedAt = time.Now()
	issue.UpdatedAt = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := issueCollection.InsertOne(ctx, issue)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create issue"})
		return
	}

	c.JSON(http.StatusCreated, issue)
}

// GetIssue retrieves an issue by its ID
func GetIssue(c *gin.Context) {
	idParam := c.Param("id")
	issueID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue ID"})
		return
	}

	var issue models.Issue
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = issueCollection.FindOne(ctx, bson.M{"_id": issueID}).Decode(&issue)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Issue not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve issue"})
		}
		return
	}

	c.JSON(http.StatusOK, issue)
}

// UpdateIssue updates an existing issue by its ID
func UpdateIssue(c *gin.Context) {
	idParam := c.Param("id")
	issueID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue ID"})
		return
	}

	var issue models.Issue
	if err := c.ShouldBindJSON(&issue); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	issue.UpdatedAt = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"title":       issue.Title,
			"description": issue.Description,
			"status":      issue.Status,
			"updatedAt":   issue.UpdatedAt,
		},
	}

	result, err := issueCollection.UpdateOne(ctx, bson.M{"_id": issueID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update issue"})
		return
	}
	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Issue not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Issue updated successfully"})
}

// DeleteIssue deletes an issue by its ID
func DeleteIssue(c *gin.Context) {
	idParam := c.Param("id")
	issueID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := issueCollection.DeleteOne(ctx, bson.M{"_id": issueID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete issue"})
		return
	}
	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Issue not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Issue deleted successfully"})
}
