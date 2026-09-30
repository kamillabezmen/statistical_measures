package repository

type StatisticalMeasure struct {
	ID              int
	Title           string
	Description     string
	FullDescription string
	Formula         string
	CalculationTime int
	Status          string
	ImageKey        string
	VideoKey        string
	Likes           []int
}

var statisticalMeasures = []StatisticalMeasure{
	{
		ID:              1,
		Title:           "Среднее арифметическое",
		Description:     "Среднее значение набора чисел.",
		FullDescription: "Рассчитывается как сумма всех значений, делённая на их количество. Используется для характеристики центрального значения набора данных.",
		Formula:         "x̄ = (x₁ + x₂ + ... + xₙ) / n",
		CalculationTime: 1,
		Status:          "published",
		ImageKey:        "mean.jpg",
		VideoKey:        "mean.mp4",
		Likes:           []int{1, 4, 7},
	},
	{
		ID:              2,
		Title:           "Медиана",
		Description:     "Центральное значение упорядоченного набора данных.",
		FullDescription: "Медиана делит упорядоченный набор данных на две равные части. Половина значений находится ниже медианы, а другая половина — выше.",
		Formula:         "Me = x((n+1)/2)",
		CalculationTime: 2,
		Status:          "published",
		ImageKey:        "median.png",
		VideoKey:        "median.mp4",
		Likes:           []int{2, 5},
	},
	{
		ID:              3,
		Title:           "Дисперсия",
		Description:     "Показывает разброс значений относительно среднего.",
		FullDescription: "Дисперсия характеризует степень разброса значений относительно среднего значения. Чем больше дисперсия, тем сильнее значения отличаются от среднего.",
		Formula:         "σ² = Σ(xᵢ - μ)² / n",
		CalculationTime: 3,
		Status:          "published",
		ImageKey:        "variance.png",
		VideoKey:        "variance.mp4",
		Likes:           []int{1, 3, 6, 8},
	},
	{
		ID:              4,
		Title:           "Стандартное отклонение",
		Description:     "Показывает типичное отклонение значений от среднего.",
		FullDescription: "Стандартное отклонение показывает, насколько значения набора данных обычно отклоняются от среднего. Оно является квадратным корнем из дисперсии.",
		Formula:         "σ = √σ²",
		CalculationTime: 4,
		Status:          "published",
		ImageKey:        "standard_deviation.png",
		VideoKey:        "standard_deviation.mp4",
		Likes:           []int{3, 5, 9},
	},
	{
		ID:              5,
		Title:           "Ковариация",
		Description:     "Показывает совместное изменение двух величин.",
		FullDescription: "Ковариация показывает направление совместного изменения двух величин. Положительное значение указывает на изменение в одном направлении, отрицательное — в противоположных.",
		Formula:         "cov(X,Y) = Σ(xᵢ-x̄)(yᵢ-ȳ) / n",
		CalculationTime: 5,
		Status:          "draft",
		ImageKey:        "covariance.png",
		VideoKey:        "covariance.mp4",
		Likes:           []int{},
	},
	{
		ID:              6,
		Title:           "Коэффициент корреляции Пирсона",
		Description:     "Показывает силу и направление линейной связи двух величин.",
		FullDescription: "Коэффициент корреляции Пирсона характеризует силу и направление линейной связи между двумя величинами и принимает значения от −1 до 1.",
		Formula:         "r = cov(X,Y) / (σₓσᵧ)",
		CalculationTime: 6,
		Status:          "deleted",
		ImageKey:        "pearson.png",
		VideoKey:        "pearson_.mp4",
		Likes:           []int{2, 4},
	},
}

type Repository struct {
}
//создаёт новый репозиторий.
func NewRepository() (*Repository, error) {
	return &Repository{}, nil

}
//возвращает все опубликованные расчёты.
func (r *Repository) GetPublished() []StatisticalMeasure {
	var result []StatisticalMeasure

	for _, measure := range statisticalMeasures {
		if measure.Status == "published" {
			result = append(result, measure)
		}
	}

	return result
}
//возвращает черновик расчёта.
func (r *Repository) GetDraft() *StatisticalMeasure {
	for i := range statisticalMeasures {
		if statisticalMeasures[i].Status == "draft" {
			return &statisticalMeasures[i]
		}
	}

	return nil
}
//находит опубликованный расчёт по ID.
func (r *Repository) GetByID(id int) *StatisticalMeasure {
	for i := range statisticalMeasures {
		if statisticalMeasures[i].ID == id &&
			statisticalMeasures[i].Status == "published" {
			return &statisticalMeasures[i]
		}
	}

	return nil
}
//возвращает следующий опубликованный расчёт.
func (r *Repository) GetNext(id int) *StatisticalMeasure {

	foundCurrent := false

	for i := range statisticalMeasures {

		if statisticalMeasures[i].ID == id {
			foundCurrent = true
			continue
		}

		if foundCurrent &&
			statisticalMeasures[i].Status == "published" {

			return &statisticalMeasures[i]
		}
	}

	for i := range statisticalMeasures {

		if statisticalMeasures[i].Status == "published" {
			return &statisticalMeasures[i]
		}
	}

	return nil
}
//фильтрует опубликованные расчёты по времени.
//CalculationTime — условное значение для демонстрации фильтрации.
func (r *Repository) GetPublishedByCalculationTime(maxTime int) []StatisticalMeasure {
	var result []StatisticalMeasure

	for _, measure := range statisticalMeasures {
		if measure.Status == "published" && measure.CalculationTime <= maxTime {
			result = append(result, measure)
		}
	}

	return result
}
