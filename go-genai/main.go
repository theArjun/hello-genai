package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// getLLMEndpoint returns the complete LLM API endpoint URL
func getLLMEndpoint() string {
	baseURL := os.Getenv("LLM_BASE_URL")
	return baseURL + "/chat/completions"
}

// ChatRequest represents the structure of a chat request to the LLM API
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

// ChatMessage represents a message in the chat
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse represents the response from the LLM API
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
}

// getModelName returns the model name to use for API requests
func getModelName() string {
	modelName := os.Getenv("LLM_MODEL_NAME")
	return modelName
}

func main() {
	// Configure server
	port := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	// Create a new server mux
	mux := http.NewServeMux()

	// Handler for the root path - chat interface
	mux.HandleFunc("/", handleChatInterface)

	// Handler for chat API
	mux.HandleFunc("/api/chat", handleChatAPI)

	// Start the server
	serverAddr := ":" + port
	fmt.Printf("Server starting on http://localhost%s\n", serverAddr)
	fmt.Printf("Using LLM endpoint: %s\n", getLLMEndpoint())
	fmt.Printf("Using model: %s\n", getModelName())
	log.Fatal(http.ListenAndServe(serverAddr, mux))
}

// handleChatInterface serves the chat web interface
func handleChatInterface(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Generate HTML response
	html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hello-GenAI in Go</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://cdn.jsdelivr.net/npm/marked/marked.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/highlight.js@11.7.0/lib/core.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/highlight.js@11.7.0/lib/languages/go.min.js"></script>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/highlight.js@11.7.0/styles/github.min.css">
    <style>
        .prose pre {
            padding: 1rem;
            border-radius: 0.5rem;
            overflow-x: auto;
        }
        .prose code {
            background-color: #f3f4f6;
            padding: 0.2rem 0.4rem;
            border-radius: 0.25rem;
            font-size: 0.875em;
        }
        .prose pre code {
            background-color: transparent;
            padding: 0;
        }
        .message-content {
            max-width: 100%;
            overflow-x: auto;
        }
        .loading {
            margin: 0.5rem 0;
            padding: 0.5rem;
            color: #6366f1;
            font-style: italic;
        }
    </style>
</head>
<body class="bg-gray-50 min-h-screen">
    <div class="max-w-4xl mx-auto px-4 py-8">
        <h1 class="text-3xl font-bold text-center text-indigo-600 mb-8">Hello-GenAI in Go</h1>
        
        <div class="bg-white rounded-lg shadow-lg overflow-hidden">
            <div id="chat-box" class="h-[70vh] overflow-y-auto p-6 space-y-4">
                <div class="flex items-start space-x-3">
                    <div class="flex-shrink-0">
                        <div class="w-8 h-8 rounded-full bg-indigo-100 flex items-center justify-center">
                            <svg class="w-5 h-5 text-indigo-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                            </svg>
                        </div>
                    </div>
                    <div class="flex-1 bg-indigo-50 rounded-lg p-4 prose max-w-none">
                        <div class="message-content">
                            Hello! I'm your GenAI assistant. How can I help you today?
                        </div>
                    </div>
                </div>
            </div>

            <div class="border-t border-gray-200 p-4">
                <div class="flex space-x-4">
                    <input type="text" id="message-input" 
                           class="flex-1 rounded-lg border border-gray-300 px-4 py-2 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                           placeholder="Type your message here..." autofocus>
                    <button id="send-button" 
                            class="bg-indigo-600 text-white px-6 py-2 rounded-lg hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 transition-colors">
                        Send
                    </button>
                </div>
            </div>
        </div>

        <div class="mt-4 text-center text-sm text-gray-500">
            &copy; 2025 hello-genai | Powered by <span id="model-name" class="font-medium">Loading model info...</span>
        </div>
    </div>

    <script>
        // Configure marked
        marked.setOptions({
            highlight: function(code, lang) {
                if (lang && hljs.getLanguage(lang)) {
                    return hljs.highlight(code, { language: lang }).value;
                }
                return code;
            },
            breaks: true
        });

        document.addEventListener('DOMContentLoaded', function() {
            const chatBox = document.getElementById('chat-box');
            const messageInput = document.getElementById('message-input');
            const sendButton = document.getElementById('send-button');
            const modelNameSpan = document.getElementById('model-name');

            // Get model info
            fetch('/api/chat', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ message: "!modelinfo" }),
            })
            .then(response => response.json())
            .then(data => {
                if (data.model) {
                    modelNameSpan.textContent = data.model;
                } else {
                    modelNameSpan.textContent = "AI Language Model";
                }
            })
            .catch(error => {
                modelNameSpan.textContent = "AI Language Model";
            });

            function sendMessage() {
                const message = messageInput.value.trim();
                if (!message) return;

                // Add user message to chat
                addMessageToChat('user', message);
                messageInput.value = '';

                // Show loading indicator
                const loadingDiv = document.createElement('div');
                loadingDiv.className = 'loading';
                loadingDiv.textContent = 'Thinking...';
                chatBox.appendChild(loadingDiv);
                chatBox.scrollTop = chatBox.scrollHeight;

                // Send message to API
                fetch('/api/chat', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ message: message }),
                })
                .then(response => response.json())
                .then(data => {
                    // Remove loading indicator
                    chatBox.removeChild(loadingDiv);
                    
                    // Add bot's response to chat
                    addMessageToChat('bot', data.response);
                })
                .catch(error => {
                    // Remove loading indicator
                    chatBox.removeChild(loadingDiv);
                    
                    // Show error message
                    addMessageToChat('bot', 'Sorry, I encountered an error. Please try again.');
                    console.error('Error:', error);
                });
            }

            function addMessageToChat(role, content) {
                const messageDiv = document.createElement('div');
                messageDiv.className = 'flex items-start space-x-3';
                
                const iconDiv = document.createElement('div');
                iconDiv.className = 'flex-shrink-0';
                iconDiv.innerHTML = role === 'user' 
                    ? '<div class="w-8 h-8 rounded-full bg-gray-100 flex items-center justify-center"><svg class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" /></svg></div>'
                    : '<div class="w-8 h-8 rounded-full bg-indigo-100 flex items-center justify-center"><svg class="w-5 h-5 text-indigo-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" /></svg></div>';
                
                const contentDiv = document.createElement('div');
                contentDiv.className = role === 'user' 
                    ? 'flex-1 bg-gray-50 rounded-lg p-4 prose max-w-none'
                    : 'flex-1 bg-indigo-50 rounded-lg p-4 prose max-w-none';
                
                const messageContent = document.createElement('div');
                messageContent.className = 'message-content';
                messageContent.innerHTML = marked.parse(content);
                
                contentDiv.appendChild(messageContent);
                messageDiv.appendChild(iconDiv);
                messageDiv.appendChild(contentDiv);
                
                chatBox.appendChild(messageDiv);
                chatBox.scrollTop = chatBox.scrollHeight;
            }

            // Event listeners
            sendButton.addEventListener('click', sendMessage);
            messageInput.addEventListener('keypress', function(e) {
                if (e.key === 'Enter') {
                    sendMessage();
                }
            });
        });
    </script>
