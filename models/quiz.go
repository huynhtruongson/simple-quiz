package models

type Quiz struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Questions     []Question `json:"questions"`
	TotalQuestion int        `json:"total_question"`
}
