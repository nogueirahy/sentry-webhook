package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type SentryPayload struct {
	Project     string    `json:"project"`
	Message     string    `json:"message"`
	URL         string    `json:"url"`
	Environment string    `json:"environment,omitempty"`
	Timestamp   time.Time `json:"timestamp,omitempty"`
	Culprit     string    `json:"culprit,omitempty"`
	Event       Event     `json:"event"`
	User        User      `json:"user,omitempty"`
	Tags        []Tag     `json:"tags,omitempty"`
}

type Event struct {
	EventID     string            `json:"event_id,omitempty"`
	Level       string            `json:"level"`
	Title       string            `json:"title"`
	Fingerprint []string          `json:"fingerprint,omitempty"`
	Platform    string            `json:"platform,omitempty"`
	Extra       map[string]string `json:"extra,omitempty"`
}

type User struct {
	ID       string `json:"id,omitempty"`
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
}

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type GChatMessage struct {
	Text  string `json:"text,omitempty"`
	Cards []Card `json:"cards,omitempty"`
}

type Card struct {
	Header   CardHeader `json:"header,omitempty"`
	Sections []Section  `json:"sections,omitempty"`
}

type CardHeader struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
}

type Section struct {
	Widgets []Widget `json:"widgets"`
}

type Widget struct {
	KeyValue *KeyValue `json:"keyValue,omitempty"`
	Buttons  []Button  `json:"buttons,omitempty"`
}

type KeyValue struct {
	TopLabel         string `json:"topLabel"`
	Content          string `json:"content"`
	ContentMultiline bool   `json:"contentMultiline,omitempty"`
	BottomLabel      string `json:"bottomLabel,omitempty"`
	Icon             string `json:"icon,omitempty"`
}

type Button struct {
	TextButton TextButton `json:"textButton"`
}

type TextButton struct {
	Text    string  `json:"text"`
	OnClick OnClick `json:"onClick"`
}

type OnClick struct {
	OpenLink OpenLink `json:"openLink"`
}

type OpenLink struct {
	URL string `json:"url"`
}

func (p *SentryPayload) getLevelEmoji() string {
	switch strings.ToLower(p.Event.Level) {
	case "fatal", "error":
		return "🚨"
	case "warning":
		return "⚠️"
	case "info":
		return "ℹ️"
	case "debug":
		return "🐛"
	default:
		return "📋"
	}
}

func (p *SentryPayload) formatSimpleMessage() string {
	emoji := p.getLevelEmoji()
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("%s *Alerta Sentry* - *%s* %s\n\n", emoji, p.Project, emoji))
	builder.WriteString(fmt.Sprintf("⚠️ %s\n", p.Event.Title))
	builder.WriteString(fmt.Sprintf("*Nível:* %s\n", strings.ToUpper(p.Event.Level)))

	if p.Environment != "" {
		builder.WriteString(fmt.Sprintf("*Ambiente:* %s\n", p.Environment))
	}

	if p.Culprit != "" {
		builder.WriteString(fmt.Sprintf("*Origem:* %s\n", p.Culprit))
	}

	if p.User.Email != "" || p.User.Username != "" {
		builder.WriteString(fmt.Sprintf("*Usuário:* %s\n", p.getUserInfo()))
	}

	builder.WriteString(fmt.Sprintf("\n%s", p.URL))

	return builder.String()
}

func (p *SentryPayload) getUserInfo() string {
	if p.User.Email != "" && p.User.Username != "" {
		return fmt.Sprintf("%s (%s)", p.User.Username, p.User.Email)
	}
	if p.User.Email != "" {
		return p.User.Email
	}
	if p.User.Username != "" {
		return p.User.Username
	}
	if p.User.ID != "" {
		return fmt.Sprintf("ID: %s", p.User.ID)
	}
	return "N/A"
}

