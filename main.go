package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/julienschmidt/httprouter"
	"golang.org/x/time/rate"
)

// Create a map to hold the rate limiters for each visitor and a mutex.
var visitors = make(map[string]*rate.Limiter)
var mtx = sync.Mutex{}

// Create a new rate limiter and add it to the visitors map, with the IP address as the key
func addVisitor(ip string) *rate.Limiter {
	limiter := rate.NewLimiter(1, 5) // limit to 1 request per second with a burst of 5

	mtx.Lock()
	visitors[ip] = limiter
	mtx.Unlock()

	return limiter
}

// Retrieve and return the rate limiter for the current visitor if it already exists.
// Otherwise call the addVisitor function to add a new entry to the map.
func getVisitor(ip string) *rate.Limiter {
	mtx.Lock()
	limiter, exists := visitors[ip]
	mtx.Unlock()

	if !exists {
		return addVisitor(ip)
	}

	return limiter
}

// Middleware function which will get the rate limiter for the current visitor and
// reject the request if would exceed the rate limit.
func rateLimit(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		limiter := getVisitor(r.RemoteAddr)
		if !limiter.Allow() {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		next(w, r, ps)
	}
}

func reverseProxy(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Create reverse proxy
	url, err := url.Parse("http://localhost:3000/test")
	if err != nil {
		http.Error(w, "Error parsing proxy URL", http.StatusInternalServerError)
		log.Printf("Error parsing proxy URL: %v", err)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(url)

	// Update the headers to allow for SSL redirection
	r.URL.Host = url.Host
	r.URL.Scheme = url.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = url.Host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(w, r)
}

func main() {
	router := httprouter.New()

	// Apply the rate limiter to the API redirect endpoint.
	router.GET("/", rateLimit(reverseProxy))

	log.Println("Server is starting...")
	err := http.ListenAndServe(":8080", router)
	if err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}