</body>
</html>
`

	// Set content type and write response
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

// handleChatAPI processes chat API requests
func handleChatAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body
	var requestBody struct {
		Message string `json:"message"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Special command for getting model info
	if requestBody.Message == "!modelinfo" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"model": getModelName(),
		})
		return
	}

	// Call the LLM API
	response, err := callLLMAPI(requestBody.Message)
	if err != nil {
		log.Printf("Error calling LLM API: %v", err)
		http.Error(w, "Failed to get response from LLM", http.StatusInternalServerError)
		return
	}

	// Return the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"response": response,
	})
}

// callLLMAPI calls the LLM API and returns the response
func callLLMAPI(userMessage string) (string, error) {
	// Prepare the request body
	chatRequest := ChatRequest{
		Model: getModelName(),
		Messages: []ChatMessage{
			{
				Role:    "system",
				Content: "You are a helpful assistant.",
			},
			{
				Role:    "user",
				Content: userMessage,
			},
		},
	}

	requestBody, err := json.Marshal(chatRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", getLLMEndpoint(), bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Set a timeout for the HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check if the status code is not 200 OK
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status code %d: %s", resp.StatusCode, respBody)
	}

	// Parse the response
	var chatResponse ChatResponse
	err = json.Unmarshal(respBody, &chatResponse)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Extract the assistant's message
	if len(chatResponse.Choices) > 0 {
		return strings.TrimSpace(chatResponse.Choices[0].Message.Content), nil
	}

	return "", fmt.Errorf("no response choices returned from API")
}
