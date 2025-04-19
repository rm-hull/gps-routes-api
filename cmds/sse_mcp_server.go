package cmds

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

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
		"ref-data://gps-routes",
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

	baseUrl := "http://localhost:8080/v1/gps-routes" // TODO: use env var
	req, err := http.NewRequestWithContext(ctx, "GET", baseUrl+"/ref-data", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)

	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", "MCP-GPS-Routes-API")
	req.Header.Add("Accept-Language", "en-GB,en;q=0.9,en-US;q=0.8")

	log.Printf("Making request to %s", req.URL.String())
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return []mcp.ResourceContents{mcp.TextResourceContents{
		URI:      request.Params.URI,
		MIMEType: "application/json",
		Text:     string(body),
	}}, nil
}
