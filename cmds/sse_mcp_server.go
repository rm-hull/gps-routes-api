package cmds

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/earthboundkid/versioninfo/v2"
	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewSseMcpServer(port int) {

	// Create MCP server
	s := server.NewMCPServer(
		"gps-routes",
		versioninfo.Short(),
		server.WithLogging(),
		server.WithToolCapabilities(true),
	)

	weatherTool := mcp.NewTool("get_weather",
		mcp.WithDescription("Get current weather information for a location"),
		mcp.WithString("location",
			mcp.Required(),
			mcp.Description("City name or zip"),
		),
	)

	gpsRoutesTool := mcp.NewTool("gps_routes",
		mcp.WithDescription("Searches for walking and cycling routes given "),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("")),
		mcp.WithString("nearby",
			mcp.Description("Limits the search to the specific area, used in conjunction with distanceKm")),
		mcp.WithString("distanceKm",
			mcp.Description("The search radius used when nearby searching"),
			mcp.DefaultNumber(50)),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			ReadOnlyHint:    true,
			DestructiveHint: false,
			IdempotentHint:  true,
			OpenWorldHint:   true,
		}),
	)

	s.AddTool(weatherTool, weatherHandler)
	s.AddTool(gpsRoutesTool, gpsRoutesHandler)

	log.Printf("Starting MCP SSE Server on port %d...", port)
	err := server.NewSSEServer(s).Start(fmt.Sprintf(":%d", port))
	log.Fatalf("MCP SSE Server failed to start on port %d: %v", port, err)
}

func McpMiddleware(sseServer *server.SSEServer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// c.Writer.Header().Set("content-type", "application/json; charset=utf-8")
		sseServer.ServeHTTP(c.Writer, c.Request)
	}
}

func weatherHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	location, ok := request.Params.Arguments["location"].(string)
	if !ok {
		return nil, errors.New("location must be a string")
	}

	return mcp.NewToolResultText(fmt.Sprintf("Current weather in %s:\nTemperature: 72°F\nConditions: Partly cloudy", location)), nil
}

func gpsRoutesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("The Knaresborough round is a challenging 20Km walk with some very picturesque settings"), nil
}
