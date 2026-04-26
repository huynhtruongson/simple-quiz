package main

import (
	"log"

	"github.com/huynhtruongson/simple-quiz/hub"
	"github.com/huynhtruongson/simple-quiz/repo"
	"github.com/huynhtruongson/simple-quiz/server"
)

func main() {
	repo, err := repo.NewQuizRepository()
	if err != nil {
		log.Fatal(err)
	}
	hb := hub.NewHub(repo)
	srv := server.New(repo, hb)

	log.Println("quiz server listening on :8080")
	if err := srv.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
