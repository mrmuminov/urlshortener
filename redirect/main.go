package main

import (
    "context"
    "database/sql"
    "log"
    "net/http"

    "github.com/go-redis/redis/v8"
    _ "github.com/lib/pq"
    pb "urlshortener/proto"  // import the generated gRPC code

    "google.golang.org/grpc"
)

const (
    postgresDSN = "host=postgres user=postgres password=postgres dbname=urlshortener sslmode=disable"
    redisAddr   = "redis:6379"
    grpcAddr    = "shortener:50051"
)

var rdb *redis.Client
var db *sql.DB
var grpcClient pb.URLShortenerClient

func main() {
    var err error

    // Connect to PostgreSQL
    db, err = sql.Open("postgres", postgresDSN)
    if err != nil {
        log.Fatalf("Failed to connect to PostgreSQL: %v", err)
    }
    defer db.Close()

    // Connect to Redis
    rdb = redis.NewClient(&redis.Options{
        Addr: redisAddr,
    })
    _, err = rdb.Ping(context.Background()).Result()
    if err != nil {
        log.Fatalf("Failed to connect to Redis: %v", err)
    }

    // Set up gRPC client
    conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure())
    if err != nil {
        log.Fatalf("Failed to connect to gRPC server: %v", err)
    }
    grpcClient = pb.NewURLShortenerClient(conn)

    http.HandleFunc("/", handleRedirect)
    log.Println("Starting redirect service on :8080...")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleRedirect(w http.ResponseWriter, r *http.Request) {
    shortURL := r.URL.Path[1:]

    // Try to get original URL from Redis
    originalURL, err := rdb.Get(context.Background(), shortURL).Result()
    if err == redis.Nil {
        // If not found in Redis, use gRPC to query original URL
        req := &pb.GetOriginalURLRequest{ShortUrl: shortURL}
        res, err := grpcClient.GetOriginalURL(context.Background(), req)
        if err != nil {
            http.Error(w, "URL not found", http.StatusNotFound)
            return
        }
        originalURL = res.OriginalUrl
        // Cache result in Redis
        _ = rdb.Set(context.Background(), shortURL, originalURL, 0).Err()
    } else if err != nil {
        http.Error(w, "Server error", http.StatusInternalServerError)
        return
    }

    http.Redirect(w, r, originalURL, http.StatusFound)
}
