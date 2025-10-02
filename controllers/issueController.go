package controllers

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"time"

	"civicsync-be/config"
	"civicsync-be/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var issueCollection *mongo.Collection = config.GetCollection("issues")
var voteCollection *mongo.Collection = config.GetCollection("votes")
var userCollection *mongo.Collection = config.GetCollection("users")

// CreateIssue handles the creation of a new issue
func CreateIssue(c *gin.Context) {
	// Extract user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert user ID to ObjectID
	createdByID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var input struct {
		Title       string   `json:"title" binding:"required,max=200"`
		Description string   `json:"description" binding:"required,max=1000"`
		Category    string   `json:"category" binding:"required"`
		Location    string   `json:"location" binding:"required,max=200"`
		ImageURL    *string  `json:"imageUrl,omitempty"`
		Status      *string  `json:"status,omitempty"`
		Latitude    *float64 `json:"latitude,omitempty"`
		Longitude   *float64 `json:"longitude,omitempty"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate category
	validCategories := map[string]bool{
		"Road": true, "Water": true, "Sanitation": true,
		"Electricity": true, "Other": true,
	}
	if !validCategories[input.Category] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category"})
		return
	}

	// Set default status if not provided
	status := models.Pending
	if input.Status != nil {
		switch *input.Status {
		case "Pending", "In Progress", "Resolved":
			status = models.IssueStatus(*input.Status)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
			return
		}
	}

	// Create the issue
	issue := models.Issue{
		ID:          primitive.NewObjectID(),
		Title:       input.Title,
		Description: input.Description,
		Category:    models.IssueCategory(input.Category),
		Location:    input.Location,
		ImageURL:    input.ImageURL,
		Status:      status,
		CreatedBy:   createdByID,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = issueCollection.InsertOne(ctx, issue)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create issue"})
		return
	}

	c.JSON(http.StatusCreated, issue)
}

// GetAllIssues handles retrieving all issues with filtering, pagination, and vote counts
func GetAllIssues(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parse query parameters
	category := c.Query("category")
	status := c.Query("status")
	search := c.Query("search")
	sort := c.DefaultQuery("sort", "newest")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Build query filter
	filter := bson.M{}

	if category != "" && category != "all" {
		filter["category"] = category
	}

	if status != "" && status != "all" {
		filter["status"] = status
	}

	if search != "" {
		filter["$or"] = []bson.M{
			{"title": bson.M{"$regex": search, "$options": "i"}},
			{"description": bson.M{"$regex": search, "$options": "i"}},
		}
	}

	// Calculate pagination
	skip := (page - 1) * limit

	// Sort options
	var sortOptions bson.D
	switch sort {
	case "oldest":
		sortOptions = bson.D{{Key: "createdAt", Value: 1}}
	case "newest":
		fallthrough
	default:
		sortOptions = bson.D{{Key: "createdAt", Value: -1}}
	}

	// Get total count for pagination
	totalCount, err := issueCollection.CountDocuments(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count issues"})
		return
	}

	// Find issues with pagination and sorting
	findOptions := options.Find().
		SetSort(sortOptions).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := issueCollection.Find(ctx, filter, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve issues"})
		return
	}
	defer cursor.Close(ctx)

	var issues []models.Issue
	if err := cursor.All(ctx, &issues); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode issues"})
		return
	}

	// Get current user ID for vote checking (if authenticated)
	var currentUserID *primitive.ObjectID
	if userIDStr, exists := c.Get("user_id"); exists {
		if objID, err := primitive.ObjectIDFromHex(userIDStr.(string)); err == nil {
			currentUserID = &objID
		}
	}

	// Enhance issues with vote counts and user vote status
	type IssueWithVotes struct {
		models.Issue
		Votes        int64                  `json:"votes"`
		UserHasVoted bool                   `json:"userHasVoted"`
		CreatedBy    map[string]interface{} `json:"createdBy"`
	}

	issuesWithVotes := make([]IssueWithVotes, 0, len(issues))

	for _, issue := range issues {
		// Count votes for this issue
		voteCount, err := voteCollection.CountDocuments(ctx, bson.M{"issue": issue.ID})
		if err != nil {
			voteCount = 0
		}

		// Check if current user has voted
		userHasVoted := false
		if currentUserID != nil {
			count, err := voteCollection.CountDocuments(ctx, bson.M{
				"issue": issue.ID,
				"user":  *currentUserID,
			})
			if err == nil && count > 0 {
				userHasVoted = true
			}
		}

		// Get creator info
		var creator models.User
		createdByMap := map[string]interface{}{
			"id": issue.CreatedBy,
		}

		if err := userCollection.FindOne(ctx, bson.M{"_id": issue.CreatedBy}).Decode(&creator); err == nil {
			createdByMap["name"] = creator.Name
			createdByMap["email"] = creator.Email
		}

		issueWithVotes := IssueWithVotes{
			Issue:        issue,
			Votes:        voteCount,
			UserHasVoted: userHasVoted,
			CreatedBy:    createdByMap,
		}

		issuesWithVotes = append(issuesWithVotes, issueWithVotes)
	}

	// Calculate pagination info
	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))

	response := gin.H{
		"issues":      issuesWithVotes,
		"totalIssues": totalCount,
		"totalPages":  totalPages,
		"currentPage": page,
	}

	c.JSON(http.StatusOK, response)
}

// GetIssue retrieves an issue by its ID with vote information
func GetIssue(c *gin.Context) {
	idParam := c.Param("id")
	issueID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var issue models.Issue
	err = issueCollection.FindOne(ctx, bson.M{"_id": issueID}).Decode(&issue)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Issue not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve issue"})
		}
		return
	}

	// Count votes for this issue
	voteCount, err := voteCollection.CountDocuments(ctx, bson.M{"issue": issueID})
	if err != nil {
		voteCount = 0
	}

	// Check if current user has voted (if authenticated)
	userHasVoted := false
	if userIDStr, exists := c.Get("user_id"); exists {
		if currentUserID, err := primitive.ObjectIDFromHex(userIDStr.(string)); err == nil {
			count, err := voteCollection.CountDocuments(ctx, bson.M{
				"issue": issueID,
				"user":  currentUserID,
			})
			if err == nil && count > 0 {
				userHasVoted = true
			}
		}
	}

	// Get creator info
	var creator models.User
	createdByMap := map[string]interface{}{
		"id": issue.CreatedBy,
	}

	if err := userCollection.FindOne(ctx, bson.M{"_id": issue.CreatedBy}).Decode(&creator); err == nil {
		createdByMap["name"] = creator.Name
		createdByMap["email"] = creator.Email
	}

	// Create response with vote information
	response := gin.H{
		"id":           issue.ID,
		"title":        issue.Title,
		"description":  issue.Description,
		"category":     issue.Category,
		"location":     issue.Location,
		"imageUrl":     issue.ImageURL,
		"status":       issue.Status,
		"createdBy":    createdByMap,
		"latitude":     issue.Latitude,
		"longitude":    issue.Longitude,
		"createdAt":    issue.CreatedAt,
		"updatedAt":    issue.UpdatedAt,
		"votes":        voteCount,
		"userHasVoted": userHasVoted,
	}

	c.JSON(http.StatusOK, response)
}

// GetIssuesByUser retrieves all issues created by a specific user
func GetIssuesByUser(c *gin.Context) {
	// Extract user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert user ID to ObjectID
	userObjID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Find issues created by the specified user
	cursor, err := issueCollection.Find(ctx, bson.M{"createdBy": userObjID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve issues"})
		return
	}
	defer cursor.Close(ctx)

	var issues []models.Issue
	if err := cursor.All(ctx, &issues); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode issues"})
		return
	}

	// Get current user ID for vote checking
	currentUserID := &userObjID

	// Enhance issues with vote counts and user vote status
	type IssueWithVotes struct {
		models.Issue
		Votes        int64                  `json:"votes"`
		UserHasVoted bool                   `json:"userHasVoted"`
		CreatedBy    map[string]interface{} `json:"createdBy"`
	}

	issuesWithVotes := make([]IssueWithVotes, 0, len(issues))

	for _, issue := range issues {
		// Count votes for this issue
		voteCount, err := voteCollection.CountDocuments(ctx, bson.M{"issue": issue.ID})
		if err != nil {
			voteCount = 0
		}

		// Check if current user has voted
		userHasVoted := false
		if currentUserID != nil {
			count, err := voteCollection.CountDocuments(ctx, bson.M{
				"issue": issue.ID,
				"user":  *currentUserID,
			})
			if err == nil && count > 0 {
				userHasVoted = true
			}
		}

		// Get creator info
		var creator models.User
		createdByMap := map[string]interface{}{
			"id": issue.CreatedBy,
		}

		if err := userCollection.FindOne(ctx, bson.M{"_id": issue.CreatedBy}).Decode(&creator); err == nil {
			createdByMap["name"] = creator.Name
			createdByMap["email"] = creator.Email
		}

		issueWithVotes := IssueWithVotes{
			Issue:        issue,
			Votes:        voteCount,
			UserHasVoted: userHasVoted,
			CreatedBy:    createdByMap,
		}

		issuesWithVotes = append(issuesWithVotes, issueWithVotes)
	}

	c.JSON(http.StatusOK, issuesWithVotes)
}

// UpdateIssue allows the creator of an issue to update its details
func UpdateIssue(c *gin.Context) {
	idParam := c.Param("id")
	issueID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue ID"})
		return
	}

	// Extract user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert user ID to ObjectID
	userObjID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var input struct {
		Title       *string  `json:"title,omitempty"`
		Description *string  `json:"description,omitempty"`
		Category    *string  `json:"category,omitempty"`
		Location    *string  `json:"location,omitempty"`
		ImageURL    *string  `json:"imageUrl,omitempty"`
		Status      *string  `json:"status,omitempty"`
		Latitude    *float64 `json:"latitude,omitempty"`
		Longitude   *float64 `json:"longitude,omitempty"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if the issue exists and is created by the requesting user
	var issue models.Issue
	err = issueCollection.FindOne(ctx, bson.M{"_id": issueID}).Decode(&issue)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Issue not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve issue"})
		}
		return
	}

	if issue.CreatedBy != userObjID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to update this issue"})
		return
	}

	// Build update document
	update := bson.M{"updatedAt": time.Now()}
	if input.Title != nil {
		update["title"] = *input.Title
	}
	if input.Description != nil {
		update["description"] = *input.Description
	}
	if input.Category != nil {
		validCategories := map[string]bool{
			"Road": true, "Water": true, "Sanitation": true,
			"Electricity": true, "Other": true,
		}
		if !validCategories[*input.Category] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category"})
			return
		}
		update["category"] = *input.Category
	}
	if input.Location != nil {
		update["location"] = *input.Location
	}
	if input.ImageURL != nil {
		update["imageUrl"] = input.ImageURL
	}
	if input.Status != nil {
		switch *input.Status {
		case "Pending", "In Progress", "Resolved":
			update["status"] = *input.Status
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
			return
		}
	}
	if input.Latitude != nil {
		update["latitude"] = *input.Latitude
	}
	if input.Longitude != nil {
		update["longitude"] = *input.Longitude
	}

	// Update the issue in the database
	_, err = issueCollection.UpdateOne(ctx, bson.M{"_id": issueID}, bson.M{"$set": update})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update issue"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Issue updated successfully"})
}

