package repository

import (
	"database/sql"
	"errors"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type StatisticalMeasure struct {
	ID              int        `gorm:"column:id;primaryKey"`
	Title           string     `gorm:"column:title"`
	Description     string     `gorm:"column:short_description"`
	FullDescription string     `gorm:"column:full_description"`
	Formula         string     `gorm:"column:formula"`
	CalculationTime int        `gorm:"column:calculation_time"`
	Status          string     `gorm:"column:status"`
	ImageURL        string     `gorm:"column:image_url"`
	VideoURL        string     `gorm:"column:video_url"`
	CreatorID       int        `gorm:"column:creator_id"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	FormedAt        *time.Time `gorm:"column:formed_at"`
	LikesCount      int        `gorm:"column:likes_count;->"`
}

// задаём имя таблицы
func (StatisticalMeasure) TableName() string {
	return "statistical_measures"
}

type Repository struct {
	db    *gorm.DB
	sqlDB *sql.DB
}

// подключаемся к PostgreSQL.
func NewRepository() (*Repository, error) {
	dsn := "host=localhost user=postgres password=postgres dbname=statistical_measures port=5433 sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	return &Repository{
		db:    db,
		sqlDB: sqlDB,
	}, nil
}

//возвращаем опубликованные расчёты через ORM
func (r *Repository) GetPublished() []StatisticalMeasure {
	var measures []StatisticalMeasure

	r.db.
		Table("statistical_measures").
		Select(`
			statistical_measures.*,
			COUNT(user_statistical_measures_likes.id) AS likes_count
		`).
		Joins(`
			LEFT JOIN user_statistical_measures_likes
			ON user_statistical_measures_likes.statistical_measure_id = statistical_measures.id
		`).
		Where("statistical_measures.status = ?", "published").
		Group("statistical_measures.id").
		Order("statistical_measures.id").
		Scan(&measures)

	return measures
}

//выполняем поиск через ORM.
func (r *Repository) GetPublishedByCalculationTime(maxTime int) []StatisticalMeasure {
	var measures []StatisticalMeasure

	r.db.
		Table("statistical_measures").
		Select(`
			statistical_measures.*,
			COUNT(user_statistical_measures_likes.id) AS likes_count
		`).
		Joins(`
			LEFT JOIN user_statistical_measures_likes
			ON user_statistical_measures_likes.statistical_measure_id = statistical_measures.id
		`).
		Where(
			"statistical_measures.status = ? AND statistical_measures.calculation_time <= ?",
			"published",
			maxTime,
		).
		Group("statistical_measures.id").
		Order("statistical_measures.id").
		Scan(&measures)

	return measures
}

//возвращаем черновик пользователя через ORM.
func (r *Repository) GetDraft() *StatisticalMeasure {
	var measure StatisticalMeasure

	result := r.db.
		Where("status = ? AND creator_id = ?", "draft", 1).
		First(&measure)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	if result.Error != nil {
		return nil
	}

	r.loadLikesCount(&measure)

	return &measure
}

// возвращаем ровно один опубликованный расчёт через ORM.
// Deleted и draft открыть через ленту нельзя
func (r *Repository) GetByID(id int) *StatisticalMeasure {
	var measure StatisticalMeasure

	result := r.db.
		Where("id = ? AND status = ?", id, "published").
		First(&measure)

	if result.Error != nil {
		return nil
	}

	r.loadLikesCount(&measure)

	return &measure
}

// возвращаем ровно один следующий опубликованный расчёт.
func (r *Repository) GetNext(id int) *StatisticalMeasure {
	var measure StatisticalMeasure

	result := r.db.
		Where("id > ? AND status = ?", id, "published").
		Order("id ASC").
		Limit(1).
		First(&measure)

	// Если дошли до конца, возвращаем первый опубликованный расчёт.
	if result.Error != nil {
		result = r.db.
			Where("status = ?", "published").
			Order("id ASC").
			Limit(1).
			First(&measure)
	}

	if result.Error != nil {
		return nil
	}

	r.loadLikesCount(&measure)

	return &measure
}

// создаем черновик через ORM.
func (r *Repository) CreateDraft(title string) error {
	// У одного пользователя может быть максимум один draft.
	var count int64

	if err := r.db.
		Model(&StatisticalMeasure{}).
		Where("creator_id = ? AND status = ?", 1, "draft").
		Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return errors.New("у пользователя уже есть черновик")
	}

	measure := StatisticalMeasure{
		Title:           title,
		Description:     "",
		FullDescription: "",
		Formula:         "",
		CalculationTime: 0,
		Status:          "draft",
		ImageURL:        "http://invalid.local/new-image.png",
		VideoURL:        "http://invalid.local/new-video.mp4",
		CreatorID:       1,
		CreatedAt:       time.Now(),
		FormedAt:        nil,
	}

	return r.db.Create(&measure).Error
}

//заполняем черновик и публикует его через ORM.
func (r *Repository) PublishDraft(
	id int,
	description string,
	formula string,
	calculationTime int,
) error {
	now := time.Now()

	result := r.db.
		Model(&StatisticalMeasure{}).
		Where("id = ? AND creator_id = ? AND status = ?", id, 1, "draft").
		Updates(map[string]interface{}{
			"short_description": description,
			"full_description":  description,
			"formula":           formula,
			"calculation_time":  calculationTime,
			"status":            "published",
			"formed_at":         now,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("черновик не найден")
	}

	return nil
}

// логически удаляем расчёт
func (r *Repository) DeleteMeasure(id int) error {
	result, err := r.sqlDB.Exec(
		`UPDATE statistical_measures
		 SET status = 'deleted'
		 WHERE id = $1 AND status = 'published'`,
		id,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("опубликованный расчёт не найден")
	}

	return nil
}

//получаем количество лайков из M:M таблицы.
func (r *Repository) loadLikesCount(measure *StatisticalMeasure) {
	var count int64

	r.db.
		Table("user_statistical_measures_likes").
		Where("statistical_measure_id = ?", measure.ID).
		Count(&count)

	measure.LikesCount = int(count)
}