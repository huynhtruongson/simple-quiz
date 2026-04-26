package session

import (
	"math"
	"sort"
	"time"

	"github.com/huynhtruongson/simple-quiz/models"
)

const (
	QuestionDuration   = 10 * time.Second
	ShowResultDuration = 5 * time.Second
)

type QuizSessionState string

const (
	QuizSessionStateWaiting        QuizSessionState = "waiting"
	QuizSessionStateQuestionActive QuizSessionState = "question_active"
	QuizSessionStateAnswerReview   QuizSessionState = "answer_review"
	QuizSessionStateFinished       QuizSessionState = "finished"
)

type incomingEvent struct {
	Internal      string
	PlayerID      string
	PlayerMessage *IncomingMessage
}

type QuizSession struct {
	roomID             string
	quiz               *models.Quiz
	state              QuizSessionState
	hostID             string
	players            map[string]*Player // all players in the session
	scores             map[string]int     // playerID to total score
	questionScores     map[string]int     // playerID to score for current question
	answeredPlayer     map[string]bool    // playerID to whether they answer current question
	currentQuestionIdx int
	questionStart      time.Time

	register   chan *Player
	unregister chan *Player

	incomingEvent chan incomingEvent

	onEmptyHandler func(roomID string)
	quit           chan struct{}
}

func NewQuizSession(roomID string, quiz *models.Quiz, onEmptyHandler func(roomID string)) *QuizSession {
	return &QuizSession{
		roomID:             roomID,
		quiz:               quiz,
		state:              QuizSessionStateWaiting,
		players:            make(map[string]*Player),
		scores:             make(map[string]int),
		answeredPlayer:     make(map[string]bool),
		questionScores:     make(map[string]int),
		currentQuestionIdx: -1,
		register:           make(chan *Player),
		unregister:         make(chan *Player),
		incomingEvent:      make(chan incomingEvent, 100),
		quit:               make(chan struct{}),
		onEmptyHandler:     onEmptyHandler,
	}
}

func (s *QuizSession) Run() {
	for {
		select {
		case <-s.quit:
			return
		case p := <-s.register:
			s.handleRegister(p)
		case p := <-s.unregister:
			s.handleUnregister(p)
		case event := <-s.incomingEvent:
			s.handleIncoming(event)
		}
	}
}

func (s *QuizSession) Register(p *Player) {
	s.register <- p
}

func (s *QuizSession) Unregister(p *Player) {
	s.unregister <- p
}

func (s *QuizSession) EnqueuePlayerMessage(playerID string, msg IncomingMessage) {
	s.incomingEvent <- incomingEvent{
		PlayerID:      playerID,
		PlayerMessage: &msg,
	}
}

func (s *QuizSession) State() QuizSessionState {
	return s.state
}

func (s *QuizSession) RoomID() string {
	return s.roomID
}

func (s *QuizSession) Quiz() *models.Quiz {
	return s.quiz
}

func (s *QuizSession) handleRegister(p *Player) {
	if s.state == QuizSessionStateFinished {
		s.sendError(p, "join_closed", "quiz already finished")
		close(p.Send)
		return
	}

	if len(s.players) == 0 {
		s.hostID = p.ID
		p.IsHost = true
	}

	s.players[p.ID] = p
	p.QuizSession = s
	if _, ok := s.scores[p.ID]; !ok {
		s.scores[p.ID] = 0
	}

	s.sendToPlayer(p, OutgoingMessage{
		Type: "session_joined",
		Payload: map[string]interface{}{
			"player_id":   p.ID,
			"name":        p.Name,
			"is_host":     p.IsHost,
			"player_list": s.playerList(),
			"leaderboard": s.leaderboard(),
		},
	})

	s.broadcast(OutgoingMessage{
		Type: "player_joined",
		Payload: map[string]interface{}{
			"player_id":   p.ID,
			"name":        p.Name,
			"player_list": s.playerList(),
			"leaderboard": s.leaderboard(),
		},
	})
}

func (s *QuizSession) handleUnregister(p *Player) {
	existing, ok := s.players[p.ID]
	if !ok || existing != p {
		return
	}
	delete(s.players, p.ID)
	delete(s.answeredPlayer, p.ID)
	close(p.Send)

	if len(s.players) == 0 {
		if s.onEmptyHandler != nil {
			s.onEmptyHandler(s.roomID)
		}
		close(s.quit)
		return
	}

	if p.ID == s.hostID {
		s.hostID = ""
		for _, p := range s.players {
			p.IsHost = true
			s.hostID = p.ID
			break
		}
	}

	s.broadcast(OutgoingMessage{
		Type: "player_left",
		Payload: map[string]interface{}{
			"host_player_id": s.hostID,
			"player_list":    s.playerList(),
			"leaderboard":    s.leaderboard(),
		},
	})
}

func (s *QuizSession) handleIncoming(event incomingEvent) {
	if event.PlayerMessage != nil {
		switch event.PlayerMessage.Type {
		case "start_quiz":
			s.handleStartQuiz(event.PlayerID)
		case "submit_answer":
			s.handleSubmitAnswer(event.PlayerID, event.PlayerMessage)
		default:
			if player := s.players[event.PlayerID]; player != nil {
				s.sendError(player, "invalid_type", "unsupported message type")
			}
		}
		return
	}

	switch event.Internal {
	case "timer_fired":
		if s.state == QuizSessionStateQuestionActive {
			s.showResult()
		}
	case "next_question":
		if s.state == QuizSessionStateAnswerReview {
			s.startNextQuestion()
		}
	case "quiz_end":
		if s.state == QuizSessionStateAnswerReview || s.state == QuizSessionStateQuestionActive {
			s.finishQuiz()
		}
	}
}