func (p *SentryPayload) createCardMessage() GChatMessage {
	emoji := p.getLevelEmoji()

	widgets := []Widget{
		{
			KeyValue: &KeyValue{
				TopLabel: "Projeto",
				Content:  p.Project,
				Icon:     "BOOKMARK",
			},
		},
		{
			KeyValue: &KeyValue{
				TopLabel: "Nível",
				Content:  strings.ToUpper(p.Event.Level),
				Icon:     "ERROR",
			},
		},
	}

	if p.Environment != "" {
		widgets = append(widgets, Widget{
			KeyValue: &KeyValue{
				TopLabel: "Ambiente",
				Content:  p.Environment,
				Icon:     "STAR",
			},
		})
	}

	if p.Culprit != "" {
		widgets = append(widgets, Widget{
			KeyValue: &KeyValue{
				TopLabel: "Origem",
				Content:  p.Culprit,
				Icon:     "DESCRIPTION",
			},
		})
	}

	if p.User.Email != "" || p.User.Username != "" {
		widgets = append(widgets, Widget{
			KeyValue: &KeyValue{
				TopLabel: "Usuário",
				Content:  p.getUserInfo(),
				Icon:     "PERSON",
			},
		})
	}

	if !p.Timestamp.IsZero() {
		widgets = append(widgets, Widget{
			KeyValue: &KeyValue{
				TopLabel: "Timestamp",
				Content:  p.Timestamp.Format("02/01/2006 15:04:05"),
				Icon:     "CLOCK",
			},
		})
	}

	buttonWidget := Widget{
		Buttons: []Button{
			{
				TextButton: TextButton{
					Text: "Ver no Sentry",
					OnClick: OnClick{
						OpenLink: OpenLink{
							URL: p.URL,
						},
					},
				},
			},
		},
	}

	widgets = append(widgets, buttonWidget)

	return GChatMessage{
		Cards: []Card{
			{
				Header: CardHeader{
					Title:    fmt.Sprintf("%s Alerta Sentry", emoji),
					Subtitle: p.Event.Title,
				},
				Sections: []Section{
					{
						Widgets: widgets,
					},
				},
			},
		},
	}
}

func sendToGChat(message interface{}) error {
	webhookURL := os.Getenv("GCHAT_WEBHOOK")
	if webhookURL == "" {
		return fmt.Errorf("webhook do GChat não definido")
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("erro ao serializar mensagem: %v", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("erro ao enviar para o GChat: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GChat retornou status %d", resp.StatusCode)
	}

	return nil
}

func handleSentry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var payload SentryPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("Erro ao decodificar payload: %v", err)
		http.Error(w, "erro ao decodificar payload", http.StatusBadRequest)
		return
	}

	log.Printf("Webhook recebido - Projeto: %s, Nível: %s, Título: %s",
		payload.Project, payload.Event.Level, payload.Event.Title)

	useCards := os.Getenv("USE_CARDS") == "true"

	var message interface{}
	if useCards {
		message = payload.createCardMessage()
	} else {
		message = GChatMessage{
			Text: payload.formatSimpleMessage(),
		}
	}

	if err := sendToGChat(message); err != nil {
		log.Printf("Erro ao enviar para GChat: %v", err)
		http.Error(w, "erro ao enviar para o GChat", http.StatusInternalServerError)
		return
	}

	log.Println("Mensagem enviada com sucesso para o Google Chat")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Nenhum arquivo .env encontrado. Usando variáveis do ambiente.")
	}

	http.HandleFunc("/sentry-webhook", handleSentry)
	http.HandleFunc("/health", healthCheck)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Sentry to Google Chat Webhook - Funcionando!"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	if os.Getenv("GCHAT_WEBHOOK") == "" {
		log.Println("⚠️  AVISO: GCHAT_WEBHOOK não está definido")
	}

	log.Printf("🚀 Servidor iniciado na porta %s", port)
	log.Printf("📋 Endpoints disponíveis:")
	log.Printf("   POST /sentry-webhook - Recebe webhooks do Sentry")
	log.Printf("   GET  /health         - Health check")
	log.Printf("   GET  /               - Status do serviço")

	if os.Getenv("USE_CARDS") == "true" {
		log.Printf("🎨 Formato: Cards do Google Chat")
	} else {
		log.Printf("💬 Formato: Mensagem de texto simples")
	}

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