// DeleteIssue allows the creator of an issue to delete it
func DeleteIssue(c *gin.Context) {
	idParam := c.Param("id")
	issueID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue ID"})
		return
	}

	// Extract user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert user ID to ObjectID
	userObjID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if the issue exists and is created by the requesting user
	var issue models.Issue
	err = issueCollection.FindOne(ctx, bson.M{"_id": issueID}).Decode(&issue)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Issue not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve issue"})
		}
		return
	}

	if issue.CreatedBy != userObjID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to delete this issue"})
		return
	}

	// Delete the issue from the database
	_, err = issueCollection.DeleteOne(ctx, bson.M{"_id": issueID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete issue"})
		return
	}

	// Delete associated votes
	_, _ = voteCollection.DeleteMany(ctx, bson.M{"issue": issueID})

	c.JSON(http.StatusOK, gin.H{"message": "Issue deleted successfully"})
}

// HandleVoteOnIssue toggles the user's vote on an issue (vote if not voted, unvote if already voted)
func HandleVoteOnIssue(c *gin.Context) {
	idParam := c.Param("id")
	issueID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue ID"})
		return
	}

	// Extract user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert user ID to ObjectID
	userObjID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if the issue exists
	var issue models.Issue
	err = issueCollection.FindOne(ctx, bson.M{"_id": issueID}).Decode(&issue)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Issue not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve issue"})
		}
		return
	}

	// Check if the user has already voted on this issue
	count, err := voteCollection.CountDocuments(ctx, bson.M{
		"issue": issueID,
		"user":  userObjID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing votes"})
		return
	}

	if count > 0 {
		// User has already voted, remove the vote
		_, err = voteCollection.DeleteOne(ctx, bson.M{
			"issue": issueID,
			"user":  userObjID,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove vote"})
			return
		}

		// Get updated vote count
		updatedVoteCount, err := voteCollection.CountDocuments(ctx, bson.M{"issue": issueID})
		if err != nil {
			updatedVoteCount = 0
		}

		c.JSON(http.StatusOK, gin.H{
			"message":      "Vote removed successfully",
			"voted":        false,
			"votes":        updatedVoteCount,
			"userHasVoted": false,
		})
	} else {
		// User hasn't voted, create a new vote
		vote := models.Vote{
			ID:        primitive.NewObjectID(),
			Issue:     issueID,
			User:      userObjID,
			CreatedAt: time.Now(),
		}

		_, err = voteCollection.InsertOne(ctx, vote)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cast vote"})
			return
		}

		// Get updated vote count
		updatedVoteCount, err := voteCollection.CountDocuments(ctx, bson.M{"issue": issueID})
		if err != nil {
			updatedVoteCount = 1 // At least the vote we just added
		}

		c.JSON(http.StatusOK, gin.H{
			"message":      "Vote cast successfully",
			"voted":        true,
			"votes":        updatedVoteCount,
			"userHasVoted": true,
		})
	}
}

