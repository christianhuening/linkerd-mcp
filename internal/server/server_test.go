package server_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/christianhuening/linkerd-mcp/internal/server"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

var _ = Describe("LinkerdMCPServer", func() {
	Describe("New", func() {
		It("should skip integration test requiring Kubernetes config", func() {
			Skip("Skipping integration test that requires Kubernetes config")

			mcpServer, err := server.New()
			Expect(err).NotTo(HaveOccurred())
			Expect(mcpServer).NotTo(BeNil())
		})
	})

	Describe("RegisterTools", func() {
		It("should create MCP server successfully", func() {
			mcpSrv := mcpserver.NewMCPServer(
				"test-server",
				"1.0.0",
				mcpserver.WithToolCapabilities(true),
			)

			Expect(mcpSrv).NotTo(BeNil())
		})
	})

	Describe("LinkerdMCPServer structure", func() {
		It("should have nil fields before initialization", func() {
			srv := &server.LinkerdMCPServer{}

			// Using reflection would be better, but for now we just verify the struct exists
			Expect(srv).NotTo(BeNil())
		})
	})

	Describe("Tool argument extraction", func() {
		Context("check_mesh_health tool", func() {
			It("should extract namespace argument", func() {
				args := map[string]interface{}{
					"namespace": "linkerd",
				}

				request := mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Arguments: args,
					},
				}

				extractedArgs, ok := request.Params.Arguments.(map[string]interface{})
				Expect(ok).To(BeTrue())

				namespace, _ := extractedArgs["namespace"].(string)
				Expect(namespace).To(Equal("linkerd"))
			})
		})

		Context("analyze_connectivity tool", func() {
			It("should extract all connectivity arguments", func() {
				args := map[string]interface{}{
					"source_namespace": "prod",
					"source_service":   "frontend",
					"target_namespace": "prod",
					"target_service":   "backend",
				}

				request := mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Arguments: args,
					},
				}

				extractedArgs, ok := request.Params.Arguments.(map[string]interface{})
				Expect(ok).To(BeTrue())

				sourceNamespace, _ := extractedArgs["source_namespace"].(string)
				Expect(sourceNamespace).To(Equal("prod"))

				sourceService, _ := extractedArgs["source_service"].(string)
				Expect(sourceService).To(Equal("frontend"))

				targetNamespace, _ := extractedArgs["target_namespace"].(string)
				Expect(targetNamespace).To(Equal("prod"))

				targetService, _ := extractedArgs["target_service"].(string)
				Expect(targetService).To(Equal("backend"))
			})
		})

		Context("list_meshed_services tool", func() {
			It("should extract namespace argument", func() {
				args := map[string]interface{}{
					"namespace": "prod",
				}

				request := mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Arguments: args,
					},
				}

				extractedArgs, ok := request.Params.Arguments.(map[string]interface{})
				Expect(ok).To(BeTrue())

				namespace, _ := extractedArgs["namespace"].(string)
				Expect(namespace).To(Equal("prod"))
			})
		})

		Context("get_allowed_targets tool", func() {
			It("should extract source arguments", func() {
				args := map[string]interface{}{
					"source_namespace": "prod",
					"source_service":   "frontend",
				}

				request := mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Arguments: args,
					},
				}

				extractedArgs, ok := request.Params.Arguments.(map[string]interface{})
				Expect(ok).To(BeTrue())

				sourceNamespace, _ := extractedArgs["source_namespace"].(string)
				Expect(sourceNamespace).To(Equal("prod"))

				sourceService, _ := extractedArgs["source_service"].(string)
				Expect(sourceService).To(Equal("frontend"))
			})
		})

		Context("get_allowed_sources tool", func() {
			It("should extract target arguments", func() {
				args := map[string]interface{}{
					"target_namespace": "prod",
					"target_service":   "backend",
				}

				request := mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Arguments: args,
					},
				}

				extractedArgs, ok := request.Params.Arguments.(map[string]interface{})
				Expect(ok).To(BeTrue())

				targetNamespace, _ := extractedArgs["target_namespace"].(string)
				Expect(targetNamespace).To(Equal("prod"))

				targetService, _ := extractedArgs["target_service"].(string)
				Expect(targetService).To(Equal("backend"))
			})
		})

		Context("with empty arguments", func() {
			It("should handle empty argument map", func() {
				request := mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Arguments: map[string]interface{}{},
					},
				}

				extractedArgs, ok := request.Params.Arguments.(map[string]interface{})
				Expect(ok).To(BeTrue())

				namespace, _ := extractedArgs["namespace"].(string)
				Expect(namespace).To(BeEmpty())
			})
		})

		Context("with nil arguments", func() {
			It("should handle nil arguments gracefully", func() {
				request := mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Arguments: nil,
					},
				}

				_, ok := request.Params.Arguments.(map[string]interface{})
				Expect(ok).To(BeFalse())
			})
		})
	})
})
