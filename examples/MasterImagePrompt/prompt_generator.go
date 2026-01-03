package image

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
)

// CQAI LLM endpoint - update based on verification
const CQAI_LLM_ENDPOINT = "http://cqai.nlaakstudios/api/llm/generate"

// Master negative prompt - ALWAYS included to prevent text in images
const MASTER_NEGATIVE_PROMPT = `text, letters, words, typography, watermark, signature, logo, brand names, writing, captions, subtitles, title, credit, copyright notice, numbers, symbols, alphabet, characters, ui elements, overlays, labels, tags, readable signs, store names, street signs, billboards with text, posters with words, ugly, blurry, low quality, distorted, deformed, disfigured, cartoon, anime, CGI, artificial, fake, amateur, pixelated, grainy, noisy, oversaturated, undersaturated, washed out`

// LLM system prompt for generating image descriptions
const IMAGE_PROMPT_SYSTEM = `You are an expert cinematic photographer creating detailed image prompts for AI image generation.

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

// LLMRequest represents a request to the CQAI LLM
type LLMRequest struct {
    Model     string `json:"model"`
    Prompt    string `json:"prompt"`
    MaxTokens int    `json:"max_tokens"`
}

// LLMResponse represents the response from CQAI LLM
type LLMResponse struct {
    Text string `json:"text"`
}

// PromptComponents holds the elements for building an image prompt
type PromptComponents struct {
    Scene      string
    Location   string
    Subject    string
    Lighting   string
    Mood       string
    Colors     string
    Camera     string
    Quality    string
}

// GenerateImagePrompt creates a detailed image prompt from song lyrics using LLM
func GenerateImagePrompt(songTitle, genre, sectionType string, lyrics []string) (string, error) {
    // Build lyrics text
    lyricsText := strings.Join(lyrics, "\n")
    
    // Limit lyrics to prevent token overflow (approx 500 chars)
    if len(lyricsText) > 500 {
        lyricsText = lyricsText[:500] + "..."
    }
    
    // Create the user prompt
    userPrompt := fmt.Sprintf(`Song: "%s"
Genre: %s
Section: %s

Lyrics:
%s