// GetIssueAnalytics returns analytical data about issues
func GetIssueAnalytics(c *gin.Context) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get issues by category using aggregation
	categoryPipeline := []bson.M{
		{
			"$group": bson.M{
				"_id":   "$category",
				"count": bson.M{"$sum": 1},
			},
		},
		{
			"$project": bson.M{
				"name":  "$_id",
				"value": "$count",
				"_id":   0,
			},
		},
	}

	categoryCursor, err := issueCollection.Aggregate(ctx, categoryPipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get category analytics"})
		return
	}
	defer categoryCursor.Close(ctx)

	var issuesByCategory []bson.M
	if err := categoryCursor.All(ctx, &issuesByCategory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode category analytics"})
		return
	}

	// Get last 7 days data
	var last7Days []gin.H
	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

		nextDate := date.AddDate(0, 0, 1)

		count, err := issueCollection.CountDocuments(ctx, bson.M{
			"createdAt": bson.M{
				"$gte": date,
				"$lt":  nextDate,
			},
		})
		if err != nil {
			count = 0
		}

		last7Days = append(last7Days, gin.H{
			"date":  date.Format("2006-01-02"),
			"count": count,
		})
	}

	// Get top voted issues
	// First get recent issues (last 50)
	findOptions := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(50)

	cursor, err := issueCollection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve issues for vote analysis"})
		return
	}
	defer cursor.Close(ctx)

	var issues []models.Issue
	if err := cursor.All(ctx, &issues); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode issues"})
		return
	}

	// Get vote counts for each issue
	type IssueWithVoteCount struct {
		ID       primitive.ObjectID `json:"id"`
		Title    string             `json:"title"`
		Category string             `json:"category"`
		Votes    int64              `json:"votes"`
	}

	var issuesWithVotes []IssueWithVoteCount
	for _, issue := range issues {
		voteCount, err := voteCollection.CountDocuments(ctx, bson.M{"issue": issue.ID})
		if err != nil {
			voteCount = 0
		}

		issuesWithVotes = append(issuesWithVotes, IssueWithVoteCount{
			ID:       issue.ID,
			Title:    issue.Title,
			Category: string(issue.Category),
			Votes:    voteCount,
		})
	}

	// Sort by votes (descending) using sort.Slice
	sort.Slice(issuesWithVotes, func(i, j int) bool {
		return issuesWithVotes[i].Votes > issuesWithVotes[j].Votes
	})

	// Take top 5
	topVotedIssues := issuesWithVotes
	if len(issuesWithVotes) > 5 {
		topVotedIssues = issuesWithVotes[:5]
	}

	// Get total counts
	totalIssues, err := issueCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		totalIssues = 0
	}

	totalVotes, err := voteCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		totalVotes = 0
	}

	openIssues, err := issueCollection.CountDocuments(ctx, bson.M{
		"status": bson.M{"$in": []string{"Pending", "In Progress"}},
	})
	if err != nil {
		openIssues = 0
	}

	// Return analytics response
	response := gin.H{
		"issuesByCategory": issuesByCategory,
		"last7Days":        last7Days,
		"topVotedIssues":   topVotedIssues,
		"totalIssues":      totalIssues,
		"totalVotes":       totalVotes,
		"openIssues":       openIssues,
	}

	c.JSON(http.StatusOK, response)
}

