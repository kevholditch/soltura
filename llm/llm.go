package llm

import "context"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Completer interface {
	StreamCompletion(ctx context.Context, system string, messages []Message, onChunk func(string)) (string, error)
	Complete(ctx context.Context, system string, messages []Message) (string, error)
}
