package main

import (
	blogpb "blog/blog/gen"
	"context"
	"fmt"
	"time"
)

type BlogServer struct {
	blogpb.UnimplementedBlogServiceServer
}

var posts = []*blogpb.Post{
	{
		Id: "1",
		Author: &blogpb.Author{
			Id:       "user1",
			Nickname: "Alice",
			Avatar:   "https://example.com/avatar1.png",
		},
		Body:      "Первый пост gRPC!",
		CreatedAt: "2025-04-01T10:00:00Z",
		LikeCount: 5,
		IsLike:    false,
	},
	{
		Id: "2",
		Author: &blogpb.Author{
			Id:       "user2",
			Nickname: "Bob",
			Avatar:   "https://example.com/avatar2.png",
		},
		Body:      "Второй пост gRPC!",
		CreatedAt: "2025-04-01T11:00:00Z",
		LikeCount: 3,
		IsLike:    false,
	},
}

func (s *BlogServer) GetPosts(ctx context.Context, req *blogpb.GetPostsRequest) (*blogpb.GetPostsResponse, error) {
	offset := int(req.Offset)
	limit := int(req.Limit)
	userId := req.UserId

	if offset < 0 || offset >= len(posts) {
		offset = 0
	}
	if limit <= 0 || limit > len(posts) {
		limit = len(posts)
	}
	end := offset + limit
	if end > len(posts) {
		end = len(posts)
	}
	selectedPost := posts[offset:end]
	for _, post := range selectedPost {
		post.IsLike = post.Author.Id == userId
	}
	return &blogpb.GetPostsResponse{
		Posts: selectedPost,
	}, nil
}

func (s *BlogServer) CreatePost(ctx context.Context, req *blogpb.CreatePostRequest) (*blogpb.CreatePostResponse, error) {
	newID := fmt.Sprintf("%d", len(posts)+1)
	post := &blogpb.Post{
		Id: newID,
		Author: &blogpb.Author{
			Id:       req.UserId,
			Nickname: req.UserId,
			Avatar:   "https://example.com/avatar1.png",
		},
		Body:      req.Body,
		CreatedAt: time.Now().Format(time.RFC3339),
		LikeCount: 0,
		IsLike:    false,
	}
	posts = append([]*blogpb.Post{post}, posts...)
	return &blogpb.CreatePostResponse{Post: post}, nil
}

func (s *BlogServer) DeletePost(ctx context.Context, req *blogpb.DeletePostRequest) (*blogpb.DeletePostResponse, error) {
	postId := req.PostId
	userId := req.UserId

	for i, post := range posts {
		if post.Id == postId {
			if post.Author.Id != userId {
				return &blogpb.DeletePostResponse{Success: false}, nil
			}
			posts = append(posts[:i], posts[i+1:]...)
			return &blogpb.DeletePostResponse{Success: true}, nil
		}
	}
	return &blogpb.DeletePostResponse{Success: false}, nil
}

func (s *BlogServer) EditPost(ctx context.Context, req *blogpb.EditPostRequest) (*blogpb.EditPostResponse, error) {
	postId := req.PostId
	userId := req.UserId

	for _, post := range posts {
		if post.Id == postId {
			if post.Author.Id == userId {
				post.Body = req.Body
				return &blogpb.EditPostResponse{Post: post}, nil
			}
			return nil, fmt.Errorf("not the author")
		}
	}
	return nil, fmt.Errorf("post not found")
}

func (s *BlogServer) LikePost(ctx context.Context, req *blogpb.LikePostRequest) (*blogpb.LikePostResponse, error) {
	postId := req.PostId

	for _, post := range posts {
		if post.Id == postId {
			if !post.IsLike {
				post.LikeCount++
				post.IsLike = true
			}
			return &blogpb.LikePostResponse{Post: post}, nil
		}
	}
	return nil, fmt.Errorf("post not found")
}

func (s *BlogServer) UnlikePost(ctx context.Context, req *blogpb.UnlikePostRequest) (*blogpb.UnlikePostResponse, error) {
	postId := req.PostId

	for _, post := range posts {
		if post.Id == postId {
			if post.IsLike && post.LikeCount > 0 {
				post.LikeCount--
				post.IsLike = false
			}
			return &blogpb.UnlikePostResponse{Post: post}, nil
		}
	}
	return nil, fmt.Errorf("post not found")
}
