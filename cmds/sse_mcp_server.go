package cmds

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/earthboundkid/versioninfo/v2"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewSseMcpServer(port int) {

	// Create MCP server
	s := server.NewMCPServer(
		"gps-routes",
		versioninfo.Short(),
		server.WithLogging(),
		server.WithRecovery(),
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
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

	refDataResources := mcp.NewResource(
		"/ref-data",
		"Reference Data",
		mcp.WithMIMEType("application/json"),
		mcp.WithResourceDescription(
			"Reference data for the GPS Routes API, detailing all the facets and their possbile values. Note that the key represents the valid facet values, while the numeric values are the counts of how many instances there are. You should infer higher numbers mean more popular facets."),
	)

	s.AddResource(refDataResources, refDataHandler)
	s.AddTool(gpsRoutesTool, gpsRoutesHandler)

	log.Printf("Starting MCP SSE Server on port %d...", port)
	err := server.NewSSEServer(s).Start(fmt.Sprintf(":%d", port))
	log.Fatalf("MCP SSE Server failed to start on port %d: %v", port, err)
}

// func McpMiddleware(sseServer *server.SSEServer) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		// c.Writer.Header().Set("content-type", "application/json; charset=utf-8")
// 		sseServer.ServeHTTP(c.Writer, c.Request)
// 	}
// }

func gpsRoutesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("The Knaresborough round is a challenging 20Km walk with some very picturesque settings"), nil
}

func refDataHandler(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Simulate fetching reference data
	refData := map[string]any{
		"facets": map[string]any{
			"difficulty": map[string]int64{
				"easy":   100,
				"medium": 200,
				"hard":   50,
			},
			"length": map[string]int64{
				"short":  150,
				"medium": 300,
				"long":   50,
			},
		},
	}

	data, err := json.Marshal(refData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal reference data: %w", err)
	}

	return []mcp.ResourceContents{mcp.TextResourceContents{
		URI:      "ref-data",
		MIMEType: "application/json",
		Text:     string(data),
	}}, nil
}
