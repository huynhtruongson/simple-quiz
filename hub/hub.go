package hub

import (
	"sync"

	"github.com/huynhtruongson/simple-quiz/models"
	"github.com/huynhtruongson/simple-quiz/session"
	"github.com/huynhtruongson/simple-quiz/utils"
)

type QuizRepository interface {
	ListQuizzes() ([]models.Quiz, error)
	GetQuizByID(id string) (*models.Quiz, error)
}

type Hub struct {
	mu       sync.RWMutex
	sessions map[string]*session.QuizSession // rommID to QuizSession
	quizRepo QuizRepository
}

func NewHub(quizRepo QuizRepository) *Hub {
	return &Hub{
		sessions: make(map[string]*session.QuizSession),
		quizRepo: quizRepo,
	}
}

type QuizRoom struct {
	RoomID    string `json:"room_id"`
	QuizID    string `json:"quiz_id"`
	QuizTitle string `json:"quiz_title"`
}

func (h *Hub) CreateQuizRoom(quizID string) (*QuizRoom, error) {
	quiz, err := h.quizRepo.GetQuizByID(quizID)
	if err != nil {
		return nil, err
	}

	roomID := utils.GenerateID()

	h.mu.Lock()
	defer h.mu.Unlock()
	sess := session.NewQuizSession(roomID, quiz, h.Delete)
	h.sessions[roomID] = sess
	go sess.Run()
	return &QuizRoom{
		RoomID:    roomID,
		QuizID:    quizID,
		QuizTitle: quiz.Title,
	}, nil
}

func (h *Hub) GetQuizRoom(roomID string) (*session.QuizSession, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	sess, ok := h.sessions[roomID]
	return sess, ok
}

func (h *Hub) Delete(roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.sessions, roomID)
}
