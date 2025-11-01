package ai

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    anthropic "github.com/liushuangls/go-anthropic/v2"
    "github.com/rs/zerolog/log"
    "ugc-platform/internal/models"
)

// CharacterSelector handles AI-powered character selection
// ...existing code...
type CharacterSelector struct {
    client      *anthropic.Client
    maxTokens   int
    temperature float64
}

type CharacterSelection struct {
    CharacterType string                 `json:"character_type"`
    CharacterName string                 `json:"character_name"`
    Reasoning     string                 `json:"reasoning"`
    Confidence    float64                `json:"confidence"`
    Alternatives  []AlternativeCharacter `json:"alternatives"`
    ProductAnalysis ProductAnalysis      `json:"product_analysis"`
}

type AlternativeCharacter struct {
    CharacterType string  `json:"character_type"`
    CharacterName string  `json:"character_name"`
    Score         float64 `json:"score"`
    Reason        string  `json:"reason"`
}

type ProductAnalysis struct {
    Category      string   `json:"category"`
    TargetAge     string   `json:"target_age"`
    TargetGender  string   `json:"target_gender"`
    PriceRange    string   `json:"price_range"`
    Tone          string   `json:"tone"`
    KeyFeatures   []string `json:"key_features"`
    EmotionalTone string   `json:"emotional_tone"`
}

const (
    CharacterTypeYoungProfessional = "young_professional"
    CharacterTypeFamilyParent      = "family_parent"
    CharacterTypeFitnessEnthusiast = "fitness_enthusiast"
    CharacterTypeTechSavvy         = "tech_savvy"
    CharacterTypeLifestyleBlogger  = "lifestyle_blogger"
    CharacterTypeExpert            = "expert"
    CharacterTypeCelebrity         = "celebrity"
    CharacterTypeEverydayPerson    = "everyday_person"
)

func NewCharacterSelector(apiKey string, maxTokens int, temperature float64) *CharacterSelector {
    client := anthropic.NewClient(anthropic.WithAPIKey(apiKey))
    return &CharacterSelector{
        client:      client,
        maxTokens:   maxTokens,
        temperature: temperature,
    }
}

func (cs *CharacterSelector) SelectCharacter(ctx context.Context, productName, productDesc, imageURL string) (*CharacterSelection, error) {
    log.Info().Str("product_name", productName).Str("image_url", imageURL).Msg("Starting character selection")
    startTime := time.Now()
    prompt := cs.buildCharacterSelectionPrompt(productName, productDesc)
    messages := []anthropic.MessageParam{
        anthropic.NewUserMessage(
            anthropic.NewImageBlock(imageURL, anthropic.ImageBlockParamSourceTypeURL, "image/jpeg"),
            anthropic.NewTextBlock(prompt),
        ),
    }
    response, err := cs.client.Messages.New(ctx, anthropic.MessageNewParams{
        Model:       anthropic.F(anthropic.ModelClaude_3_5_Sonnet_20241022),
        MaxTokens:   anthropic.Int(cs.maxTokens),
        Temperature: anthropic.Float(cs.temperature),
        Messages:    anthropic.F(messages),
    })
    if err != nil {
        log.Error().Err(err).Msg("Failed to call Claude API")
        return nil, fmt.Errorf("failed to call Claude API: %w", err)
    }
    var textResponse string
    for _, block := range response.Content {
        if block.Type == anthropic.ContentBlockTypeText {
            textResponse += block.Text
        }
    }
    selection, err := cs.parseCharacterResponse(textResponse)
    if err != nil {
        log.Error().Err(err).Str("response", textResponse).Msg("Failed to parse character response")
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }
    duration := time.Since(startTime)
    log.Info().Str("character_type", selection.CharacterType).Str("character_name", selection.CharacterName).Float64("confidence", selection.Confidence).Dur("duration", duration).Msg("Character selection completed")
    return selection, nil
}

