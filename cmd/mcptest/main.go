package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
)

// This program tests MCP tool discovery without needing an LLM API key.
func main() {
	ctx := context.Background()

	// Create MCP stdio client
	mcpCli, err := client.NewStdioMCPClient(
		"python3", nil,
		"-m", "mcp_weather_server",
	)
	if err != nil {
		log.Fatalf("Failed to create MCP client: %v", err)
	}
	defer mcpCli.Close()

	// Initialize
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "mcp-test",
		Version: "1.0.0",
	}
	_, err = mcpCli.Initialize(ctx, initRequest)
	if err != nil {
		log.Fatalf("Failed to initialize MCP: %v", err)
	}

	// Get tools
	mcpTools, err := mcpp.GetTools(ctx, &mcpp.Config{Cli: mcpCli})
	if err != nil {
		log.Fatalf("Failed to get tools: %v", err)
	}

	fmt.Printf("Discovered %d MCP tools:\n\n", len(mcpTools))
	for i, t := range mcpTools {
		info, _ := t.Info(ctx)
		fmt.Printf("%d. %s\n   %s\n\n", i+1, info.Name, info.Desc)
	}

	// Test calling get_current_weather for Beijing
	fmt.Println("--- Testing get_current_weather for Beijing ---")
	for _, t := range mcpTools {
		info, _ := t.Info(ctx)
		if info.Name == "get_current_weather" {
			result, err := t.(tool.InvokableTool).InvokableRun(ctx, `{"city": "Beijing"}`)
			if err != nil {
				log.Fatalf("Tool call failed: %v", err)
			}
			fmt.Println("Result:", result)
			break
		}
	}
}
