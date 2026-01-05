package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/models"
)

// Client handles AI API calls for metadata enrichment
type Client struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewClient creates a new AI client using CQAI/Ollama
func NewClient() *Client {
	baseURL := os.Getenv("CQAI_URL")
	if baseURL == "" {
		baseURL = "http://cqai.nlaakstudios:11434"
	}

	model := os.Getenv("CQAI_LLM_MODEL")
	if model == "" {
		model = "qwen2.5:7b"
	}

	return &Client{
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: 120 * time.Second, // Longer timeout for local LLM
		},
	}
}

// EnrichSongMetadata generates AI-powered metadata for a song
func (c *Client) EnrichSongMetadata(song *models.Song) (*models.SongMetadataEnrichment, error) {

	// Build the prompt
	prompt, err := c.buildPrompt(song)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Call the LLM
	response, err := c.callLLM(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM: %w", err)
	}

	// Parse the response
	metadata, err := c.parseMetadata(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Validate primary genre
	if !models.IsValidGenre(metadata.GenrePrimary) {
		return nil, fmt.Errorf("invalid primary genre: %s (must be one of the 15 allowed genres)", metadata.GenrePrimary)
	}

	return metadata, nil
}

// buildPrompt creates the enrichment prompt from the template
func (c *Client) buildPrompt(song *models.Song) (string, error) {
	// Read the prompt template
	templatePath := "track-studio-docs/TODOs/AI-PROMPT-METADATA-ENRICHMENT.txt"
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		// Fallback to embedded prompt if file not found
		return c.buildEmbeddedPrompt(song), nil
	}

	template := string(templateBytes)

	// Replace placeholders
	prompt := strings.ReplaceAll(template, "{{BPM}}", fmt.Sprintf("%.1f", song.BPM))
	prompt = strings.ReplaceAll(prompt, "{{KEY}}", song.Key)
	prompt = strings.ReplaceAll(prompt, "{{TEMPO}}", song.Tempo)
	prompt = strings.ReplaceAll(prompt, "{{LYRICS}}", song.Lyrics)
	prompt = strings.ReplaceAll(prompt, "{{TITLE}}", song.Title)
	prompt = strings.ReplaceAll(prompt, "{{ARTIST}}", song.ArtistName)

	return prompt, nil
}

// buildEmbeddedPrompt creates a minimal prompt when template file is not available
func (c *Client) buildEmbeddedPrompt(song *models.Song) string {
	return fmt.Sprintf(`You are a professional music metadata analyst. Analyze this song and provide metadata as JSON.

Song: %s by %s
BPM: %.1f
Key: %s
Tempo: %s

Lyrics:
%s

Return ONLY a valid JSON object (no markdown, no explanations):
{
  "genre_primary": "One of: Pop, Rock, Hip-Hop/Rap, Country, R&B/Soul, Electronic/Dance, Latin, Metal, Jazz, Blues, Folk, Classical, Reggae, Gospel/Christian, Ballad",
  "genre_secondary": ["Genre2", "Genre3"],
  "tags": ["tag1", "tag2", "tag3", "tag4", "tag5", "tag6"],
  "style_descriptors": ["descriptor1", "descriptor2", "descriptor3"],
  "mood": ["mood1", "mood2", "mood3"],
  "themes": ["theme1", "theme2", "theme3"],
  "similar_artists": ["Artist1", "Artist2", "Artist3"],
  "summary": "2-3 sentence description",
  "target_audience": "Description of ideal listener",
  "energy_level": "Low|Medium-Low|Medium|Medium-High|High",
  "vocal_style": "Description of vocal delivery"
}`, song.Title, song.ArtistName, song.BPM, song.Key, song.Tempo, song.Lyrics)
}

// anthropicRequest represents the Claude API request structure
type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// anthropicResponse represents the Claude API response structure
type ollamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// callLLM sends the prompt to CQAI/Ollama and returns the response
func (c *Client) callLLM(prompt string) (string, error) {
	reqBody := ollamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp ollamaResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.Response == "" {
		return "", fmt.Errorf("empty response from API")
	}

	return apiResp.Response, nil
}

// parseMetadata parses the LLM JSON response into metadata struct
func (c *Client) parseMetadata(response string) (*models.SongMetadataEnrichment, error) {
	// Clean response - remove markdown code blocks if present
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var metadata models.SongMetadataEnrichment
	if err := json.Unmarshal([]byte(response), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Validate required fields
	if metadata.GenrePrimary == "" {
		return nil, fmt.Errorf("missing required field: genre_primary")
	}
	if len(metadata.Tags) == 0 {
		return nil, fmt.Errorf("missing required field: tags")
	}
	if metadata.Summary == "" {
		return nil, fmt.Errorf("missing required field: summary")
	}

	return &metadata, nil
}