// RecentIssues returns the most recent issues that have latitude and longitude
func RecentIssues(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	limit := 19

	// Filter for issues that have both latitude and longitude
	filter := bson.M{
		"latitude":  bson.M{"$exists": true, "$ne": nil},
		"longitude": bson.M{"$exists": true, "$ne": nil},
	}

	// Project only the required fields
	projection := bson.M{
		"_id":       1,
		"title":     1,
		"latitude":  1,
		"longitude": 1,
		"location":  1,
		"category":  1,
		"createdAt": 1,
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(int64(limit)).
		SetProjection(projection)

	cursor, err := issueCollection.Find(ctx, filter, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve recent issues"})
		return
	}
	// Define a minimal struct for the projected data
	type IssueProjection struct {
		ID        primitive.ObjectID `bson:"_id" json:"id"`
		Title     string             `bson:"title" json:"title"`
		Latitude  *float64           `bson:"latitude" json:"latitude"`
		Longitude *float64           `bson:"longitude" json:"longitude"`
		Location  string             `bson:"location" json:"location"`
		Category  string             `bson:"category" json:"category"`
		CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	}

	var issues []IssueProjection
	if err := cursor.All(ctx, &issues); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode recent issues"})
		return
	}

	// Transform to the required Issue type format
	type IssueResponse struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		Latitude  float64   `json:"latitude"`
		Longitude float64   `json:"longitude"`
		Location  string    `json:"location"`
		Category  string    `json:"category,omitempty"`
		CreatedAt time.Time `json:"createdAt,omitempty"`
	}

	var response []IssueResponse
	for _, issue := range issues {
		if issue.Latitude != nil && issue.Longitude != nil {
			response = append(response, IssueResponse{
				ID:        issue.ID.Hex(),
				Title:     issue.Title,
				Latitude:  *issue.Latitude,
				Longitude: *issue.Longitude,
				Location:  issue.Location,
				Category:  issue.Category,
				CreatedAt: issue.CreatedAt,
			})
		}
	}

	c.JSON(http.StatusOK, response)
}
