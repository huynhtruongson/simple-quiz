package session

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	maxMessageSize = 512
)

type IncomingMessage struct {
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	QuestionID  string `json:"question_id,omitempty"`
	AnswerIndex int    `json:"answer_index,omitempty"`
}

type OutgoingMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

type Player struct {
	Conn        *websocket.Conn
	QuizSession *QuizSession
	ID          string
	Name        string
	IsHost      bool
	Send        chan OutgoingMessage
}

func NewPlayer(conn *websocket.Conn, quizSession *QuizSession, id, name string) *Player {
	return &Player{
		Conn:        conn,
		QuizSession: quizSession,
		ID:          id,
		Name:        name,
		Send:        make(chan OutgoingMessage),
	}
}

func (p *Player) Close() {
	if p.Conn != nil {
		_ = p.Conn.Close()
	}
}

func (p *Player) ReadMessage() {
	defer func() {
		if p.QuizSession != nil {
			p.QuizSession.Unregister(p)
		}
		p.Close()
	}()

	p.Conn.SetReadLimit(maxMessageSize)

	for {
		var msg IncomingMessage
		if err := p.Conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("read message error player=%s: %v", p.ID, err)
			}
			break
		}
		if p.QuizSession != nil {
			p.QuizSession.EnqueuePlayerMessage(p.ID, msg)
		}
	}
}

func (p *Player) WriteMessage() {
	defer func() {
		p.Close()
	}()

	for {
		select {
		case msg, ok := <-p.Send:
			_ = p.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = p.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := p.Conn.WriteJSON(msg); err != nil {
				return
			}
		}
	}
}
