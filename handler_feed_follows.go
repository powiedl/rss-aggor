package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/powiedl/rss-aggor/internal/database"
)

func (apiCfg *apiConfig) handlerCreateFeedFollow(w http.ResponseWriter, r *http.Request, user database.User) {
	type parameters struct {
		FeedID uuid.UUID `json:"feed_id"`
	}
	params := parameters{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w,400,fmt.Sprintf("Error parsing JSON: %v",err))
		return
	}
	feedFollow, err := apiCfg.DB.CreateFeedFollow(r.Context(),database.CreateFeedFollowParams{
		ID: uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID: user.ID,
		FeedID: params.FeedID,
	})

	if err != nil {
		respondWithError(w,400,fmt.Sprintf("Couldn't create feed follow:%v",err))
		return
	}

	respondWithJSON(w, 201, databaseFeedFollowToFeedFollow(feedFollow))
}

func (apiCfg *apiConfig) handlerGetFeedFollows(w http.ResponseWriter, r *http.Request, user database.User) {

	feedFollows, err := apiCfg.DB.GetFeedFollows(r.Context(),user.ID)

	if err != nil {
		respondWithError(w,400,fmt.Sprintf("Couldn't get the feeds the user follows:%v",err))
		return
	}

	respondWithJSON(w, 201,  databaseFeedFollowsToFeedFollows(feedFollows))
}

func (apiCfg *apiConfig) handlerDeleteFeedFollows(w http.ResponseWriter, r *http.Request, user database.User) {
	feedFollowIDStr := chi.URLParam(r,"feedFollowID")
	feedFollowID,err := uuid.Parse(feedFollowIDStr)
	if err != nil {
		respondWithError(w,400,fmt.Sprintf("Couldn't parse feed follow id:%v",err))
		return
	}

	err = apiCfg.DB.DeleteFeedFollow(r.Context(),database.DeleteFeedFollowParams{
		UserID:user.ID,
		ID:feedFollowID,
	})
	if err != nil {
		respondWithError(w,400,fmt.Sprintf("Couldn't delete the feed follow:%v",err))
		return
	}

	respondWithJSON(w, 200,  struct{}{})
}
