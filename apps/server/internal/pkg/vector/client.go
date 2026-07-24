package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Config for embedding + qdrant.
type Config struct {
	QdrantURL      string
	Collection     string
	EmbeddingURL   string
	EmbeddingKey   string
	EmbeddingModel string
	Dimension      int
	Threshold      float64
	MaxResults     int
}

func (c Config) Normalize() Config {
	if c.Collection == "" {
		c.Collection = "websites"
	}
	if c.EmbeddingModel == "" {
		c.EmbeddingModel = "text-embedding-3-small"
	}
	if c.Dimension <= 0 {
		c.Dimension = 1536
	}
	if c.Threshold <= 0 {
		c.Threshold = 0.3
	}
	if c.MaxResults <= 0 {
		c.MaxResults = 50
	}
	c.QdrantURL = strings.TrimRight(strings.TrimSpace(c.QdrantURL), "/")
	c.EmbeddingURL = strings.TrimRight(strings.TrimSpace(c.EmbeddingURL), "/")
	return c
}

type Client struct {
	cfg    Config
	http   *http.Client
}

func NewClient(cfg Config) *Client {
	cfg = cfg.Normalize()
	return &Client{
		cfg: cfg,
		http: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) Config() Config { return c.cfg }

// Embed generates embedding for text via OpenAI-compatible API.
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	if c.cfg.EmbeddingURL == "" || c.cfg.EmbeddingKey == "" {
		return nil, fmt.Errorf("embedding api not configured")
	}
	url := c.cfg.EmbeddingURL + "/v1/embeddings"
	body := map[string]any{
		"model": c.cfg.EmbeddingModel,
		"input": text,
	}
	var out struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := c.postJSON(ctx, url, c.cfg.EmbeddingKey, body, &out); err != nil {
		return nil, err
	}
	if out.Error != nil {
		return nil, fmt.Errorf("embedding: %s", out.Error.Message)
	}
	if len(out.Data) == 0 || len(out.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding")
	}
	// update dimension if auto
	if c.cfg.Dimension == 1536 && len(out.Data[0].Embedding) != 1536 {
		c.cfg.Dimension = len(out.Data[0].Embedding)
	}
	return out.Data[0].Embedding, nil
}

// EnsureCollection creates collection if missing.
func (c *Client) EnsureCollection(ctx context.Context) error {
	if c.cfg.QdrantURL == "" {
		return fmt.Errorf("qdrant url not configured")
	}
	// GET collections
	var list struct {
		Result struct {
			Collections []struct {
				Name string `json:"name"`
			} `json:"collections"`
		} `json:"result"`
	}
	if err := c.getJSON(ctx, c.cfg.QdrantURL+"/collections", &list); err != nil {
		return err
	}
	for _, col := range list.Result.Collections {
		if col.Name == c.cfg.Collection {
			return nil
		}
	}
	// create
	body := map[string]any{
		"vectors": map[string]any{
			"size":     c.cfg.Dimension,
			"distance": "Cosine",
		},
	}
	return c.putJSON(ctx, fmt.Sprintf("%s/collections/%s", c.cfg.QdrantURL, c.cfg.Collection), body, nil)
}

// UpsertPoint writes one website vector.
func (c *Client) UpsertPoint(ctx context.Context, id int64, vector []float32, payload map[string]any) error {
	if err := c.EnsureCollection(ctx); err != nil {
		return err
	}
	body := map[string]any{
		"points": []map[string]any{
			{
				"id":      id,
				"vector":  vector,
				"payload": payload,
			},
		},
	}
	return c.putJSON(ctx, fmt.Sprintf("%s/collections/%s/points?wait=true", c.cfg.QdrantURL, c.cfg.Collection), body, nil)
}

// DeletePoint removes a point by id.
func (c *Client) DeletePoint(ctx context.Context, id int64) error {
	body := map[string]any{"points": []int64{id}}
	return c.postJSON(ctx, fmt.Sprintf("%s/collections/%s/points/delete?wait=true", c.cfg.QdrantURL, c.cfg.Collection), "", body, nil)
}

// DeletePoints removes multiple points by id (best-effort batch).
func (c *Client) DeletePoints(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	body := map[string]any{"points": ids}
	return c.postJSON(ctx, fmt.Sprintf("%s/collections/%s/points/delete?wait=true", c.cfg.QdrantURL, c.cfg.Collection), "", body, nil)
}

// ClearCollection wipes the collection by recreate (legacy: clear all site vectors).
func (c *Client) ClearCollection(ctx context.Context) error {
	if c.cfg.QdrantURL == "" {
		return fmt.Errorf("qdrant url not configured")
	}
	// drop collection if exists, then EnsureCollection rebuilds empty
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/collections/%s", c.cfg.QdrantURL, c.cfg.Collection), nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	// 404 = already gone — fine
	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		return fmt.Errorf("delete collection: status %d", resp.StatusCode)
	}
	return c.EnsureCollection(ctx)
}

// SearchResult from qdrant.
type SearchResult struct {
	ID    int64
	Score float64
}

// Search by vector.
func (c *Client) Search(ctx context.Context, vector []float32, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = c.cfg.MaxResults
	}
	body := map[string]any{
		"vector":       vector,
		"limit":        limit,
		"with_payload": false,
		"score_threshold": c.cfg.Threshold,
	}
	var out struct {
		Result []struct {
			ID    any     `json:"id"`
			Score float64 `json:"score"`
		} `json:"result"`
	}
	if err := c.postJSON(ctx, fmt.Sprintf("%s/collections/%s/points/search", c.cfg.QdrantURL, c.cfg.Collection), "", body, &out); err != nil {
		return nil, err
	}
	var res []SearchResult
	for _, r := range out.Result {
		id := toInt64(r.ID)
		if id > 0 {
			res = append(res, SearchResult{ID: id, Score: r.Score})
		}
	}
	return res, nil
}

// PingQdrant checks connectivity.
func (c *Client) PingQdrant(ctx context.Context) error {
	var out map[string]any
	return c.getJSON(ctx, c.cfg.QdrantURL+"/collections", &out)
}

// PingEmbedding sends a tiny embed request.
func (c *Client) PingEmbedding(ctx context.Context) error {
	_, err := c.Embed(ctx, "ping")
	return err
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case int:
		return int64(t)
	case json.Number:
		i, _ := t.Int64()
		return i
	default:
		return 0
	}
}

func (c *Client) getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("GET %s: %s %s", url, resp.Status, string(b))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(b, out)
}

func (c *Client) putJSON(ctx context.Context, url string, body any, out any) error {
	return c.doJSON(ctx, http.MethodPut, url, "", body, out)
}

func (c *Client) postJSON(ctx context.Context, url, bearer string, body any, out any) error {
	return c.doJSON(ctx, http.MethodPost, url, bearer, body, out)
}

func (c *Client) doJSON(ctx context.Context, method, url, bearer string, body any, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: %s %s", method, url, resp.Status, string(b))
	}
	if out == nil || len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, out)
}
