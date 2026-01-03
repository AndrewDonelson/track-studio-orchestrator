package image

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	CQAI_BASE_URL  = "http://cqai.nlaakstudios"       // z-image API
	CQAI_LLM_URL   = "http://cqai.nlaakstudios:11434" // Ollama API for LLM
	IMAGE_MODEL    = "z-image-nsfw"
	LLM_MODEL      = "qwen2.5:7b"
	DEFAULT_WIDTH  = 1920
	DEFAULT_HEIGHT = 1024
	DEFAULT_STEPS  = 25

	// Master negative prompt - ALWAYS included to prevent text in images
	MASTER_NEGATIVE_PROMPT = `text, letters, words, typography, watermark, signature, logo, brand names, writing, captions, subtitles, title, credit, copyright notice, numbers, symbols, alphabet, characters, ui elements, overlays, labels, tags, readable signs, store names, street signs, billboards with text, posters with words, ugly, blurry, low quality, distorted, deformed, disfigured, cartoon, anime, CGI, artificial, fake, amateur, pixelated, grainy, noisy, oversaturated, undersaturated, washed out`

	// LLM system prompt for generating cinematic image descriptions
	IMAGE_PROMPT_SYSTEM = `You are an expert cinematic photographer creating detailed image prompts for AI image generation.

CRITICAL RULES:
1. NEVER include text, letters, words, or any written content in the image description
2. Create photorealistic, cinematic scenes only
3. Be extremely specific about visual details
4. Always include: scene, location, lighting, mood, colors, and camera details
5. Output length: 150-200 words
6. Professional photography quality

STRUCTURE YOUR RESPONSE:
[Vivid scene description] at [specific location with details], [subject and action if any], [detailed lighting description with source and quality], [atmospheric mood], [specific color palette with 3-5 colors], shot with [camera lens and settings], [composition style], photorealistic, professional photography, 8K resolution, ultra detailed, sharp focus, cinematic composition, award-winning photography

EXAMPLE:
"Beautiful beach at golden hour at Miami coastline with distant palm trees and gentle waves, woman in flowing white dress standing at water's edge with back to camera, dramatic golden hour sunlight streaming through clouds creating warm rim lighting, romantic and dreamy atmosphere, warm color palette with deep oranges, soft pinks, and purple sky gradients, shot with 85mm lens at f/2.8 creating shallow depth of field from low angle emphasizing dramatic sky, rule of thirds composition, photorealistic, professional photography, 8K resolution, ultra detailed, sharp focus, cinematic composition, award-winning photography"

DO NOT include any preamble or explanation - output ONLY the image prompt.`
)

type ImageGenerator struct {
	BaseURL    string
	LLMURL     string
	ImageModel string
	LLMModel   string
	OutputDir  string
	Width      int
	Height     int
	Steps      int
	Timeout    time.Duration

	// Timing statistics for adaptive timeouts and ETAs
	LLMTimings       []time.Duration
	ImageTimings     []time.Duration
	MaxTimingSamples int
}

// LLM request/response (Ollama API)
type LLMRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type LLMResponse struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Response  string    `json:"response"`
	Done      bool      `json:"done"`
}

