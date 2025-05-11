package main

import (
	"encoding/json"
	"net/http"
	"time"
)

// Описывает Автора поста

type Author struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// Описывает пост в ленте
// Теги нужны, чтобы поля правильно сериализовались в JSON c нужными названиями

type Post struct {
	ID        string    `json:"id"`         // id поста
	Author    Author    `json:"author"`     // У каждого поста есть автор
	Body      string    `json:"body"`       // Содержание поста
	CreatedAt time.Time `json:"created_at"` // Время создания поста
	LikeCount int       `json:"like_count"` // Сколько лайков набрал пост
	IsLiked   bool      `json:"is_liked"`   // Лайкнул ли какой-то пользователь
}

//

func getPostsHandler(w http.ResponseWriter, r *http.Request) {
	//userID := r.Header.Get("X-USER-ID")

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(posts)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func main() {
	// Все запросы Get /posts будут обрабатываться этой функцией
	http.HandleFunc("/posts", getPostsHandler)

	// Запускаем сервер на порту :8080
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
