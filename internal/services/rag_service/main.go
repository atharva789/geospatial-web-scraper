package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type searchResult struct {
	URL       string  `json:"url"`
	Label     string  `json:"label"`
	Rationale string  `json:"rationale"`
	Score     float64 `json:"score"`
}

type embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type bedrockEmbedder struct {
	client   *bedrockruntime.Client
	model    string
	fallback embedder
}

func (b *bedrockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if b.client == nil || b.model == "" {
		return b.fallback.Embed(ctx, text)
	}
	body := map[string]any{
		"inputText": text,
	}
	payload, _ := json.Marshal(body)
	resp, err := b.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(b.model),
		ContentType: aws.String("application/json"),
		Body:        payload,
	})
	if err != nil {
		return b.fallback.Embed(ctx, text)
	}
	var parsed struct {
		Embedding []float32 `json:"embedding"`
		Vector    []float32 `json:"vector"`
	}
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		return b.fallback.Embed(ctx, text)
	}
	if len(parsed.Embedding) > 0 {
		return parsed.Embedding, nil
	}
	if len(parsed.Vector) > 0 {
		return parsed.Vector, nil
	}
	return b.fallback.Embed(ctx, text)
}

type hashEmbedder struct {
	dim int
}

func (h hashEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if h.dim <= 0 {
		h.dim = 1536
	}
	sum := sha256.Sum256([]byte(text))
	bytes := sum[:]
	out := make([]float32, h.dim)
	for i := 0; i < h.dim; i++ {
		out[i] = float32(bytes[i%len(bytes)]) / 255.0
	}
	return out, nil
}

type server struct {
	pool        *pgxpool.Pool
	table       string
	vectorDim   int
	embedClient embedder
	origins     []string
}

func sanitizeTable(name string) string {
	if name == "" {
		name = "dataset_vectors"
	}
	return pgx.Identifier{name}.Sanitize()
}

func ensureSchema(ctx context.Context, pool *pgxpool.Pool, table string, dim int) error {
	query := fmt.Sprintf(`
		CREATE EXTENSION IF NOT EXISTS vector;
		CREATE TABLE IF NOT EXISTS %s (
			id BIGSERIAL PRIMARY KEY,
			dataset_url TEXT UNIQUE NOT NULL,
			label TEXT,
			rationale TEXT,
			embedding VECTOR(%d),
			created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS %s_embedding_idx ON %s USING ivfflat (embedding vector_cosine_ops);
	`, table, dim, table, table)
	_, err := pool.Exec(ctx, query)
	return err
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (s *server) handleReady(w http.ResponseWriter, r *http.Request) {
	if err := s.pool.Ping(r.Context()); err != nil {
		http.Error(w, "db not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}
func (s *server) handleSearch(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r, s.origins)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		http.Error(w, "missing q", http.StatusBadRequest)
		return
	}
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 50 {
			limit = v
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	vecVals, err := s.embedClient.Embed(ctx, q)
	if err != nil {
		http.Error(w, "embedding failed", http.StatusInternalServerError)
		return
	}
	if len(vecVals) != s.vectorDim {
		// pad or trim
		if len(vecVals) < s.vectorDim {
			padded := make([]float32, s.vectorDim)
			copy(padded, vecVals)
			vecVals = padded
		} else {
			vecVals = vecVals[:s.vectorDim]
		}
	}
	vec := pgvector.NewVector(vecVals)

	query := fmt.Sprintf(`
		SELECT dataset_url, label, rationale, embedding <-> $1 AS distance
		FROM %s
		ORDER BY embedding <-> $1
		LIMIT $2;
	`, s.table)

	rows, err := s.pool.Query(ctx, query, vec, limit)
	if err != nil {
		http.Error(w, "db query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []searchResult
	for rows.Next() {
		var url, label, rationale string
		var distance float64
		if err := rows.Scan(&url, &label, &rationale, &distance); err != nil {
			continue
		}
		score := 1 / (1 + distance)
		results = append(results, searchResult{
			URL:       url,
			Label:     label,
			Rationale: rationale,
			Score:     score,
		})
	}
	if rows.Err() != nil {
		http.Error(w, "db rows error", http.StatusInternalServerError)
		return
	}

	resp := map[string]any{
		"count":   len(results),
		"results": results,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func enableCORS(w http.ResponseWriter, r *http.Request, origins []string) {
	origin := r.Header.Get("Origin")
	for _, allowed := range origins {
		if allowed == "*" || allowed == origin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			return
		}
	}
}

func buildDBURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	host := os.Getenv("VECTORDB_HOST")
	name := os.Getenv("VECTORDB_NAME")
	user := os.Getenv("VECTORDB_USER")
	pass := os.Getenv("VECTORDB_PASSWORD")
	port := os.Getenv("VECTORDB_PORT")
	if port == "" {
		port = "5432"
	}
	if host == "" || name == "" || user == "" {
		return ""
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, name)
}

func makeEmbedder() embedder {
	fallback := hashEmbedder{dim: parseIntEnv("VECTOR_DIM", 1536)}
	model := os.Getenv("BEDROCK_EMBED_MODEL")
	region := os.Getenv("AWS_REGION")
	if model == "" || region == "" {
		return fallback
	}
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return fallback
	}
	client := bedrockruntime.NewFromConfig(cfg)
	return &bedrockEmbedder{
		client:   client,
		model:    model,
		fallback: fallback,
	}
}

func parseIntEnv(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func parseOrigins() []string {
	raw := os.Getenv("CORS_ORIGINS")
	if raw == "" {
		return []string{"*"}
	}
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func main() {
	addr := os.Getenv("RAG_LISTEN_ADDR")
	if addr == "" {
		addr = ":8082"
	}

	dbURL := buildDBURL()
	if dbURL == "" {
		log.Fatal("database connection env vars not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	s := &server{
		pool:        pool,
		table:       sanitizeTable(os.Getenv("VECTORDB_TABLE")),
		vectorDim:   parseIntEnv("VECTOR_DIM", 1536),
		embedClient: makeEmbedder(),
		origins:     parseOrigins(),
	}

	if err := ensureSchema(ctx, s.pool, s.table, s.vectorDim); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}

	if strings.EqualFold(os.Getenv("SEED_DEMO_DATA"), "true") {
		if err := SeedDemo(ctx, s.pool, s.table, s.vectorDim); err != nil {
			log.Printf("seed data error: %v", err)
		}
	}

	// Start background sync worker to vectorize discovered datasets
	go startSyncWorker(s.pool, s.table, s.embedClient, s.vectorDim)

	http.HandleFunc("/healthz", s.handleHealth)
	http.HandleFunc("/search", s.handleSearch)
	http.HandleFunc("/ready", s.handleReady)

	srv := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("rag backend listening on %s", addr)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("Shutting down rag backend...")
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctxTimeout); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Println("RAG backend stopped.")
}
