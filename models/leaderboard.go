package models

type Leaderboard struct {
	PlayerID string `json:"player_id"`
	Name     string `json:"name"`
	Rank     int    `json:"rank"`
	Score    int    `json:"score"`
}
