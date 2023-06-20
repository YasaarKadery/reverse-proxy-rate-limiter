package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/julienschmidt/httprouter"
	// "github.com/rs/cors"
)

var rdb *redis.Client

func main() {
	// c := cors.New(cors.Options{
	// 	AllowedOrigins:   []string{"http://localhost:3000"},
	// 	AllowCredentials: true,
	// })
	rdb = redis.NewClient(&redis.Options{
		Addr:     "some-redis:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	// Ping Redis to make sure our connection is good
	_, err := rdb.Ping(rdb.Context()).Result()
	if err != nil {
		log.Fatalf("Error connecting to Redis: %v", err)
	}

	router := httprouter.New()

	// Assuming your actual API is defined in TARGET_URL environment variable
	targetUrl := "http://localhost:3000"
	url, _ := url.Parse(targetUrl)
	proxy := httputil.NewSingleHostReverseProxy(url)

	router.Handler("GET", "/api/*path", rateLimit(proxy))

	log.Fatal(http.ListenAndServe(":8080", router))
}

func rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.RemoteAddr

		// Check how many tokens the client has left
		tokensLeft, _ := rdb.Get(r.Context(), key).Int()

		// If the client has no tokens left, reject the request
		if tokensLeft <= 0 {
			http.Error(w, "Rate limit exceeded.", http.StatusTooManyRequests)
			return
		}

		// Decrement the token count for the client
		rdb.Decr(r.Context(), key)

		// If it's the client's first request, initialize the token count and set the expiration time
		if tokensLeft == 0 {
			rdb.Set(r.Context(), key, 100, time.Hour)
		}

		// If the client has tokens left, forward the request to the target API
		next.ServeHTTP(w, r)
	})
}
