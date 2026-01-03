# Image Generation Logic for Lyrics Sections

## Overview

Each section of a song gets a background image generated from its lyrics content. Sections of the same type can **reuse** the same background image.

## Rules

### Unique Images Per Section Type

1. **Verses** - Each verse gets its own unique image
   - Verse 1 → `bg-verse-1.png`
   - Verse 2 → `bg-verse-2.png`
   - Verse 3 → `bg-verse-3.png` (if exists)

2. **Pre-Choruses** - All pre-choruses share the same image
   - Pre-Chorus (1st occurrence) → `bg-prechorus.png`
   - Pre-Chorus (2nd occurrence) → `bg-prechorus.png` (reused)

3. **Choruses** - All choruses share the same image
   - Chorus (1st occurrence) → `bg-chorus.png`
   - Chorus (2nd occurrence) → `bg-chorus.png` (reused)
   - Chorus (3rd occurrence) → `bg-chorus.png` (reused)

4. **Bridge** - Gets its own unique image
   - Bridge → `bg-bridge.png`

5. **Intro/Outro** - Get their own unique images (if present)
   - Intro → `bg-intro.png`
   - Outro → `bg-outro.png`

## Example: "Land of Love (Cover Hung)"

### Song Structure (7 sections, 5 unique images)

```
[Verse 1] → bg-verse-1.png (NEW #1)
In the city streets of Saigon, where the night air whispers low
I saw you standing there, like a vision from above
Your eyes, like the Mekong River, flowing deep and wide
And I knew in that moment, my heart would be yours to reside

[Pre-Chorus] → bg-prechorus.png (NEW #2)
Oh, my love, with skin as smooth as silk
You're the moonlight on the Perfume River, my heart's only milk
In your eyes, I see a love so true
A love that's worth waiting for, a love that's meant for me and you

[Chorus] → bg-chorus.png (NEW #3)
I've been searching for a love like yours, in every place I roam
But none have ever touched my heart, like the way you make me feel at home
In your arms, is where I belong
You are my land of love, my heart beats for you alone

[Verse 2] → bg-verse-2.png (NEW #4)
We'd walk along the beach, in Nha Trang's golden light
And I'd tell you stories of my dreams, and the love we'd ignite
You'd smile and laugh, and my heart would skip a beat
And I knew in that moment, our love would forever be unique

[Pre-Chorus] → bg-prechorus.png (REUSED)
Oh, my love, with skin as smooth as silk
You're the moonlight on the Perfume River, my heart's only milk
In your eyes, I see a love so true
A love that's worth waiting for, a love that's meant for me and you

[Bridge] → bg-bridge.png (NEW #5)
We'll dance beneath the stars, on a warm summer night
With gentle music playing, and incense drifting through the air tonight
And I'll whisper "I love you" softly in your ear
And you'll whisper back, "I love you" for me to hear

[Chorus] → bg-chorus.png (REUSED)
I've been searching for a love like yours, in every place I roam
But none have ever touched my heart, like the way you make me feel at home
In your arms, is where I belong
You are my land of love, my heart beats for you alone
```

### Image Generation Summary

- **Total Sections**: 7
- **Unique Images**: 5
- **Reused Images**: 2

### Unique Images to Generate

1. `bg-verse-1.png` - Saigon city streets scene
2. `bg-prechorus.png` - Moonlight on Perfume River
3. `bg-chorus.png` - Romantic embrace, home feeling
4. `bg-verse-2.png` - Nha Trang beach scene
5. `bg-bridge.png` - Dancing under stars

## Implementation

### Phase 7: Image Generation

```go
// Process each section
for _, section := range sections {
    var imageFile string
    var shouldGenerate bool
    
    switch section.Type {
    case "verse":
        imageFile = fmt.Sprintf("bg-verse-%d.png", section.Number)
        shouldGenerate = true // Always generate new for verses
        
    case "pre-chorus":
        imageFile = "bg-prechorus.png"
        shouldGenerate = !fileExists(imageFile) // Only if not exists
        
    case "chorus":
        imageFile = "bg-chorus.png"
        shouldGenerate = !fileExists(imageFile) // Only if not exists
        
    case "bridge":
        imageFile = "bg-bridge.png"
        shouldGenerate = true
        
    case "intro":
        imageFile = "bg-intro.png"
        shouldGenerate = true
        
    case "outro":
        imageFile = "bg-outro.png"
        shouldGenerate = true
    }
    
    if shouldGenerate {
        // Generate image using CQAI with section lyrics as prompt
        prompt := buildImagePrompt(section)
        imagePath := generateImage(prompt, imageFile)
    }
    
    // Store mapping: section -> imageFile
    sectionImageMap[section] = imageFile
}
```

### Phase 8: Video Rendering

When rendering video, use the `sectionImageMap` to know which background image to display during each section's timing.

```go
for _, section := range sections {
    imageFile := sectionImageMap[section]
    startTime := section.Lines[0].StartTime
    endTime := section.Lines[len(section.Lines)-1].EndTime
    
    // Add image to video timeline
    addBackgroundImage(imageFile, startTime, endTime)
}
```

## Benefits

1. **Consistency** - Choruses always have same visual theme
2. **Efficiency** - Only generate unique images needed
3. **Recognition** - Viewers associate chorus with specific visual
4. **Storage** - Fewer images to store and manage
5. **Cost** - Fewer CQAI API calls

## Test Results

✅ Parser detects all 7 sections correctly
✅ Identifies 5 unique images needed
✅ Correctly reuses chorus and pre-chorus images
✅ Verse numbering works properly
✅ JSON output is valid and complete
