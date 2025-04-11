package models

import (
	"gorm.io/gorm"
)

type Like struct {
	gorm.Model
	PostID uint `json:"post_id"`
	UserID uint `json:"user_id" gorm:"constraint:OnDelete:CASCADE"`
	Post   Post `json:"-"`
	User   User `json:"-"`
}

func (like *Like) Save() (*Like, error) {
	err := Database.Create(like).Error
	if err != nil {
		return &Like{}, err
	}
	return like, nil
}

func FetchLikesByPostID(postID uint) (*[]Like, error) {
	var likes []Like
	err := Database.Where("post_id = ?", postID).Find(&likes).Error
	if err != nil {
		return &[]Like{}, err
	}
	return &likes, nil
}

// This below is used to check if a user has already liked a post
// It returns the like if it exists, otherwise it returns nil
// This helps give us the like/unlike functionality on the frontend
func FindLikeByUserIDAndPostID(userID uint, postID uint) (*Like, error) {
	var like Like
	err := Database.Where("user_id = ? AND post_id = ?", userID, postID).First(&like).Error
	if err != nil {
		return nil, err
	}
	return &like, nil
}

func (like *Like) Delete() error {
	return Database.Delete(like).Error
}
