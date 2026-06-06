# veniceai-go

Go client for the [Venice.ai API](https://docs.veniceai.ai). Composes the official [openai-go](https://github.com/openai/openai-go) client for OpenAI-compatible endpoints with a generated client covering all Venice-specific endpoints.

## Install

```bash
go get github.com/13rac1/veniceai-go
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/13rac1/veniceai-go"
	"github.com/13rac1/veniceai-go/venicegen"
	openai "github.com/openai/openai-go/v3"
)

func main() {
	client, err := veniceai.NewClient("your-api-key")
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	// OpenAI-compatible endpoints (chat, embeddings, audio, images, models)
	chat, err := client.OpenAI.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: "llama-3.3-70b",
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Hello from Venice!"),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(chat.Choices[0].Message.Content)

	// Venice-specific endpoints use generated types from the venicegen package
	resp, err := client.API.GenerateImageWithResponse(ctx, nil, venicegen.GenerateImageJSONRequestBody{
		Model:  "fluently-xl",
		Prompt: "a cat in space",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.StatusCode())
}
```

### Options

```go
// Custom base URL
client, err := veniceai.NewClient("key", veniceai.WithBaseURL("https://custom.endpoint/api/v1"))

// Custom HTTP client
client, err := veniceai.NewClient("key", veniceai.WithHTTPClient(myHTTPClient))
```

## Updating the generated client

When the Venice API spec changes:

```bash
cd veniceai-api-docs && git pull && cd ..
make check  # regenerate, lint, test
```
