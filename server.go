package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"text/template"

	_ "embed"

	"github.com/gorilla/websocket"
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
			messageBytes, err := json.Marshal(message)
			if err != nil {
				log.Fatal("Failed to encode as JSON:", message, err)
			}
			if err := connection.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
				log.Println("Failed to write message:", err)
				break
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
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend:     genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{APIVersion: "v1beta"},
	})
	model := "gemini-2.0-flash-live-001"
	if err != nil {
		log.Fatal("Failed to create a Gemini client:", err)
	}

	session, err := client.Live.Connect(ctx, model, &genai.LiveConnectConfig{})
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