func (cs *CharacterSelector) buildCharacterSelectionPrompt(productName, productDesc string) string {
    return fmt.Sprintf(`You are an expert UGC (User Generated Content) ad strategist. Analyze the product image and description to select the most effective character/spokesperson for an advertisement video.

Product Name: %s
Product Description: %s

Your task is to:
1. Analyze the product category, target audience, and key features from the image and description
2. Select the most suitable character type for this product
3. Provide detailed reasoning for your selection
4. Suggest 2-3 alternative character options with scores

Available Character Types:
- young_professional: Ages 25-35, career-focused, urban lifestyle
- family_parent: Parents with children, family-oriented content
- fitness_enthusiast: Active lifestyle, health-conscious
- tech_savvy: Technology enthusiasts, early adopters
- lifestyle_blogger: Influencer-style, aspirational content
- expert: Industry expert or professional demonstrating expertise
- celebrity: Celebrity endorsement style
- everyday_person: Relatable, authentic everyday consumer

Response Format (JSON):
{
  "character_type": "string (one of the types above)",
  "character_name": "string (suggested name for the character)",
  "reasoning": "string (detailed explanation for this choice)",
  "confidence": float (0.0-1.0),
  "alternatives": [
    {
      "character_type": "string",
      "character_name": "string",
      "score": float (0.0-1.0),
      "reason": "string"
    }
  ],
  "product_analysis": {
    "category": "string",
    "target_age": "string",
    "target_gender": "string",
    "price_range": "string (budget/mid-range/premium/luxury)",
    "tone": "string (professional/casual/playful/serious)",
    "key_features": ["feature1", "feature2", "feature3"],
    "emotional_tone": "string (excitement/trust/comfort/aspiration)"
  }
}

Provide ONLY the JSON response, no additional text.`, productName, productDesc)
}

func (cs *CharacterSelector) parseCharacterResponse(response string) (*CharacterSelection, error) {
    response = strings.TrimSpace(response)
    response = strings.TrimPrefix(response, "```json")
    response = strings.TrimPrefix(response, "```")
    response = strings.TrimSuffix(response, "```")
    response = strings.TrimSpace(response)
    var selection CharacterSelection
    if err := json.Unmarshal([]byte(response), &selection); err != nil {
        return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
    }
    validTypes := []string{
        CharacterTypeYoungProfessional,
        CharacterTypeFamilyParent,
        CharacterTypeFitnessEnthusiast,
        CharacterTypeTechSavvy,
        CharacterTypeLifestyleBlogger,
        CharacterTypeExpert,
        CharacterTypeCelebrity,
        CharacterTypeEverydayPerson,
    }
    isValid := false
    for _, vt := range validTypes {
        if selection.CharacterType == vt {
            isValid = true
            break
        }
    }
    if !isValid {
        return nil, fmt.Errorf("invalid character type: %s", selection.CharacterType)
    }
    if selection.Confidence < 0 {
        selection.Confidence = 0
    } else if selection.Confidence > 1 {
        selection.Confidence = 1
    }
    return &selection, nil
}

