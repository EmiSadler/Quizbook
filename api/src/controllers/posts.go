package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/makersacademy/go-react-acebook-template/api/src/auth"
	"github.com/makersacademy/go-react-acebook-template/api/src/models"
)

type PostCommentJSON struct {
	UserID   uint   `json:"userID"`
	Username string `json:"username"`
	Contents string `json:"contents"`
}

type JSONPost struct {
	ID         uint              `json:"_id"`
	Question   string            `json:"question"`
	Answer     string            `json:"answer"`
	UserID     uint              `json:"user_id"`
	Username   string            `json:"username"`
	User       JSONPostUser      `json:"user"`
	Comments   []PostCommentJSON `json:"comments"`
	NumOfLikes int               `json:"numOfLikes"`
	Liked      bool              `json:"liked"`
	CreatedAt  string            `json:"created_at"`
}

type JSONPostUser struct {
	ID                uint   `json:"_id"`
	Username          string `json:"username"`
	ProfilePictureURL string `json:"profilePicture"`
}

func GetAllPosts(ctx *gin.Context) {
	// ============================= Fetch all posts from the database ==========================
	posts, err := models.FetchAllPosts()
	if err != nil {
		SendInternalError(ctx, err)
		return
	}

	// ========== Get the user ID from the context (set by AuthenticationMiddleware) ============
	val, _ := ctx.Get("userID")
	userID := val.(string)
	userIDUint, err := strconv.ParseUint(userID, 10, 32)
	if err != nil {
		SendInternalError(ctx, err)
		return
	}
	token, _ := auth.GenerateToken(userID) // Generate new token for the response

	// ============================= Convert posts to JSON Structs ==============================
	jsonPosts := make([]JSONPost, 0)
	for _, post := range *posts {
		// Grab the post author's username
		author, err := models.FindUser(strconv.Itoa(int(post.UserID)))
		authorUsername := "Unknown" // Default if author not found, although this shouldn't happen
		if err == nil {
			authorUsername = author.Username
		}

		// grab comments for the post
		comments, err := models.FetchCommentsByPostID(post.ID)
		if err != nil {
			// I think this is probably the best way to hangle this? maybe continue to next post?
			// For now, we'll just send an internal error for simplicity.
			SendInternalError(ctx, err)
			return
		}

		// ============================= Convert comments to JSON Structs ==========================
		jsonComments := make([]PostCommentJSON, 0)
		for _, comment := range *comments {
			// Find the user who made the comment to get their username
			username := "Unknown" // Default username if user not found
			user, err := models.FindUser(strconv.Itoa(int(comment.UserID)))
			if err == nil {
				username = user.Username
			}

			// Append the comment to the JSON comments
			jsonComments = append(jsonComments, PostCommentJSON{
				UserID:   comment.UserID,
				Username: username,
				Contents: comment.Content,
			})
		}

		// ============================= Fetch likes for the post ===================================
		likes, err := models.FetchLikesByPostID(post.ID)
		if err != nil {
			SendInternalError(ctx, err)
			return
		}
		numOfLikes := len(*likes)

		// ============================= Check if current user has liked this post ==================
		liked := false
		existingLike, err := models.FindLikeByUserIDAndPostID(uint(userIDUint), post.ID)
		if err == nil && existingLike != nil {
			liked = true
		}

		// ============================= Append to JSON posts for response ==========================
		jsonPosts = append(jsonPosts, JSONPost{
			ID:       post.ID,
			Question: post.Question,
			Answer:   post.Answer,
			UserID:   post.UserID,
			Username: authorUsername,
			User: JSONPostUser{
				ID:                author.ID,
				Username:          author.Username,
				ProfilePictureURL: author.ProfilePictureURL,
			},
			Comments:   jsonComments,
			NumOfLikes: numOfLikes,
			Liked:      liked,
			CreatedAt:  post.CreatedAt.Format(time.RFC3339),
		})
	}

	// ============================ Send response (including token) ================================
	ctx.JSON(http.StatusOK, gin.H{"posts": jsonPosts, "token": token})
}

type createPostRequestBody struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

