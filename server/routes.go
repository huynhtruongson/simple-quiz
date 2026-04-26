package server

func (s *Server) registerRoutes() {
	s.router.GET("/quizzes", s.handleListQuizzes)
	s.router.POST("/rooms", s.handleCreateQuizRoom)
	s.router.GET("/rooms/:roomID", s.handleGetQuizRoom)
	s.router.GET("/ws/rooms/:roomID", s.handleWebSocket)
}