// z-image API request/response
type ZImageRequest struct {
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	Model          string `json:"model"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	Steps          int    `json:"steps"`
}

type ZImageResponse struct {
	Image          string  `json:"image"` // base64 encoded PNG
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	Steps          int     `json:"steps"`
	GenerationTime float64 `json:"generation_time"` // seconds
	Error          string  `json:"error,omitempty"`
}

func NewImageGenerator(outputDir string) *ImageGenerator {
	return &ImageGenerator{
		BaseURL:          CQAI_BASE_URL,
		LLMURL:           CQAI_LLM_URL,
		ImageModel:       IMAGE_MODEL,
		LLMModel:         LLM_MODEL,
		OutputDir:        outputDir,
		Width:            DEFAULT_WIDTH,
		Height:           DEFAULT_HEIGHT,
		Steps:            DEFAULT_STEPS,
		Timeout:          300 * time.Second, // 5 minutes for image generation
		LLMTimings:       make([]time.Duration, 0),
		ImageTimings:     make([]time.Duration, 0),
		MaxTimingSamples: 10, // Keep last 10 samples for rolling average
	}
}

func (ig *ImageGenerator) EnhancePromptWithLLM(sectionType, lyricsContent, styleKeywords string) (string, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		ig.LLMTimings = append(ig.LLMTimings, duration)
		if len(ig.LLMTimings) > ig.MaxTimingSamples {
			ig.LLMTimings = ig.LLMTimings[1:]
		}
	}()

	// Limit lyrics to prevent token overflow (approx 500 chars)
	if len(lyricsContent) > 500 {
		lyricsContent = lyricsContent[:500] + "..."
	}

	// Create cinematic image prompt using MasterImagePrompt template
	userPrompt := fmt.Sprintf(`Song Section: %s
Additional Style: %s

Lyrics:
%s

Generate a cinematic, photorealistic image prompt that captures the visual essence of these lyrics. Remember: NO text or letters in the image.`,
		sectionType,
		styleKeywords,
		lyricsContent)

	req := LLMRequest{
		Model:  ig.LLMModel,
		Prompt: IMAGE_PROMPT_SYSTEM + "\n\n" + userPrompt,
		Stream: false,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal LLM request: %w", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Post(
		ig.LLMURL+"/api/generate",
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return "", fmt.Errorf("LLM request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLM API error %d: %s", resp.StatusCode, string(body))
	}

	var llmResp LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {
		return "", fmt.Errorf("failed to decode LLM response: %w", err)
	}

	// Clean up the response (remove any potential quotes or formatting)
	enhancedPrompt := strings.TrimSpace(llmResp.Response)
	enhancedPrompt = strings.Trim(enhancedPrompt, "\"'")

	return enhancedPrompt, nil
}

func (ig *ImageGenerator) GenerateImage(prompt, outputFilename string) (string, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		ig.ImageTimings = append(ig.ImageTimings, duration)
		if len(ig.ImageTimings) > ig.MaxTimingSamples {
			ig.ImageTimings = ig.ImageTimings[1:]
		}
	}()

	if err := os.MkdirAll(ig.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Add quality modifiers and negative prompt handling
	enhancedPrompt := fmt.Sprintf("%s, photorealistic, professional photography, 8K resolution, ultra detailed, sharp focus, cinematic composition, award-winning photography", prompt)

	req := ZImageRequest{
		Prompt:         enhancedPrompt,
		NegativePrompt: MASTER_NEGATIVE_PROMPT,
		Model:          ig.ImageModel,
		Width:          ig.Width,
		Height:         ig.Height,
		Steps:          ig.Steps,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal image request: %w", err)
	}

	// Calculate adaptive timeout: average + 20% buffer, minimum 60s
	timeout := ig.Timeout
	if avgTime := ig.GetAverageImageTime(); avgTime > 0 {
		timeout = time.Duration(float64(avgTime) * 1.2)
		if timeout < 60*time.Second {
			timeout = 60 * time.Second
		}
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Post(
		ig.BaseURL+"/api/zimage/generate",
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return "", fmt.Errorf("image generation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("image API error %d: %s", resp.StatusCode, string(body))
	}

	var imgResp ZImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&imgResp); err != nil {
		return "", fmt.Errorf("failed to decode image response: %w", err)
	}

	if imgResp.Error != "" {
		return "", fmt.Errorf("image generation error: %s", imgResp.Error)
	}

	if imgResp.Image == "" {
		return "", fmt.Errorf("no image data returned from API")
	}

	imageData, err := base64.StdEncoding.DecodeString(imgResp.Image)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 image: %w", err)
	}

	outputPath := filepath.Join(ig.OutputDir, outputFilename)
	if err := os.WriteFile(outputPath, imageData, 0644); err != nil {
		return "", fmt.Errorf("failed to write image file: %w", err)
	}

	fmt.Printf("Image generated: %dx%d, %d steps, %.2fs\n",
		imgResp.Width, imgResp.Height, imgResp.Steps, imgResp.GenerationTime)
	fmt.Printf("Image saved: %s\n", outputPath)
	return outputPath, nil
}

func (ig *ImageGenerator) GenerateFromSection(sectionType string, sectionNumber int, lyrics, styleKeywords string) (string, error) {
	var filename string
	switch sectionType {
	case "verse":
		filename = fmt.Sprintf("bg-verse-%d.png", sectionNumber)
	case "pre-chorus":
		filename = "bg-prechorus.png"
	case "chorus":
		filename = "bg-chorus.png"
	case "bridge":
		filename = "bg-bridge.png"
	case "intro":
		filename = "bg-intro.png"
	case "outro":
		filename = "bg-outro.png"
	default:
		filename = fmt.Sprintf("bg-%s-%d.png", sectionType, sectionNumber)
	}

	outputPath := filepath.Join(ig.OutputDir, filename)
	if _, err := os.Stat(outputPath); err == nil {
		return outputPath, nil
	}

	fmt.Printf("Enhancing prompt for %s %d with LLM...\n", sectionType, sectionNumber)
	enhancedPrompt, err := ig.EnhancePromptWithLLM(sectionType, lyrics, styleKeywords)
	if err != nil {
		return "", fmt.Errorf("failed to enhance prompt: %w", err)
	}

	promptPreview := enhancedPrompt
	if len(promptPreview) > 100 {
		promptPreview = promptPreview[:100] + "..."
	}
	fmt.Printf("Enhanced prompt: %s\n", promptPreview)

	fmt.Printf("Generating image for %s %d...\n", sectionType, sectionNumber)
	imagePath, err := ig.GenerateImage(enhancedPrompt, filename)
	if err != nil {
		return "", fmt.Errorf("failed to generate image: %w", err)
	}

	fmt.Printf("Image saved: %s\n", imagePath)
	return imagePath, nil
}

// GetAverageLLMTime returns the average time for LLM prompt enhancement
func (ig *ImageGenerator) GetAverageLLMTime() time.Duration {
	if len(ig.LLMTimings) == 0 {
		return 0
	}
	var total time.Duration
	for _, t := range ig.LLMTimings {
		total += t
	}
	return total / time.Duration(len(ig.LLMTimings))
}

// GetAverageImageTime returns the average time for image generation
func (ig *ImageGenerator) GetAverageImageTime() time.Duration {
	if len(ig.ImageTimings) == 0 {
		return 0
	}
	var total time.Duration
	for _, t := range ig.ImageTimings {
		total += t
	}
	return total / time.Duration(len(ig.ImageTimings))
}

// EstimateRemainingTime estimates time for remaining images based on averages
func (ig *ImageGenerator) EstimateRemainingTime(remainingImages int) time.Duration {
	avgLLM := ig.GetAverageLLMTime()
	avgImage := ig.GetAverageImageTime()

	// If no data yet, use reasonable defaults: 5s LLM + 60s image
	if avgLLM == 0 {
		avgLLM = 5 * time.Second
	}
	if avgImage == 0 {
		avgImage = 60 * time.Second
	}

	perImageTime := avgLLM + avgImage
	return perImageTime * time.Duration(remainingImages)
}

// GetTimingStats returns timing statistics as a formatted string
func (ig *ImageGenerator) GetTimingStats() string {
	avgLLM := ig.GetAverageLLMTime()
	avgImage := ig.GetAverageImageTime()

	if avgLLM == 0 && avgImage == 0 {
		return "No timing data yet"
	}

	return fmt.Sprintf("Avg LLM: %.1fs, Avg Image: %.1fs (samples: %d LLM, %d Image)",
		avgLLM.Seconds(), avgImage.Seconds(), len(ig.LLMTimings), len(ig.ImageTimings))
}

func BuildStyleKeywords(genre, backgroundStyle string) string {
	keywords := []string{backgroundStyle, "cinematic", "professional photography"}

	switch strings.ToLower(genre) {
	case "romantic pop", "romantic", "pop":
		keywords = append(keywords, "romantic lighting", "warm tones", "intimate atmosphere")
	case "electronic", "edm":
		keywords = append(keywords, "vibrant colors", "neon lights", "futuristic")
	case "rock", "metal":
		keywords = append(keywords, "dramatic lighting", "high contrast", "intense")
	case "hip hop", "rap":
		keywords = append(keywords, "urban setting", "street photography", "bold")
	case "country":
		keywords = append(keywords, "natural lighting", "outdoor scenery", "authentic")
	default:
		keywords = append(keywords, "beautiful composition", "artistic")
	}

	return strings.Join(keywords, ", ")
}