func CreatePost(ctx *gin.Context) {
	// ============================= Get the request body =========================================
	var requestBody createPostRequestBody
	err := ctx.BindJSON(&requestBody)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err})
		return
	}

	if len(requestBody.Question) == 0 || len(requestBody.Answer) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Both question and answer are required"})
		return
	}

	// ========== Get the user ID from the context (set by AuthenticationMiddleware) ============
	val, _ := ctx.Get("userID")
	userID, ok := val.(string)
	if !ok {
		SendInternalError(ctx, errors.New("userID is not a string"))
		return
	}

	// ================== Convert userID string to uint for the database ========================
	parsed, err := strconv.ParseUint(userID, 10, 32)
	if err != nil {
		SendInternalError(ctx, err)
		return
	}

	// ============================= Create the new post =========================================
	// Create the new post
	newPost := models.Post{
		Question: requestBody.Question,
		Answer:   requestBody.Answer,
		UserID:   uint(parsed),
	}

	// Save the new post to the database
	_, err = newPost.Save()
	if err != nil {
		SendInternalError(ctx, err)
		return
	}

	// ========================== Generate token & send response ================================
	token, _ := auth.GenerateToken(userID)
	ctx.JSON(http.StatusCreated, gin.H{"message": "Post created", "token": token})
}

func GetPostsByUserID(ctx *gin.Context) {
	// ======================= Get the user ID from the URL params ==============================
	userIDParam := ctx.Param("id")
	userID, err := strconv.ParseUint(userIDParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid user ID"})
		return
	}

	// ============================= Fetch posts by the user ID =================================
	posts, err := models.FetchPostsByUserID(uint(userID))
	if err != nil {
		SendInternalError(ctx, err)
		return
	}

	// ========== Get the user ID from the context (set by AuthenticationMiddleware) ============
	val, _ := ctx.Get("userID")
	tokenUserID := val.(string)
	currentUserIDUint, err := strconv.ParseUint(tokenUserID, 10, 32)
	if err != nil {
		SendInternalError(ctx, err)
		return
	}
	token, _ := auth.GenerateToken(tokenUserID) // Generate new token for the response

	// ============================= Convert posts to JSON Structs ==============================
	jsonPosts := make([]JSONPost, 0)
	for _, post := range *posts {
		// Grab the post author's username
		authorUsername := "Unknown" // Default if author not found
		author, err := models.FindUser(strconv.Itoa(int(post.UserID)))
		if err == nil {
			authorUsername = author.Username
		}

		// Grab comments for the post
		comments, err := models.FetchCommentsByPostID(post.ID)
		if err != nil {
			SendInternalError(ctx, err)
			return
		}
		// Convert comments to JSON structs
		jsonComments := make([]PostCommentJSON, 0)
		for _, comment := range *comments {
			username := "Unknown" // Default if user not found
			user, err := models.FindUser(strconv.Itoa(int(comment.UserID)))
			if err == nil {
				username = user.Username
			}

			jsonComments = append(jsonComments, PostCommentJSON{
				UserID:   comment.UserID,
				Username: username,
				Contents: comment.Content,
			})
		}

		// Grab likes for the post
		likes, err := models.FetchLikesByPostID(post.ID)
		if err != nil {
			SendInternalError(ctx, err)
			return
		}
		numOfLikes := len(*likes)

		// ============================= Check if current user has liked this post ==================
		liked := false
		existingLike, err := models.FindLikeByUserIDAndPostID(uint(currentUserIDUint), post.ID)
		if err == nil && existingLike != nil {
			liked = true
		}

		// ========================= Append to JSON posts for response ==============================
		jsonPosts = append(jsonPosts, JSONPost{
			ID:       post.ID,
			Question: post.Question,
			Answer:   post.Answer,
			UserID:   post.UserID,
			Username: authorUsername,
			User: JSONPostUser{
				ID:                author.ID,
				Username:          author.Username,
				ProfilePictureURL: author.ProfilePictureURL,
			},
			Comments:   jsonComments,
			NumOfLikes: numOfLikes,
			Liked:      liked,
			CreatedAt:  post.CreatedAt.Format(time.RFC3339),
		})
	}

	// ========================== Generate token & send response ================================
	ctx.JSON(http.StatusOK, gin.H{"posts": jsonPosts, "token": token})
}

