package dto

// StatisticalMeasureResponse - данные расчёта, которые API разрешает вернуть клиенту.
type StatisticalMeasureResponse struct {
	ID              int     `json:"id"`
	Title           string  `json:"title"`
	Description     *string `json:"description"`
	Formula         *string `json:"formula"`
	CalculationTime *int    `json:"calculation_time"`
	ImageURL        string  `json:"image_url"`
	VideoURL        string  `json:"video_url"`
	IsCreator       int     `json:"is_creator"`
	IsLiked         int     `json:"is_liked"`
	LikesCount      int     `json:"likes_count"`
}

// PublishMeasureRequest - поля, которые клиент имеет право передать при публикации черновика.
type PublishMeasureRequest struct {
	Description     *string `json:"description"`
	Formula         *string `json:"formula"`
	CalculationTime *int    `json:"calculation_time"`
}

// LikeRequest:1 - поставить лайк; 0 - убрать лайк.
type LikeRequest struct {
	Value *int `json:"value" binding:"required"`
}

// RegisterUserRequest - данные регистрации пользователя.
type RegisterUserRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// MessageResponse - простой ответ API.
type MessageResponse struct {
	Message string `json:"message"`
}

// AuthRequest contains the user's credentials.
type AuthRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}
type UserResponse struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}
type DraftResponse struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	ImageURL string `json:"image_url"`
	VideoURL string `json:"video_url"`
}
type ErrorResponse struct {
	Error string `json:"error"`
}
