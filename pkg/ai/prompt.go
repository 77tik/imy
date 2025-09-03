package ai

import (
	"bytes"
	"fmt"
	"text/template"
)

// PromptBuilder 简洁的prompt生成器
type PromptBuilder struct {
	templates map[string]string
}

// NewPromptBuilder 创建新的prompt生成器
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{
		templates: make(map[string]string),
	}
}

// AddTemplate 添加模板
func (pb *PromptBuilder) AddTemplate(name, template string) {
	pb.templates[name] = template
}

// BuildPrompt 构建提示词
func (pb *PromptBuilder) BuildPrompt(templateName string, data map[string]interface{}) (string, error) {
	tmpl, ok := pb.templates[templateName]
	if !ok {
		return "", fmt.Errorf("template %s not found", templateName)
	}

	t, err := template.New("prompt").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// BuildSimplePrompt 构建简单提示词
func (pb *PromptBuilder) BuildSimplePrompt(prompt string) string {
	return prompt
}

// 预定义模板
var defaultPrompts = map[string]string{
	"content_moderation": `请审核以下消息内容，判断是否存在违规内容：

消息内容：{{.content}}

请从以下维度进行判断：
1. 是否包含不当言论
2. 是否涉及敏感话题
3. 是否包含广告信息

回复格式：
违规类型：[无/不当言论/敏感话题/广告信息]
违规程度：[低/中/高]
建议操作：[通过/警告/拒绝]
说明：[简要说明]`,

	"chat_summary": `请对以下聊天记录进行总结：

聊天记录：
{{.chat_history}}

请提供：
1. 主要话题总结
2. 关键信息点
3. 用户情绪倾向
4. 建议后续行动

总结要求：简洁明了，不超过200字`,

	"translate": `请将以下内容翻译成{{.target_lang}}：

原文：{{.text}}

翻译要求：
1. 保持原意准确
2. 语言自然流畅
3. 符合{{.target_lang}}表达习惯

翻译结果：`,

	"code_explain": `请解释以下代码：

代码：
{{.code}}

编程语言：{{.language}}

请提供：
1. 代码功能说明
2. 关键逻辑解释
3. 可能的改进建议`,
}

// NewDefaultPromptBuilder 创建带默认模板的prompt生成器
func NewDefaultPromptBuilder() *PromptBuilder {
	pb := NewPromptBuilder()
	for name, template := range defaultPrompts {
		pb.AddTemplate(name, template)
	}
	return pb
}