func GetLikedPostsByUserID(ctx *gin.Context) {
	userId := ctx.Param("id")
	userIdUint, err := strconv.ParseUint(userId, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid user ID"})
		return
	}

	posts, err := models.FetchLikedPostsByUserID(uint(userIdUint))
	if err != nil {
		SendInternalError(ctx, err)
		return
	}

	val, _ := ctx.Get("userID")
	userID := val.(string)
	token, _ := auth.GenerateToken(userID)

	jsonPosts := make([]JSONPost, len(*posts))
	for _, post := range *posts {

		// ========================= Append to JSON posts for response ==============================
		jsonPosts = append(jsonPosts, JSONPost{
			ID:       post.ID,
			Question: post.Question,
			Answer:   post.Answer,
			UserID:   post.UserID,
			Username: post.User.Username,
			User: JSONPostUser{
				ID:                post.User.ID,
				Username:          post.User.Username,
				ProfilePictureURL: post.User.ProfilePictureURL,
			},
			CreatedAt: post.CreatedAt.Format(time.RFC3339),
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"posts": jsonPosts, "token": token})
}

func GetCurrentUserPosts(ctx *gin.Context) {
	// ========== Get the user ID from the context (set by AuthenticationMiddleware) ============
	val, _ := ctx.Get("userID")
	userID := val.(string)
	parsed, err := strconv.ParseUint(userID, 10, 32)
	if err != nil {
		SendInternalError(ctx, err)
		return
	}
	token, _ := auth.GenerateToken(userID) // Generate new token for the response

	// ============================= Fetch posts by the user ID =================================
	posts, err := models.FetchPostsByUserID(uint(parsed))
	if err != nil {
		SendInternalError(ctx, err)
		return
	}

	// ============================= Convert posts to JSON Structs ==============================
	jsonPosts := make([]JSONPost, 0)
	for _, post := range *posts {
		// Grab the post author's username
		authorUsername := "Unknown" // Default if author not found
		author, err := models.FindUser(strconv.Itoa(int(post.UserID)))
		if err == nil {
			authorUsername = author.Username
		}

		// Grab comments for the post
		comments, err := models.FetchCommentsByPostID(post.ID)
		if err != nil {
			SendInternalError(ctx, err)
			return
		}

		// ============================= Convert comments to JSON Structs ==========================
		jsonComments := make([]PostCommentJSON, 0)
		for _, comment := range *comments {
			// Find the user who made the comment to get their username
			username := "Unknown" // Default if user not found
			user, err := models.FindUser(strconv.Itoa(int(comment.UserID)))
			if err == nil {
				username = user.Username
			}

			// Append the comment to the JSON comments
			jsonComments = append(jsonComments, PostCommentJSON{
				UserID:   comment.UserID,
				Username: username,
				Contents: comment.Content,
			})
		}

		// ============================= Fetch likes for the post ===================================
		likes, err := models.FetchLikesByPostID(post.ID)
		if err != nil {
			SendInternalError(ctx, err)
			return
		}
		numOfLikes := len(*likes)

		// ============================= Check if current user has liked this post ================
		liked := false
		existingLike, err := models.FindLikeByUserIDAndPostID(uint(parsed), post.ID)
		if err == nil && existingLike != nil {
			liked = true
		}

		// ============================= Append to JSON posts for response ==========================
		jsonPosts = append(jsonPosts, JSONPost{
			ID:       post.ID,
			Question: post.Question,
			Answer:   post.Answer,
			UserID:   post.UserID,
			Username: authorUsername,
			User: JSONPostUser{
				ID:                author.ID,
				Username:          author.Username,
				ProfilePictureURL: author.ProfilePictureURL,
			},
			Comments:   jsonComments,
			NumOfLikes: numOfLikes,
			Liked:      liked,
			CreatedAt:  post.CreatedAt.Format(time.RFC3339),
		})
	}

	// ============================ Send response (including token) ================================
	ctx.JSON(http.StatusOK, gin.H{"posts": jsonPosts, "token": token})
}

func GetPostByID(ctx *gin.Context) {
	// ======================= Get the post ID from the URL params ==============================
	postIDParam := ctx.Param("id")
	postID, err := strconv.ParseUint(postIDParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid post ID"})
		return
	}

	// ============================= Fetch the post by ID =======================================
	post, err := models.FetchPostByID(uint(postID))
	if err != nil {
		if err.Error() == "record not found" { // if there's no post with that ID
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Post not found"})
			return
		}
		SendInternalError(ctx, err)
		return
	}

	// ========== Get the user ID from the context (set by AuthenticationMiddleware) ============
	val, _ := ctx.Get("userID")
	userID := val.(string)
	userIDUint, err := strconv.ParseUint(userID, 10, 32)
	if err != nil {
		SendInternalError(ctx, err)
		return
	}
	token, _ := auth.GenerateToken(userID) // Generate new token for the response

	// ============================= Convert post to JSON Struct ===============================
	// Grab the post author's username
	authorUsername := "Unknown" // Default if author not found
	author, err := models.FindUser(strconv.Itoa(int(post.UserID)))
	if err == nil {
		authorUsername = author.Username
	}

	// Grab comments for the post
	comments, err := models.FetchCommentsByPostID(post.ID)
	if err != nil {
		SendInternalError(ctx, err)
		return
	}

	// ============================= Convert comments to JSON Structs ==========================
	jsonComments := make([]PostCommentJSON, 0)
	for _, comment := range *comments {
		// Find the user who made the comment to get their username
		username := "Unknown" // Default if user not found
		user, err := models.FindUser(strconv.Itoa(int(comment.UserID)))
		if err == nil {
			username = user.Username
		}

		// Append the comment to the JSON comments
		jsonComments = append(jsonComments, PostCommentJSON{
			UserID:   comment.UserID,
			Username: username,
			Contents: comment.Content,
		})
	}

	// Grab likes for the post
	likes, err := models.FetchLikesByPostID(post.ID)
	if err != nil {
		SendInternalError(ctx, err)
		return
	}
	numOfLikes := len(*likes)

	// ============================= Check if current user has liked this post ================
	liked := false
	existingLike, err := models.FindLikeByUserIDAndPostID(uint(userIDUint), post.ID)
	if err == nil && existingLike != nil {
		liked = true
	}

	jsonPost := JSONPost{
		ID:       post.ID,
		Question: post.Question,
		Answer:   post.Answer,
		UserID:   post.UserID,
		Username: authorUsername,
		User: JSONPostUser{
			ID:                author.ID,
			Username:          author.Username,
			ProfilePictureURL: author.ProfilePictureURL,
		},
		Comments:   jsonComments,
		NumOfLikes: numOfLikes,
		Liked:      liked,
		CreatedAt:  post.CreatedAt.Format(time.RFC3339),
	}
	// ========================= Send response (including token) ==============================
	ctx.JSON(http.StatusOK, gin.H{"post": jsonPost, "token": token})
}

func UpdatePost(ctx *gin.Context) {
	// ======================= Get the post ID from the URL params ==============================
	postIDParam := ctx.Param("id")
	postID, err := strconv.ParseUint(postIDParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid post ID"})
		return
	}

	// ========== Get the user ID from the context (set by AuthenticationMiddleware) ============
	val, _ := ctx.Get("userID")
	userID, ok := val.(string)
	if !ok {
		SendInternalError(ctx, errors.New("userID is not a string"))
		return
	}

	// ================== Convert userID string to uint for the database ========================
	userIDUint, err := strconv.ParseUint(userID, 10, 32)
	if err != nil {
		SendInternalError(ctx, err)
		return
	}

	// ============================= Get the post from the database =======================================
	post, err := models.FetchPostByID(uint(postID))
	if err != nil {
		if err.Error() == "record not found" {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Post not found"})
			return
		}
		SendInternalError(ctx, err)
		return
	}

	// ============================= Check if user is the owner of the post ===========================
	if post.UserID != uint(userIDUint) {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "You can only update your own posts"})
		return
	}

	// =================== Get the request body (of the things to update) =========================
	var updates map[string]interface{}
	if err := ctx.BindJSON(&updates); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	// ============================= Validate question and answer are not blank ==============================
	if question, exists := updates["question"]; exists {
		if questionStr, ok := question.(string); ok && len(strings.TrimSpace(questionStr)) == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "Question cannot be blank"})
			return
		}
	}

	if answer, exists := updates["answer"]; exists {
		if answerStr, ok := answer.(string); ok && len(strings.TrimSpace(answerStr)) == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "Answer cannot be blank"})
			return
		}
	}

	// ============================= Update the post in the database ==============================
	_, err = models.UpdatePost(uint(postID), updates)
	if err != nil {
		SendInternalError(ctx, err)
		return
	}

	// ===================== Send a success message to the frontend (with token) ==================
	token, _ := auth.GenerateToken(userID)
	ctx.JSON(http.StatusOK, gin.H{"message": "Post updated successfully", "token": token})
}

func DeletePostByID(ctx *gin.Context) {

	// ======================= Get the post ID from the URL ==============================
	postIDParam := ctx.Param("id")
	postID, err := strconv.ParseUint(postIDParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Invalid post ID"})
		return
	}
	// ==================== Get the user ID ====================
	userIDstr, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	// Convert userID to string
	userIDstring, ok := userIDstr.(string)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid user ID format"})
		return
	}

	// Convert userID to uint
	userID, err := strconv.ParseUint(userIDstring, 10, 32)
	if err != nil {
		SendInternalError(ctx, err)
		return
	}

	// ======================= Fetch the post by ID ============================================
	post, err := models.FetchPostByID(uint(postID))
	if err != nil {
		if err.Error() == "record not found" {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Post not found"})
			return
		}
		SendInternalError(ctx, err)
		return
	}

	// ==================== Check if the user is the owner of the post =========================
	if post.UserID != uint(userID) {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "You are not authorised to delete this post"})
		return
	}

	// ======================= Delete the post ================================================
	err = models.DeletePost(uint(postID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete post"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Post deleted successfully"})
}
