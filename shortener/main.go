package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "net"

    "github.com/go-redis/redis/v8"
    _ "github.com/lib/pq"
    pb "urlshortener/proto"  // import the generated gRPC code

    "google.golang.org/grpc"
)

const (
    postgresDSN = "host=postgres user=postgres password=postgres dbname=urlshortener sslmode=disable"
    redisAddr   = "redis:6379"
)

var rdb *redis.Client
var db *sql.DB

// URLShortenerServer implements pb.URLShortenerServer
type URLShortenerServer struct {
    pb.UnimplementedURLShortenerServer
}

// ShortenURL generates a short URL, stores it in the database and cache.
func (s *URLShortenerServer) ShortenURL(ctx context.Context, req *pb.ShortenURLRequest) (*pb.ShortenURLResponse, error) {
    originalURL := req.OriginalUrl
    shortURL := generateShortURL()

    // Store in PostgreSQL
    _, err := db.Exec("INSERT INTO urls (short_url, original_url) VALUES ($1, $2)", shortURL, originalURL)
    if err != nil {
        return nil, err
    }

    // Cache the result in Redis
    err = rdb.Set(ctx, shortURL, originalURL, 0).Err()
    if err != nil {
        return nil, err
    }

    return &pb.ShortenURLResponse{ShortUrl: shortURL}, nil
}

func generateShortURL() string {
    return fmt.Sprintf("%06x", rand.Intn(1e6)) // 6-digit hex number
}

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

    // Set up gRPC server
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }
    grpcServer := grpc.NewServer()

    // Register gRPC service
    pb.RegisterURLShortenerServer(grpcServer, &URLShortenerServer{})

    log.Println("Starting gRPC URL shortener server on :50051...")
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