func (cs *CharacterSelector) GenerateScript(ctx context.Context, selection *CharacterSelection, productName, productDesc string, duration int) (string, error) {
    log.Info().Str("character_type", selection.CharacterType).Int("duration", duration).Msg("Generating ad script")
    prompt := fmt.Sprintf(`You are a professional ad copywriter. Create a natural, engaging UGC-style ad script.

Product: %s
Description: %s
Character: %s (%s)
Target Duration: %d seconds
Tone: %s
Emotional Tone: %s

Requirements:
- Write in first person from the character's perspective
- Keep it conversational and authentic
- Include a hook in the first 3 seconds
- Highlight 2-3 key product benefits naturally
- Include a clear call-to-action at the end
- Match the character's personality and speaking style
- Script should fit within %d seconds when spoken

Format the script with:
[HOOK - 0-3s]: Opening line
[INTRODUCTION - 3-8s]: Character introduces themselves naturally
[PROBLEM/NEED - 8-15s]: Relatable problem or need
[SOLUTION - 15-25s]: How product solves it
[BENEFITS - 25-35s]: Key features and benefits
[SOCIAL PROOF - 35-40s]: Personal experience or results (if applicable)
[CALL TO ACTION - 40-%ds]: Clear next step

Provide ONLY the script content, no additional formatting or explanations.`, productName, productDesc, selection.CharacterName, selection.CharacterType, duration, selection.ProductAnalysis.Tone, selection.ProductAnalysis.EmotionalTone, duration, duration)
    messages := []anthropic.MessageParam{
        anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
    }
    response, err := cs.client.Messages.New(ctx, anthropic.MessageNewParams{
        Model:       anthropic.F(anthropic.ModelClaude_3_5_Sonnet_20241022),
        MaxTokens:   anthropic.Int(2048),
        Temperature: anthropic.Float(0.8),
        Messages:    anthropic.F(messages),
    })
    if err != nil {
        return "", fmt.Errorf("failed to generate script: %w", err)
    }
    var script string
    for _, block := range response.Content {
        if block.Type == anthropic.ContentBlockTypeText {
            script += block.Text
        }
    }
    log.Info().Int("script_length", len(script)).Msg("Script generated successfully")
    return strings.TrimSpace(script), nil
}

func (cs *CharacterSelector) AnalyzeProductImage(ctx context.Context, imageURL string) (*ImageAnalysis, error) {
    prompt := `Analyze this product image and provide detailed insights.

Describe:
1. Product type and category
2. Visual design and aesthetics
3. Perceived quality level
4. Target demographic indicators
5. Key visual elements and colors
6. Packaging or presentation style
7. Brand positioning indicators

Provide response in JSON format:
{
  "product_type": "string",
  "category": "string",
  "visual_quality": "string (budget/standard/premium/luxury)",
  "colors": ["color1", "color2"],
  "design_style": "string (modern/classic/minimalist/bold)",
  "target_demographic": "string",
  "key_elements": ["element1", "element2"],
  "brand_positioning": "string"
}`
    messages := []anthropic.MessageParam{
        anthropic.NewUserMessage(
            anthropic.NewImageBlock(imageURL, anthropic.ImageBlockParamSourceTypeURL, "image/jpeg"),
            anthropic.NewTextBlock(prompt),
        ),
    }
    response, err := cs.client.Messages.New(ctx, anthropic.MessageNewParams{
        Model:       anthropic.F(anthropic.ModelClaude_3_5_Sonnet_20241022),
        MaxTokens:   anthropic.Int(1024),
        Temperature: anthropic.Float(0.5),
        Messages:    anthropic.F(messages),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to analyze image: %w", err)
    }
    var textResponse string
    for _, block := range response.Content {
        if block.Type == anthropic.ContentBlockTypeText {
            textResponse += block.Text
        }
    }
    textResponse = strings.TrimSpace(textResponse)
    textResponse = strings.TrimPrefix(textResponse, "```json")
    textResponse = strings.TrimPrefix(textResponse, "```")
    textResponse = strings.TrimSuffix(textResponse, "```")
    textResponse = strings.TrimSpace(textResponse)
    var analysis ImageAnalysis
    if err := json.Unmarshal([]byte(textResponse), &analysis); err != nil {
        return nil, fmt.Errorf("failed to parse image analysis: %w", err)
    }
    return &analysis, nil
}

type ImageAnalysis struct {
    ProductType       string   `json:"product_type"`
    Category          string   `json:"category"`
    VisualQuality     string   `json:"visual_quality"`
    Colors            []string `json:"colors"`
    DesignStyle       string   `json:"design_style"`
    TargetDemographic string   `json:"target_demographic"`
    KeyElements       []string `json:"key_elements"`
    BrandPositioning  string   `json:"brand_positioning"`
}
