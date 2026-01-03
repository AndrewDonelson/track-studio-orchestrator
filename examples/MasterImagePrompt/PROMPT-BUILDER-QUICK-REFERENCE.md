# TrackStudio Prompt Builder - Quick Reference

## Simple Prompt Formula

```
[SCENE] at [LOCATION], [SUBJECT_ACTION], [LIGHTING], [MOOD] atmosphere, 
[COLOR_PALETTE], shot with [CAMERA_SPECS], [QUALITY_TERMS], photorealistic, 
8K resolution, highly detailed, cinematic composition
```

---

## Fill-in-the-Blank Template

### For LLM to Complete:

```json
{
  "scene": "Beautiful beach at sunset",
  "location": "Miami beach with palm trees in background",
  "subject": "Woman in flowing dress standing at water's edge, back to camera",
  "lighting": "Golden hour sunlight, dramatic sky, warm glow",
  "mood": "Romantic and dreamy",
  "colors": "Warm oranges, soft pinks, deep purple sky",
  "camera": "85mm lens, f/2.8, shallow depth of field, low angle",
  "quality": "Professional photography, 8K resolution, award-winning"
}
```

**Generated Prompt** (auto-assembled):
```
Beautiful beach at sunset at Miami beach with palm trees in background, 
woman in flowing dress standing at water's edge back to camera, golden 
hour sunlight with dramatic sky and warm glow, romantic and dreamy 
atmosphere, warm oranges soft pinks and deep purple sky color palette, 
shot with 85mm lens at f/2.8 shallow depth of field from low angle, 
professional photography 8K resolution award-winning, photorealistic, 
highly detailed, cinematic composition
```

---

## Quick Categories

### Lighting Options
```
- Golden hour (sunset/sunrise warm light)
- Blue hour (twilight cool tones)
- Midday harsh sunlight
- Overcast soft diffused light
- Neon lights (urban night)
- Candlelight/firelight (intimate warm)
- Moonlight (cool mysterious)
- Volumetric light rays
- Rim lighting (backlit subject)
- Window light (natural indoor)
```

### Mood Options
```
- Romantic and dreamy
- Melancholic and introspective  
- Energetic and vibrant
- Mysterious and ethereal
- Serene and peaceful
- Intense and dramatic
- Nostalgic and warm
- Dark and moody
- Hopeful and uplifting
- Intimate and tender
```

### Color Palettes
```
Warm:
- "Warm oranges, golden yellows, deep reds"
- "Amber tones, sunset colors, warm earth tones"

Cool:
- "Deep blues, cool teals, purple shadows"
- "Icy blues, silver highlights, cool greys"

Vibrant:
- "Neon pinks, electric blues, vivid purples"
- "Saturated tropical colors, bright turquoise, coral"

Muted:
- "Desaturated pastels, soft lavenders, muted greens"
- "Earth tones, beige, olive, dusty rose"

Dramatic:
- "High contrast blacks and whites, deep shadows"
- "Rich jewel tones, emerald, sapphire, ruby"
```

### Camera Specs
```
Wide shots:
- "16mm ultra-wide lens, expansive view, dramatic perspective"
- "24mm wide angle, full scene capture"

Standard:
- "35mm lens, natural perspective, street photography style"
- "50mm lens, human eye perspective, balanced composition"

Portrait:
- "85mm lens, f/1.8, beautiful bokeh, subject isolation"
- "105mm lens, f/2.0, compressed background, shallow DOF"

Telephoto:
- "200mm lens, compressed perspective, extreme bokeh"
```

---

## ALWAYS Include (Copy-Paste)

### Positive Prompt Ending:
```
photorealistic, professional photography, 8K resolution, ultra detailed, 
sharp focus, cinematic composition, award-winning photography
```

### Negative Prompt (Use Every Time):
```
text, letters, words, typography, watermark, signature, logo, brand names, 
writing, captions, numbers, symbols, alphabet, characters, readable signs, 
ugly, blurry, low quality, distorted, deformed, cartoon, anime, CGI, 
artificial, amateur, pixelated
```

---

## 10-Second Prompt Builder

**Step 1**: What's the scene?
- Beach sunset
- City street at night  
- Mountain landscape
- Desert highway
- Tropical island

**Step 2**: Add lighting
- Golden hour → "warm sunset glow, dramatic sky"
- Night → "neon lights, bokeh from street lamps"
- Day → "soft natural sunlight, bright and airy"

**Step 3**: Add mood + colors
- Romantic → "dreamy atmosphere, warm pinks and oranges"
- Dark → "moody atmosphere, deep blues and blacks"
- Energetic → "vibrant atmosphere, saturated colors"

**Step 4**: Add camera
- "50mm lens, shallow depth of field" (general)
- "Wide angle, expansive view" (landscapes)
- "85mm, f/1.8, beautiful bokeh" (portraits)

**Step 5**: Paste endings
- Positive ending (quality terms)
- Negative prompt (anti-text)

**Done!** You have a complete prompt.

---

## Code Integration Example

```go
func GeneratePrompt(scene, location, mood, colors string) string {
    positive := fmt.Sprintf(
        "%s at %s, %s atmosphere, %s color palette, " +
        "shot with 50mm lens at f/2.8, photorealistic, professional " +
        "photography, 8K resolution, ultra detailed, sharp focus, " +
        "cinematic composition, award-winning photography",
        scene, location, mood, colors,
    )
    
    negative := "text, letters, words, typography, watermark, " +
        "signature, logo, brand names, writing, captions, numbers, " +
        "symbols, alphabet, characters, ugly, blurry, low quality, " +
        "distorted, cartoon, anime, CGI, amateur"
    
    return positive
}
```

---

## LLM System Prompt (Simplified)

```
You generate image prompts for AI image generation.

Rules:
1. NO text/letters/words in images
2. Be specific and detailed
3. Use photorealistic style
4. Include: scene, location, lighting, mood, colors, camera details
5. Output 100-150 words
6. Professional photography quality

Format:
[Scene] at [Location], [Lighting], [Mood] atmosphere, 
[Colors], shot with [Camera], photorealistic, 8K, detailed

Example:
"Beach at sunset at Miami with palm trees, golden hour lighting 
casting warm glow, romantic atmosphere, warm oranges and pinks, 
shot with 85mm lens f/2.8, photorealistic, 8K resolution"
```

---

## Quick Cheat Sheet

| Song Mood | Lighting | Colors | Location |
|-----------|----------|--------|----------|
| Happy | Golden hour | Warm yellows, oranges | Beach, field |
| Sad | Overcast | Cool blues, greys | Rain, empty room |
| Romantic | Sunset/Candlelight | Pinks, purples | Beach, rooftop |
| Energetic | Bright/Neon | Vibrant saturated | City, club |
| Dark | Low key/Night | Deep blues, blacks | Alley, shadows |
| Peaceful | Soft natural | Pastels, muted | Lake, garden |
| Mysterious | Fog/Mist | Desaturated, cool | Forest, street |
| Intense | Harsh/Dramatic | High contrast | Desert, mountain |

---

**Use this for quick prompt generation in TrackStudio!**