Generate a cinematic, photorealistic image prompt that captures the visual essence of these lyrics. Remember: NO text or letters in the image.`, 
        songTitle, genre, sectionType, lyricsText)
    
    // Call CQAI LLM
    reqBody := LLMRequest{
        Model:     "qwen2.5:7b", // Update based on CQAI verification
        Prompt:    IMAGE_PROMPT_SYSTEM + "\n\n" + userPrompt,
        MaxTokens: 300,
    }
    
    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        return "", fmt.Errorf("failed to marshal request: %w", err)
    }
    
    resp, err := http.Post(CQAI_LLM_ENDPOINT, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return "", fmt.Errorf("failed to call CQAI LLM: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return "", fmt.Errorf("CQAI LLM returned status %d", resp.StatusCode)
    }
    
    var llmResp LLMResponse
    if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {
        return "", fmt.Errorf("failed to parse LLM response: %w", err)
    }
    
    // Clean up the response (remove any potential quotes or formatting)
    prompt := strings.TrimSpace(llmResp.Text)
    prompt = strings.Trim(prompt, "\"'")
    
    return prompt, nil
}

// BuildPromptFromComponents constructs a complete image prompt from components
// This is a fallback method if LLM is unavailable
func BuildPromptFromComponents(comp PromptComponents) string {
    parts := []string{}
    
    if comp.Scene != "" {
        parts = append(parts, comp.Scene)
    }
    if comp.Location != "" {
        parts = append(parts, "at "+comp.Location)
    }
    if comp.Subject != "" {
        parts = append(parts, comp.Subject)
    }
    if comp.Lighting != "" {
        parts = append(parts, comp.Lighting)
    }
    if comp.Mood != "" {
        parts = append(parts, comp.Mood+" atmosphere")
    }
    if comp.Colors != "" {
        parts = append(parts, comp.Colors+" color palette")
    }
    if comp.Camera != "" {
        parts = append(parts, "shot with "+comp.Camera)
    }
    
    // Always add quality terms
    qualityTerms := "photorealistic, professional photography, 8K resolution, ultra detailed, sharp focus, cinematic composition, award-winning photography"
    parts = append(parts, qualityTerms)
    
    return strings.Join(parts, ", ")
}

// GetMoodBasedPrompt returns a template prompt based on song mood
// Useful for quick generation without LLM
func GetMoodBasedPrompt(mood, sectionType string) PromptComponents {
    // Default values
    comp := PromptComponents{
        Camera:  "50mm lens at f/2.8, shallow depth of field",
        Quality: "photorealistic, 8K resolution, professional photography",
    }
    
    // Mood-specific configurations
    switch strings.ToLower(mood) {
    case "romantic", "love", "passion":
        comp.Scene = "Intimate romantic scene"
        comp.Lighting = "Golden hour sunlight, warm glow, soft rim lighting"
        comp.Mood = "Romantic and dreamy"
        comp.Colors = "Warm pinks, soft oranges, deep purples"
        comp.Location = "beach at sunset with gentle waves"
        
    case "sad", "melancholic", "heartbreak":
        comp.Scene = "Melancholic solitary scene"
        comp.Lighting = "Overcast sky, diffused grey light, moody shadows"
        comp.Mood = "Melancholic and introspective"
        comp.Colors = "Desaturated blues, cool greys, muted tones"
        comp.Location = "empty urban street in rain"
        
    case "happy", "upbeat", "energetic":
        comp.Scene = "Vibrant energetic scene"
        comp.Lighting = "Bright natural sunlight, vivid and clear"
        comp.Mood = "Energetic and joyful"
        comp.Colors = "Saturated vibrant colors, bright yellows, sky blues"
        comp.Location = "sunny beach or colorful city street"
        
    case "dark", "intense", "angry":
        comp.Scene = "Dramatic intense scene"
        comp.Lighting = "Low key lighting, harsh shadows, dramatic contrast"
        comp.Mood = "Intense and dramatic"
        comp.Colors = "Deep blacks, rich reds, dark purples"
        comp.Location = "dark urban alley or stormy landscape"
        
    case "mysterious", "ethereal":
        comp.Scene = "Mysterious ethereal scene"
        comp.Lighting = "Fog with volumetric light rays, mysterious glow"
        comp.Mood = "Mysterious and ethereal"
        comp.Colors = "Cool teals, deep blues, silver highlights"
        comp.Location = "misty forest or foggy cityscape"
        
    case "peaceful", "serene", "calm":
        comp.Scene = "Peaceful serene landscape"
        comp.Lighting = "Soft natural light, gentle morning glow"
        comp.Mood = "Serene and peaceful"
        comp.Colors = "Soft pastels, muted greens, calm blues"
        comp.Location = "tranquil lake or quiet meadow"
        
    default:
        // Generic fallback
        comp.Scene = "Cinematic scene"
        comp.Lighting = "Natural lighting, well-balanced exposure"
        comp.Mood = "Atmospheric and cinematic"
        comp.Colors = "Balanced color palette"
        comp.Location = "scenic outdoor location"
    }
    
    // Adjust based on section type
    if sectionType == "chorus" {
        // Chorus should be more dramatic/memorable
        comp.Lighting = strings.Replace(comp.Lighting, "Natural", "Dramatic", 1)
        comp.Camera = "85mm lens at f/1.8, beautiful bokeh, dramatic perspective"
    } else if sectionType == "verse" {
        // Verse can be more subdued
        comp.Camera = "50mm lens at f/2.8, natural perspective"
    } else if sectionType == "bridge" {
        // Bridge should be unique/different
        comp.Camera = "35mm lens, dynamic composition, unique angle"
    }
    
    return comp
}

// GetNegativePrompt returns the master negative prompt
func GetNegativePrompt() string {
    return MASTER_NEGATIVE_PROMPT
}

// ValidatePrompt checks if a prompt might contain forbidden elements
func ValidatePrompt(prompt string) bool {
    forbidden := []string{
        "text", "letters", "words", "writing", "typography",
        "watermark", "signature", "logo", "caption", "title",
    }
    
    lowerPrompt := strings.ToLower(prompt)
    for _, word := range forbidden {
        if strings.Contains(lowerPrompt, word) {
            return false
        }
    }
    
    return true
}

// CleanPrompt removes any potential text-related terms from a prompt
func CleanPrompt(prompt string) string {
    replacements := map[string]string{
        "with text":     "",
        "with words":    "",
        "with writing":  "",
        "sign saying":   "sign",
        "poster with":   "poster",
        "billboard":     "",
    }
    
    cleaned := prompt
    for old, new := range replacements {
        cleaned = strings.ReplaceAll(cleaned, old, new)
    }
    
    return cleaned
}
