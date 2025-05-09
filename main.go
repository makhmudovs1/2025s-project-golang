package main

import (
	"blog/internal/database"
	"encoding/json"
	"github.com/google/uuid"
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

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func getPostsHandler(w http.ResponseWriter, r *http.Request) {
	//userID := r.Header.Get("X-USER-ID")
	w.Header().Set("Content-Type", "application/json")

	// Загружаем посты с авторами из БД
	var db []database.Post
	err := database.DB.Preload("Author").Find(&db).Error
	if err != nil {
		http.Error(w, "failed to load posts", http.StatusInternalServerError)
		return
	}
	// Преобразуем их в формат, нужный для JSON (с правильными полями)
	var result []Post
	for _, post := range db {
		result = append(result, Post{
			ID: post.ID,
			Author: Author{
				ID:       post.Author.ID,
				Nickname: post.Author.Nickname,
				Avatar:   post.Author.Avatar,
			},
			Body:      post.Body,
			CreatedAt: parseTime(post.CreatedAt),
			LikeCount: 0,     // пока без лайков
			IsLiked:   false, // пока без Redis
		})
	}

	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		http.Error(w, "failed to encode posts", http.StatusInternalServerError)
		return
	}
}

func createPostHandler(w http.ResponseWriter, r *http.Request) {
	var p Post
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, "failed to decode post body", http.StatusInternalServerError)
	}
	var author database.Author
	err = database.DB.First(&author, "id = ?", p.Author.ID).Error
	if err != nil {
		var result database.Author
		result.ID = p.Author.ID
		result.Nickname = p.Author.Nickname
		result.Avatar = p.Author.Avatar
		err = database.DB.Create(&result).Error
		if err != nil {
			http.Error(w, "failed to create post", http.StatusInternalServerError)
			return
		}
	}
	var result database.Post
	result.ID = uuid.New().String()
	result.Author = author
	result.Body = p.Body
	result.CreatedAt = p.CreatedAt.Format(time.RFC3339)

	err = database.DB.Create(&result).Error
	if err != nil {
		http.Error(w, "failed to create post", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Post{
		ID: result.ID,
		Author: Author{
			ID:       p.Author.ID,
			Nickname: p.Author.Nickname,
			Avatar:   p.Author.Avatar,
		},
		Body:      p.Body,
		CreatedAt: p.CreatedAt,
		LikeCount: 0,
		IsLiked:   false,
	})
}

func main() {
	database.InitPostgres()
	// Все запросы Get /posts будут обрабатываться этой функцией
	http.HandleFunc("/posts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			getPostsHandler(w, r)
			return
		}
		if r.Method == "POST" {
			createPostHandler(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	// Запускаем сервер на порту :8080
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
