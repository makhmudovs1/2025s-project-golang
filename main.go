package main

import (
	"blog/internal/database"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"log"
	"net/http"
	"strings"
	"time"
)

import blogredis "blog/internal/redis"

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

func applyIsLiked(posts *[]Post, userID string) {
	if userID == "" || len(*posts) == 0 {
		return
	}

	keys := make([]string, 0, len(*posts))
	for _, post := range *posts {
		keys = append(keys, fmt.Sprintf("post:%s:likes", post.ID))
	}

	for i, key := range keys {
		res, err := blogredis.Client.SIsMember(blogredis.Ctx, key, userID).Result()
		if err == nil {
			(*posts)[i].IsLiked = res
		}
	}
}

func getPostsHandler(w http.ResponseWriter, r *http.Request) {
	offset := 0
	limit := 10
	if val := r.URL.Query().Get("offset"); val != "" {
		fmt.Sscanf(val, "%d", &offset)
	}
	if val := r.URL.Query().Get("limit"); val != "" {
		fmt.Sscanf(val, "%d", &limit)
	}
	cacheKey := fmt.Sprintf("posts:page:%d:%d", offset, limit)
	cached, err := blogredis.Client.Get(blogredis.Ctx, cacheKey).Result()
	if err == nil {
		// кеш найден — просто возвращаем как есть
		var cachedPosts []Post
		if err := json.Unmarshal([]byte(cached), &cachedPosts); err == nil {
			// дополняем is_liked и отдаем
			applyIsLiked(&cachedPosts, r.Header.Get("X-USER-ID"))
			json.NewEncoder(w).Encode(cachedPosts)
			return
		}
	}
	userID := r.Header.Get("X-USER-ID")
	w.Header().Set("Content-Type", "application/json")

	// Загружаем посты с авторами из БД
	var db []database.Post
	err = database.DB.Preload("Author").Find(&db).Error
	if err != nil {
		http.Error(w, "failed to load posts", http.StatusInternalServerError)
		return
	}

	result := []Post{}
	// Сохраняем кеш страницы без is_liked
	clone := make([]Post, len(result))
	copy(clone, result)
	for i := range clone {
		clone[i].IsLiked = false // очищаем флаги
	}
	// Преобразуем их в формат, нужный для JSON (с правильными полями)
	bytes, _ := json.Marshal(clone)
	blogredis.Client.Set(blogredis.Ctx, cacheKey, bytes, 10*time.Second)
	for _, post := range db {
		key := fmt.Sprintf("post:%s:likes", post.ID)

		// общее число лайков
		cnt, err := blogredis.Client.SCard(blogredis.Ctx, key).Result()
		if err != nil {
			http.Error(w, "failed to load posts", http.StatusInternalServerError)
			cnt = 0
		}
		// флаг, лайкнул ли текущий USERID
		isLiked := false
		if userID != "" {
			isLiked, _ = blogredis.Client.SIsMember(blogredis.Ctx, key, userID).Result()
		}
		result = append(result, Post{
			ID: post.ID,
			Author: Author{
				ID:       post.Author.ID,
				Nickname: post.Author.Nickname,
				Avatar:   post.Author.Avatar,
			},
			Body:      post.Body,
			CreatedAt: parseTime(post.CreatedAt),
			LikeCount: int(cnt), // уже добавляем лайки
			IsLiked:   isLiked,
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			author = database.Author{
				ID:       p.Author.ID,
				Nickname: p.Author.Nickname,
				Avatar:   p.Author.Avatar,
			}
			if err = database.DB.Create(&author).Error; err != nil {
				http.Error(w, "failed to create author", http.StatusInternalServerError)
				return
			}

		} else {
			http.Error(w, "failed to query author", http.StatusInternalServerError)
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

func likePostHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("LIKE handler: %s %s", r.Method, r.URL.Path)

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "invalid URL", http.StatusBadRequest)
		return
	}
	postID := parts[2]
	userID := r.Header.Get("X-USER-ID")
	if userID == "" {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	key := fmt.Sprintf("post:%s:likes", postID)
	err := blogredis.Client.SAdd(blogredis.Ctx, key, userID).Err()
	if err != nil {
		http.Error(w, "failed to add liked post", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func main() {
	database.InitPostgres()
	blogredis.InitRedis()
	// Все запросы Get /posts будут обрабатываться этой функцией
	http.HandleFunc("/posts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getPostsHandler(w, r)
		case http.MethodPost:
			createPostHandler(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/posts/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/like") {
			likePostHandler(w, r)
			return
		}
		http.NotFound(w, r)
	})

	// Запускаем сервер на порту :8080
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
