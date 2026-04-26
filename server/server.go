package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/huynhtruongson/simple-quiz/hub"
	"github.com/huynhtruongson/simple-quiz/session"
	"github.com/huynhtruongson/simple-quiz/utils"
)

type Server struct {
	router   *gin.Engine
	hub      *hub.Hub
	quizRepo hub.QuizRepository
	upgrader websocket.Upgrader
}

func New(quizRepo hub.QuizRepository, hb *hub.Hub) *Server {
	s := &Server{
		router:   gin.Default(),
		quizRepo: quizRepo,
		hub:      hb,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
	s.registerRoutes()
	return s
}

func (s *Server) Engine() *gin.Engine {
	return s.router
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

func (s *Server) handleListQuizzes(c *gin.Context) {
	quizzes, err := s.quizRepo.ListQuizzes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"quizzes": quizzes,
	})
}

type createRoomRequest struct {
	QuizID string `json:"quiz_id"`
}

func (s *Server) handleCreateQuizRoom(c *gin.Context) {
	var req createRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.QuizID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quiz_id is required"})
		return
	}
	room, err := s.hub.CreateQuizRoom(req.QuizID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "quiz not found"})
		return
	}
	c.JSON(http.StatusCreated, room)
}

func (s *Server) handleGetQuizRoom(c *gin.Context) {
	roomID := c.Param("roomID")
	room, ok := s.hub.GetQuizRoom(roomID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"room_id": room.RoomID(),
		"quiz_id": room.Quiz().ID,
		"title":   room.Quiz().Title,
		"state":   room.State(),
	})
}

func (s *Server) handleWebSocket(c *gin.Context) {
	roomID := c.Param("roomID")
	sess, ok := s.hub.GetQuizRoom(roomID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}

	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	var joinMsg session.IncomingMessage
	if err := conn.ReadJSON(&joinMsg); err != nil {
		_ = conn.WriteJSON(session.OutgoingMessage{
			Type: "error",
			Payload: gin.H{
				"code":    "join_required",
				"message": "first message must be join",
			},
		})
		_ = conn.Close()
		return
	}

	if joinMsg.Type != "join" {
		_ = conn.WriteJSON(session.OutgoingMessage{
			Type: "error",
			Payload: gin.H{
				"code":    "join_required",
				"message": "first message must be join",
			},
		})
		_ = conn.Close()
		return
	}
	// if joinMsg.QuizID != "" && joinMsg.QuizID != room.QuizID {
	// 	_ = conn.WriteJSON(session.OutgoingMessage{
	// 		Type: "error",
	// 		Payload: gin.H{
	// 			"code":    "quiz_mismatch",
	// 			"message": "quiz_id does not match room quiz",
	// 		},
	// 	})
	// 	_ = conn.Close()
	// 	return
	// }
	name := strings.TrimSpace(joinMsg.Name)
	if name == "" {
		name = "Player-" + utils.GenerateID()
	}

	// sess, ok := s.hub.GetQuizRoom(roomID)
	// if !ok {
	// 	_ = conn.WriteJSON(session.OutgoingMessage{
	// 		Type: "error",
	// 		Payload: gin.H{
	// 			"code":    "room_not_found",
	// 			"message": "room not found",
	// 		},
	// 	})
	// 	_ = conn.Close()
	// 	return
	// }

	player := &session.Player{
		ID:   utils.GenerateID(),
		Name: name,
		Conn: conn,
		Send: make(chan session.OutgoingMessage, 64),
	}

	sess.Register(player)
	go player.WriteMessage()
	go player.ReadMessage()
}
