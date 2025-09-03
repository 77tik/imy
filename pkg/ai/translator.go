package ai

import (
	"fmt"
	"strings"
)

// TranslationResult 翻译结果
type TranslationResult struct {
	OriginalText string `json:"original_text"`
	Translated   string `json:"translated"`
	TargetLang   string `json:"target_lang"`
	SourceLang   string `json:"source_lang"`
}

// Translator 翻译器
type Translator struct {
	client *AIClient
	prompt *PromptBuilder
}

// NewTranslator 创建翻译器
func NewTranslator(client *AIClient) *Translator {
	return &Translator{
		client: client,
		prompt: NewDefaultPromptBuilder(),
	}
}

// Translate 翻译文本
func (t *Translator) Translate(text, targetLang string) (*TranslationResult, error) {
	if text == "" {
		return &TranslationResult{
			OriginalText: text,
			Translated:   "",
			TargetLang:   targetLang,
			SourceLang:   "unknown",
		}, nil
	}

	prompt, err := t.prompt.BuildPrompt("translate", map[string]interface{}{
		"text":        text,
		"target_lang": targetLang,
	})
	if err != nil {
		return nil, err
	}

	messages := []Message{
		{Role: "user", Content: prompt},
	}

	response, err := t.client.Chat(messages)
	if err != nil {
		return nil, err
	}

	return &TranslationResult{
		OriginalText: text,
		Translated:   strings.TrimSpace(response),
		TargetLang:   targetLang,
		SourceLang:   "auto",
	}, nil
}

// QuickTranslate 快速翻译（简化版）
func (t *Translator) QuickTranslate(text, targetLang string) string {
	result, err := t.Translate(text, targetLang)
	if err != nil {
		return text // 出错返回原文
	}
	return result.Translated
}

// TranslateWithContext 带上下文的翻译
func (t *Translator) TranslateWithContext(text, targetLang, context string) (*TranslationResult, error) {
	if text == "" {
		return &TranslationResult{
			OriginalText: text,
			Translated:   "",
			TargetLang:   targetLang,
			SourceLang:   "unknown",
		}, nil
	}

	prompt := fmt.Sprintf(`请根据上下文翻译以下内容：

原文：%s
目标语言：%s
上下文：%s

翻译要求：
1. 保持原意准确
2. 结合上下文语境
3. 语言自然流畅

翻译结果：`, text, targetLang, context)

	messages := []Message{
		{Role: "user", Content: prompt},
	}

	response, err := t.client.Chat(messages)
	if err != nil {
		return nil, err
	}

	return &TranslationResult{
		OriginalText: text,
		Translated:   strings.TrimSpace(response),
		TargetLang:   targetLang,
		SourceLang:   "auto",
	}, nil
}

// BatchTranslate 批量翻译
func (t *Translator) BatchTranslate(texts []string, targetLang string) ([]*TranslationResult, error) {
	results := make([]*TranslationResult, len(texts))
	for i, text := range texts {
		result, err := t.Translate(text, targetLang)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

// GetSupportedLanguages 获取支持的语言列表（简化版）
func (t *Translator) GetSupportedLanguages() []string {
	return []string{
		"中文", "英文", "日文", "韩文", "法文", "德文", "西班牙文", "俄文",
	}
}