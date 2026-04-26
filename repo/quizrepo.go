package repo

import (
	"embed"
	"encoding/json"
	"fmt"

	"github.com/huynhtruongson/simple-quiz/models"
)

//go:embed quizzes.json
var embeddedQuizData embed.FS

type QuizRepository struct {
	quizzes []models.Quiz
}

func NewQuizRepository() (*QuizRepository, error) {
	quizzes, err := loadQuizzes()
	if err != nil {
		return nil, err
	}
	return &QuizRepository{
		quizzes: quizzes,
	}, nil
}

func (r *QuizRepository) ListQuizzes() ([]models.Quiz, error) {
	return r.quizzes, nil
}

func (r *QuizRepository) GetQuizByID(id string) (*models.Quiz, error) {
	for _, quiz := range r.quizzes {
		if quiz.ID == id {
			return &quiz, nil
		}
	}
	return nil, fmt.Errorf("quiz with ID %s not found", id)
}

func loadQuizzes() ([]models.Quiz, error) {
	file, err := embeddedQuizData.ReadFile("quizzes.json")
	if err != nil {
		return nil, err
	}
	var quizzes []models.Quiz
	if err := json.Unmarshal(file, &quizzes); err != nil {
		return nil, fmt.Errorf("failed to decode quiz data from %s: %w", "./quizzes.json", err)
	}
	for i := range quizzes {
		quizzes[i].TotalQuestion = len(quizzes[i].Questions)
	}
	return quizzes, nil
}
