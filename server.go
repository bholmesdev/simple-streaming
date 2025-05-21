package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"

	_ "embed"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"google.golang.org/genai"
)

var addr = flag.String("attr", "localhost:8080", "http service address")
var upgrader = websocket.Upgrader{}

//go:embed index.html
var indexPageTemplate string

func live(w http.ResponseWriter, r *http.Request) {
	// Use "upgrade" to establish a websocket connection
	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal("Failed to upgrade to a websocket connection:", err)
		return
	}
	defer connection.Close()

	ctx := context.Background()
	session := createGeminiSession(ctx)
	defer session.Close()

	// Read messages (separate thread)
	go func() {
		for {
			message, err := session.Receive()
			if err != nil {
				log.Fatal("Failed to receive session response:", err)
			}
			// Example tool call handling
			if message.ToolCall != nil {
				responses := []*genai.FunctionResponse{}
				for _, functionCall := range message.ToolCall.FunctionCalls {
					functionResponse := genai.FunctionResponse{
						ID:       functionCall.ID,
						Name:     functionCall.Name,
						Response: map[string]any{"ok": "true"},
					}
					responses = append(responses, &functionResponse)
				}
				session.SendToolResponse(genai.LiveToolResponseInput{
					FunctionResponses: responses,
				})
			} else if message.ServerContent != nil {
				messageBytes, err := json.Marshal(message)
				if err != nil {
					log.Fatal("Failed to encode as JSON:", message, err)
				}
				if err := connection.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
					log.Println("Failed to write message:", err)
					break
				}
			}
		}
	}()

	// Send messages
	for {
		_, message, err := connection.ReadMessage()
		if err != nil {
			log.Println("Failed to read message from client:", err)
			break
		}

		var realtimeInput genai.LiveRealtimeInput
		if err := json.Unmarshal(message, &realtimeInput); err != nil {
			log.Fatal("Failed to decode message as JSON:", string(message), err)
		}
		session.SendRealtimeInput(realtimeInput)
	}
}

func indexPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("index").Parse(indexPageTemplate)
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	wsUrl := "ws://" + r.Host + "/live"
	fmt.Println(wsUrl)
	if err := tmpl.Execute(w, wsUrl); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

func createGeminiSession(ctx context.Context) *genai.Session {
	err := godotenv.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading .env file: %v\n", err)
		os.Exit(1)
	}

	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "GOOGLE_API_KEY is not set\n")
		os.Exit(1)
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:      apiKey,
		Backend:     genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{APIVersion: "v1beta"},
	})
	model := "gemini-2.0-flash-live-001"
	if err != nil {
		log.Fatal("Failed to create a Gemini client:", err)
	}

	session, err := client.Live.Connect(ctx, model, &genai.LiveConnectConfig{
		RealtimeInputConfig: &genai.RealtimeInputConfig{
			AutomaticActivityDetection: &genai.AutomaticActivityDetection{
				EndOfSpeechSensitivity: genai.EndSensitivityLow,
				SilenceDurationMs:      ptr(int32(200)),
			},
		},
		Tools: []*genai.Tool{
			{
				// Example tool call declaration
				FunctionDeclarations: []*genai.FunctionDeclaration{
					{
						Name: "turn_on_the_lights",
					},
				},
			},
		},
	})
	if err != nil {
		log.Fatal("Failed to connect to Gemini Flash live model:", err)
	}

	return session
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/", indexPage)
	http.HandleFunc("/live", live)

	const port = "8080"
	log.Printf("Running on http://localhost:%s", port)
	// Log fatal error if the HTTP server fails to start (e.g., port already in use).
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func ptr[T any](v T) *T {
	return &v
}
