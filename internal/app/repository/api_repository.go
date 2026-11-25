package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"statistical_measures/internal/app/singleton"
)

var (
	ErrDraftExists = errors.New("у пользователя уже есть черновик")
	ErrNotFound    = errors.New("запись не найдена")
	ErrInvalidLike = errors.New("значение лайка должно быть 0 или 1")
)

type APIStatisticalMeasure struct {
	StatisticalMeasure
	IsCreator int `gorm:"column:is_creator"`
	IsLiked   int `gorm:"column:is_liked"`
}

type User struct {
	ID       int    `gorm:"column:id;primaryKey"`
	Name     string `gorm:"column:name"`
	Email    string `gorm:"column:email"`
	Password string `gorm:"column:password"`
}

func (User) TableName() string {
	return "users"
}

type UserStatisticalMeasureLike struct {
	ID                   int `gorm:"column:id;primaryKey"`
	UserID               int `gorm:"column:user_id"`
	StatisticalMeasureID int `gorm:"column:statistical_measure_id"`
}

func (UserStatisticalMeasureLike) TableName() string {
	return "user_statistical_measures_likes"
}

func (r *Repository) baseAPIQuery(userID int) *gorm.DB {
	return r.db.
		Table("statistical_measures").
		Select(`
			statistical_measures.*,
			CASE WHEN statistical_measures.creator_id = ?
				THEN 1 ELSE 0 END AS is_creator,
			CASE WHEN EXISTS (
				SELECT 1
				FROM user_statistical_measures_likes
				WHERE user_statistical_measures_likes.statistical_measure_id =
					statistical_measures.id
				AND user_statistical_measures_likes.user_id = ?
			) THEN 1 ELSE 0 END AS is_liked,
			(
				SELECT COUNT(*)
				FROM user_statistical_measures_likes
				WHERE user_statistical_measures_likes.statistical_measure_id =
					statistical_measures.id
			) AS likes_count
		`, userID, userID)
}

// GET: список опубликованных расчётов с фильтром.
func (r *Repository) GetAPIPublished(
	calculationTime *int,
) ([]APIStatisticalMeasure, error) {
	userID := singleton.GetCurrentUserID()

	var measures []APIStatisticalMeasure

	query := r.baseAPIQuery(userID).
		Where("statistical_measures.status = ?", "published")

	if calculationTime != nil {
		query = query.Where(
			"statistical_measures.calculation_time = ?",
			*calculationTime,
		)
	}

	err := query.
		Order("statistical_measures.id ASC").
		Scan(&measures).Error

	return measures, err
}

// GET: черновик текущего пользователя без ID в запросе.
func (r *Repository) GetAPIDraft() (*APIStatisticalMeasure, error) {
	userID := singleton.GetCurrentUserID()

	var measure APIStatisticalMeasure

	result := r.baseAPIQuery(userID).
		Where(
			"statistical_measures.creator_id = ? AND statistical_measures.status = ?",
			userID, "draft",
		).
		Order("statistical_measures.id ASC").
		Limit(1).
		Scan(&measure)

	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	return &measure, nil
}

// GET: лента. next=true возвращает следующую запись по ID.
func (r *Repository) GetAPIFeed(
	id *int,
	next bool,
) (*APIStatisticalMeasure, error) {
	userID := singleton.GetCurrentUserID()

	var measure APIStatisticalMeasure

	query := r.baseAPIQuery(userID).
		Where("statistical_measures.status = ?", "published")

	if id == nil {
		query = query.Order("statistical_measures.id ASC")
	} else if next {
		query = query.
			Where("statistical_measures.id > ?", *id).
			Order("statistical_measures.id ASC")
	} else {
		query = query.Where("statistical_measures.id = ?", *id)
	}

	result := query.Limit(1).Scan(&measure)

	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	return &measure, nil
}

// POST: создаём черновик с уже загруженными файлами.
func (r *Repository) CreateAPIDraft(
	title, imageURL, videoURL string,
) (*StatisticalMeasure, error) {
	userID := singleton.GetCurrentUserID()

	var count int64

	err := r.db.
		Model(&StatisticalMeasure{}).
		Where(
			"creator_id = ? AND status = ?",
			userID,
			"draft",
		).
		Count(&count).
		Error

	if err != nil {
		return nil, err
	}

	if count > 0 {
		return nil, ErrDraftExists
	}

	measure := StatisticalMeasure{
		Title:           title,
		Description:     nil,
		Formula:         nil,
		CalculationTime: nil,
		Status:          "draft",
		ImageURL:        imageURL,
		VideoURL:        videoURL,
		CreatorID:       userID,
		CreatedAt:       time.Now(),
		FormedAt:        nil,
		LikesCount:      0,
	}

	if err := r.db.Create(&measure).Error; err != nil {
		return nil, err
	}

	return &measure, nil
}

// PUT: публикация только собственного черновика.
func (r *Repository) PublishAPIDraft(
	id int,
	description *string,
	formula *string,
	calculationTime *int,
) error {
	userID := singleton.GetCurrentUserID()
	now := time.Now()

	updates := map[string]interface{}{
		"status":    "published",
		"formed_at": now,
	}

	if description != nil {
		updates["description"] = *description
	}

	if formula != nil {
		updates["formula"] = *formula
	}

	if calculationTime != nil {
		updates["calculation_time"] = *calculationTime
	}

	result := r.db.
		Model(&StatisticalMeasure{}).
		Where(
			"id = ? AND creator_id = ? AND status = ?",
			id,
			userID,
			"draft",
		).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// DELETE: логическое удаление через ORM.
func (r *Repository) DeleteAPIMeasure(id int) error {
	userID := singleton.GetCurrentUserID()

	result := r.db.Model(&StatisticalMeasure{}).
		Where(
			"id = ? AND creator_id = ? AND status = ?",
			id, userID, "published",
		).
		Update("status", "deleted")

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// POST: value=1 ставит лайк, value=0 снимает.
func (r *Repository) SetAPILike(id, value int) error {
	if value != 0 && value != 1 {
		return ErrInvalidLike
	}

	userID := singleton.GetCurrentUserID()

	var count int64
	err := r.db.Model(&StatisticalMeasure{}).
		Where("id = ? AND status = ?", id, "published").
		Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}

	like := UserStatisticalMeasureLike{
		UserID:               userID,
		StatisticalMeasureID: id,
	}

	if value == 1 {
		return r.db.
			Where(
				"user_id = ? AND statistical_measure_id = ?",
				userID, id,
			).
			FirstOrCreate(&like).Error
	}

	return r.db.
		Where(
			"user_id = ? AND statistical_measure_id = ?",
			userID, id,
		).
		Delete(&UserStatisticalMeasureLike{}).Error
}

// POST: регистрация пользователя.
func (r *Repository) RegisterAPIUser(
	name, email, password string,
) (*User, error) {
	user := User{
		Name:     name,
		Email:    email,
		Password: password,
	}

	if err := r.db.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}