func (s *QuizSession) handleStartQuiz(playerID string) {
	player := s.players[playerID]
	if player == nil {
		return
	}
	if s.state != QuizSessionStateWaiting {
		s.sendError(player, "invalid_state", "quiz already started")
		return
	}
	if playerID != s.hostID {
		s.sendError(player, "forbidden", "only host can start quiz")
		return
	}

	s.broadcast(OutgoingMessage{
		Type: "quiz_started",
		Payload: map[string]interface{}{
			"total_questions": len(s.quiz.Questions),
		},
	})

	s.startNextQuestion()
}

func (s *QuizSession) startNextQuestion() {
	s.currentQuestionIdx++
	if s.currentQuestionIdx >= len(s.quiz.Questions) {
		s.finishQuiz()
		return
	}

	s.state = QuizSessionStateQuestionActive
	s.answeredPlayer = make(map[string]bool)
	s.questionScores = make(map[string]int)
	s.questionStart = time.Now()
	question := s.quiz.Questions[s.currentQuestionIdx]
	endsAt := s.questionStart.Add(QuestionDuration).Unix()

	s.broadcast(OutgoingMessage{
		Type: "question_show",
		Payload: map[string]interface{}{
			"question_id": question.ID,
			"text":        question.Text,
			"options":     question.Options,
			"index":       s.currentQuestionIdx,
			"total":       len(s.quiz.Questions),
			"ends_at":     endsAt,
		},
	})

	s.startQuestionTimer()
}

func (s *QuizSession) handleSubmitAnswer(playerID string, msg *IncomingMessage) {
	player := s.players[playerID]
	if player == nil {
		return
	}
	if s.state != QuizSessionStateQuestionActive {
		s.sendError(player, "invalid_state", "question is not active")
		return
	}
	if s.answeredPlayer[playerID] {
		s.sendError(player, "duplicate_answer", "answer already submitted")
		return
	}

	question := s.quiz.Questions[s.currentQuestionIdx]
	if msg.QuestionID != "" && msg.QuestionID != question.ID {
		s.sendError(player, "question_mismatch", "submitted question id does not match current question")
		return
	}

	s.answeredPlayer[playerID] = true

	points := 0
	isCorrect := msg.AnswerIndex == question.Answer
	if isCorrect {
		elapsed := time.Since(s.questionStart).Seconds()
		total := QuestionDuration.Seconds()
		if elapsed < 0 {
			elapsed = 0
		}
		if elapsed > total {
			elapsed = total
		}
		rate := (total - elapsed) / total
		points = int(math.Floor(100 * rate))
		if points < 10 {
			points = 10
		}
	}

	s.questionScores[playerID] = points
	s.scores[playerID] += points

	s.sendToPlayer(player, OutgoingMessage{
		Type: "answer_accepted",
		Payload: map[string]interface{}{
			"question_id":   question.ID,
			"points_earned": points,
			"total_score":   s.scores[playerID],
			"correct":       isCorrect,
		},
	})
}

func (s *QuizSession) startQuestionTimer() {
	go func() {
		<-time.After(QuestionDuration)
		s.incomingEvent <- incomingEvent{Internal: "timer_fired"}
	}()
}

func (s *QuizSession) showResult() {
	s.state = QuizSessionStateAnswerReview
	question := s.quiz.Questions[s.currentQuestionIdx]

	playerResults := make(map[string]int, len(s.players))
	for playerID := range s.players {
		playerResults[playerID] = s.questionScores[playerID]
	}

	s.broadcast(OutgoingMessage{
		Type: "question_result",
		Payload: map[string]interface{}{
			"correct_answer": question.Answer,
			"correct_text":   question.Options[question.Answer],
			"player_results": playerResults,
			"leaderboard":    s.leaderboard(),
		},
	})

	isLast := s.currentQuestionIdx == len(s.quiz.Questions)-1
	go func(isLast bool) {
		<-time.After(ShowResultDuration)
		if isLast {
			s.incomingEvent <- incomingEvent{Internal: "quiz_end"}
		} else {
			s.incomingEvent <- incomingEvent{Internal: "next_question"}
		}
	}(isLast)
}

func (s *QuizSession) finishQuiz() {
	s.state = QuizSessionStateFinished
	s.broadcast(OutgoingMessage{
		Type: "quiz_finished",
		Payload: map[string]interface{}{
			"final_leaderboard": s.leaderboard(),
		},
	})
}

func (s *QuizSession) leaderboard() []models.Leaderboard {
	entries := make([]models.Leaderboard, 0, len(s.players))
	for _, p := range s.players {
		entries = append(entries, models.Leaderboard{
			PlayerID: p.ID,
			Name:     p.Name,
			Score:    s.scores[p.ID],
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Score == entries[j].Score {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Score > entries[j].Score
	})
	for i := range entries {
		entries[i].Rank = i + 1
	}
	return entries
}

func (s *QuizSession) playerList() []models.Player {
	players := make([]models.Player, 0, len(s.players))
	for _, p := range s.players {
		players = append(players, models.Player{
			ID:     p.ID,
			Name:   p.Name,
			Score:  s.scores[p.ID],
			IsHost: p.IsHost,
		})
	}
	return players
}

func (s *QuizSession) sendToPlayer(p *Player, msg OutgoingMessage) {
	p.Send <- msg
}

func (s *QuizSession) broadcast(msg OutgoingMessage) {
	for _, player := range s.players {
		s.sendToPlayer(player, msg)
	}
}
func (s *QuizSession) sendError(player *Player, code, message string) {
	s.sendToPlayer(player, OutgoingMessage{
		Type: "error",
		Payload: map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
