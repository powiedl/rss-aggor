package main

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/powiedl/rss-aggor/internal/database"
)

func startScraping(
	db *database.Queries,
	concurrency int,
	timeBetweenRequest time.Duration,
) {
	log.Printf("Scraping on %v goroutines every %s duration",concurrency,timeBetweenRequest)
	ticker := time.NewTicker(timeBetweenRequest)
	for ; ; <-ticker.C {
		feeds,err := db.GetNextFeedsToFetch(context.Background(),int32(concurrency))
		if err != nil {
			log.Println("error fetching feeds:",err)
			continue
		}

		wg := &sync.WaitGroup{}
		for _, feed := range feeds {
			wg.Add(1)

			go scrapeFeed(db,wg,feed)

		}
		wg.Wait()

	}

}

func scrapeFeed(db *database.Queries, wg *sync.WaitGroup, feed database.Feed) {
	defer wg.Done()
	_, err := db.MarkFeedAsFetched(context.Background(), feed.ID)
	if err != nil {
		log.Println("Error marking feed as fetched:",err)
		return
	}

	rssFeed, err := urlToFeed(feed.Url)
	if err != nil {
		log.Printf("Error fetching feed %v:%v\n",feed.Name,err)
		return
	}

	for _,item := range rssFeed.Channel.Item {
		description := sql.NullString{}
		if item.Description != "" {
			description.String = item.Description
			description.Valid = true
		} 
		ok := false
		pubAt, err := time.Parse(time.RFC1123Z,item.PubDate)
		if err != nil {
			layouts:=[]string{time.RFC1123}
			for _, layout := range layouts {
				if !ok {
					tt, err := time.Parse(layout,item.PubDate)
					if err == nil {
						pubAt = tt
						ok = true
						//log.Printf("Backup timezone %v worked for %v - pubAt='%v'",layout,item.PubDate,pubAt)
					}
				}
			}
		} else {
			ok = true
		}
		if !ok && err != nil {
			log.Printf("Couldn't parse date %v with err %v\n",item.PubDate,err)
			continue
		}
		_, err = db.CreatePost(context.Background(),database.CreatePostParams{
			ID:uuid.New(),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			Title: item.Title,
			Description: description,
			Url:item.Link,
			PublishedAt: pubAt,
			FeedID: feed.ID,
		})

		if err != nil {
			if strings.Contains(err.Error(),"duplicate key") {
				continue
			}
			log.Printf("Couldn't save the post '%v', published at %v, got error:%v\n",item.Title,item.PubDate,err)
		}

	}

	log.Printf("Feed %s collected, %v posts found",feed.Name,len(rssFeed.Channel.Item))
}