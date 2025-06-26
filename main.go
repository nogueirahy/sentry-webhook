package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type SentryPayload struct {
	Project string `json:"project"`
	Message string `json:"message"`
	URL     string `json:"url"`
	Event   struct {
		Level string `json:"level"`
		Title string `json:"title"`
	} `json:"event"`
}

type GChatMessage struct {
	Text string `json:"text"`
}

func handleSentry(w http.ResponseWriter, r *http.Request) {
	var payload SentryPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "erro ao decodificar payload", http.StatusBadRequest)
		return
	}

	text := "🚨 Erro no Sentry!*\n" +
		"*Projeto:* " + payload.Project + "\n" +
		"*Mensagem:* " + payload.Event.Title + "\n" +
		"*URL:* " + payload.URL

	msg := GChatMessage{Text: text}
	body, _ := json.Marshal(msg)

	webhookURL := os.Getenv("GCHAT_WEBHOOK")
	if webhookURL == "" {
		http.Error(w, "Webhook do GChat não definido", http.StatusInternalServerError)
		return
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Println("Erro ao enviar para o GChat:", err)
		http.Error(w, "Erro ao enviar para o GChat", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("nenhuma .env encontrado. Variáveis devem estar no ambiente.")
	}

	http.HandleFunc("/sentry-webhook", handleSentry)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Servidor iniciado na porta", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
