package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
)

func main() {
	ctx := context.Background()

	// 1. Create Ark ChatModel (火山方舟)
	apiKey := os.Getenv("ARK_API_KEY")
	model := os.Getenv("ARK_MODEL")
	if apiKey == "" {
		log.Fatal("Please set ARK_API_KEY environment variable")
	}
	if model == "" {
		log.Fatal("Please set ARK_MODEL environment variable (e.g. your endpoint ID)")
	}

	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey: apiKey,
		Model:  model,
	})
	if err != nil {
		log.Fatalf("Failed to create Ark ChatModel: %v", err)
	}

	// 2. Create MCP stdio client connecting to mcp_weather_server
	mcpCli, err := client.NewStdioMCPClient(
		"python3", nil,
		"-m", "mcp_weather_server",
	)
	if err != nil {
		log.Fatalf("Failed to create MCP stdio client: %v", err)
	}
	defer mcpCli.Close()

	// 3. Initialize MCP protocol
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "my-lobster-weather-agent",
		Version: "1.0.0",
	}
	_, err = mcpCli.Initialize(ctx, initRequest)
	if err != nil {
		log.Fatalf("Failed to initialize MCP: %v", err)
	}

	// 4. Get tools from MCP server
	mcpTools, err := mcpp.GetTools(ctx, &mcpp.Config{Cli: mcpCli})
	if err != nil {
		log.Fatalf("Failed to get MCP tools: %v", err)
	}

	fmt.Println("=== Weather Agent (Powered by Eino + Ark + MCP) ===")
	fmt.Println("Available MCP tools:")
	for _, t := range mcpTools {
		info, _ := t.Info(ctx)
		fmt.Printf("  - %s: %s\n", info.Name, info.Desc)
	}
	fmt.Println()

	// 5. Create ReAct agent
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: mcpTools,
		},
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			systemPrompt := schema.SystemMessage(
				"你是一个天气查询助手。用户会询问天气相关的问题，你需要使用工具来查询天气信息并回答。" +
					"请用中文回答用户的问题。如果用户提供了城市名，请使用对应的工具查询天气。")
			res := make([]*schema.Message, 0, len(input)+1)
			res = append(res, systemPrompt)
			res = append(res, input...)
			return res
		},
		MaxStep: 20,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// 6. Interactive loop
	scanner := bufio.NewScanner(os.Stdin)
	var history []*schema.Message

	fmt.Println("Enter your question (type 'quit' to exit):")
	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "quit" || input == "exit" {
			fmt.Println("Bye!")
			break
		}

		history = append(history, schema.UserMessage(input))

		// Stream the response
		stream, err := agent.Stream(ctx, history)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Print("\nAssistant: ")
		var fullContent strings.Builder
		for {
			msg, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				fmt.Printf("\nStream error: %v\n", err)
				break
			}
			fmt.Print(msg.Content)
			fullContent.WriteString(msg.Content)
		}
		stream.Close()
		fmt.Println()

		if fullContent.Len() > 0 {
			history = append(history, schema.AssistantMessage(fullContent.String(), nil))
		}
	}
}
