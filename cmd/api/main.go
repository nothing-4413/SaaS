package main

import (
	"log"
	"net/http"

	"github.com/nothing-4413/saas/internal/auth"
)

func main() {
	store := auth.NewMemoryStore()
	service := auth.NewService(store)
	handler := auth.NewHandler(service)

	log.Println("api listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
