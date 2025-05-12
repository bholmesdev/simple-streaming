# Simple streaming example

This project shows a simple example of websockets to stream audio in real time.

## Technology Stack

*   **[Google Gemini Live:](https://ai.google.dev/gemini-api/docs/live)** Used to interpret audio with an LLM and respond like a real conversation.
*   **Golang:** A nice simple language for building servers. It's especially good at multi-threaded processing, which is useful for reading and writing messages over a web socket.

## Prerequisites

*   Ensure [Go installed on your system.](https://go.dev/doc/install)
*   [Create a Google API Key](https://ai.google.dev/gemini-api/docs) to access the Gemini API.

## Getting Started

1.  **Set up your Google API Key:**

    You need to set the `GOOGLE_API_KEY` environment variable. You can do this by adding the following line to your shell configuration file (e.g., `.bashrc`, `.zshrc`) or by setting it in your current terminal session:

    ```bash
    export GOOGLE_API_KEY="YOUR_API_KEY"
    ```

    Replace `"YOUR_API_KEY"` with your actual Google API Key.

2.  **Run the Development Server:**

    ```bash
    go run main.go
    ```

    The server will start on `http://localhost:8080`. Open in your browser for a simple debug view.

## License

This project is MIT licensed.

