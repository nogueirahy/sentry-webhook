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

type SentryWebhook struct {
	Action       string       `json:"action"`
	Installation Installation `json:"installation"`
	Data         WebhookData  `json:"data"`
	Actor        Actor        `json:"actor"`
}

type Installation struct {
	UUID string `json:"uuid"`
}

type WebhookData struct {
	Issue Issue `json:"issue"`
}

type Actor struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Estrutura da issue do Sentry
type Issue struct {
	URL                 string         `json:"url"`
	WebURL              string         `json:"web_url"`
	ProjectURL          string         `json:"project_url"`
	ID                  string         `json:"id"`
	ShareID             *string        `json:"shareId"`
	ShortID             string         `json:"shortId"`
	Title               string         `json:"title"`
	Culprit             string         `json:"culprit"`
	Permalink           *string        `json:"permalink"`
	Logger              *string        `json:"logger"`
	Level               string         `json:"level"`
	Status              string         `json:"status"`
	StatusDetails       map[string]any `json:"statusDetails"`
	Substatus           string         `json:"substatus"`
	IsPublic            bool           `json:"isPublic"`
	Platform            string         `json:"platform"`
	Project             Project        `json:"project"`
	Type                string         `json:"type"`
	Metadata            Metadata       `json:"metadata"`
	NumComments         int            `json:"numComments"`
	AssignedTo          *string        `json:"assignedTo"`
	IsBookmarked        bool           `json:"isBookmarked"`
	IsSubscribed        bool           `json:"isSubscribed"`
	SubscriptionDetails *string        `json:"subscriptionDetails"`
	HasSeen             bool           `json:"hasSeen"`
	Annotations         []any          `json:"annotations"`
	IssueType           string         `json:"issueType"`
	IssueCategory       string         `json:"issueCategory"`
	Priority            string         `json:"priority"`
	PriorityLockedAt    *string        `json:"priorityLockedAt"`
	IsUnhandled         bool           `json:"isUnhandled"`
	Count               string         `json:"count"`
	UserCount           int            `json:"userCount"`
	FirstSeen           time.Time      `json:"firstSeen"`
	LastSeen            time.Time      `json:"lastSeen"`
}

type Project struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Platform string `json:"platform"`
}

type Metadata struct {
	Value           string `json:"value"`
	Type            string `json:"type"`
	Filename        string `json:"filename"`
	Function        string `json:"function"`
	InAppFrameMix   string `json:"in_app_frame_mix"`
	SDK             SDK    `json:"sdk"`
	InitialPriority int    `json:"initial_priority"`
}

type SDK struct {
	Name           string `json:"name"`
	NameNormalized string `json:"name_normalized"`
}

// Estruturas para Google Chat (mantidas iguais)
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

func (w *SentryWebhook) getLevelEmoji() string {
	switch strings.ToLower(w.Data.Issue.Level) {
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

func (w *SentryWebhook) getPriorityEmoji() string {
	switch strings.ToLower(w.Data.Issue.Priority) {
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "⚪"
	}
}

func (w *SentryWebhook) formatSimpleMessage() string {
	emoji := w.getLevelEmoji()
	priorityEmoji := w.getPriorityEmoji()
	issue := w.Data.Issue
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("%s *Alerta Sentry* - *%s* %s\n\n", emoji, issue.Project.Name, emoji))
	builder.WriteString(fmt.Sprintf("🆔 *Issue:* %s\n", issue.ShortID))
	builder.WriteString(fmt.Sprintf("⚠️ *Título:* %s\n", issue.Title))
	builder.WriteString(fmt.Sprintf("*Nível:* %s\n", strings.ToUpper(issue.Level)))
	builder.WriteString(fmt.Sprintf("*Prioridade:* %s %s\n", priorityEmoji, strings.ToUpper(issue.Priority)))
	builder.WriteString(fmt.Sprintf("*Plataforma:* %s\n", issue.Platform))

	if issue.Culprit != "" {
		builder.WriteString(fmt.Sprintf("*Origem:* %s\n", issue.Culprit))
	}

	if issue.Count != "" {
		builder.WriteString(fmt.Sprintf("*Ocorrências:* %s\n", issue.Count))
	}

	builder.WriteString(fmt.Sprintf("*Status:* %s (%s)\n", issue.Status, issue.Substatus))

	if !issue.FirstSeen.IsZero() {
		builder.WriteString(fmt.Sprintf("*Primeira ocorrência:* %s\n", issue.FirstSeen.Format("02/01/2006 15:04:05")))
	}

	builder.WriteString(fmt.Sprintf("\n🔗 *Ver no Sentry:* %s", issue.WebURL))

	return builder.String()
}

func (w *SentryWebhook) createCardMessage() GChatMessage {
	emoji := w.getLevelEmoji()
	priorityEmoji := w.getPriorityEmoji()
	issue := w.Data.Issue

	widgets := []Widget{
		{
			KeyValue: &KeyValue{
				TopLabel: "Issue ID",
				Content:  issue.ShortID,
				Icon:     "BOOKMARK",
			},
		},
		{
			KeyValue: &KeyValue{
				TopLabel: "Projeto",
				Content:  issue.Project.Name,
				Icon:     "DESCRIPTION",
			},
		},
		{
			KeyValue: &KeyValue{
				TopLabel: "Nível",
				Content:  strings.ToUpper(issue.Level),
				Icon:     "ERROR",
			},
		},
		{
			KeyValue: &KeyValue{
				TopLabel: "Prioridade",
				Content:  fmt.Sprintf("%s %s", priorityEmoji, strings.ToUpper(issue.Priority)),
				Icon:     "STAR",
			},
		},
		{
			KeyValue: &KeyValue{
				TopLabel: "Plataforma",
				Content:  issue.Platform,
				Icon:     "COMPUTER",
			},
		},
	}

	if issue.Culprit != "" {
		widgets = append(widgets, Widget{
			KeyValue: &KeyValue{
				TopLabel: "Origem",
				Content:  issue.Culprit,
				Icon:     "MAP_PIN",
			},
		})
	}

	if issue.Count != "" {
		widgets = append(widgets, Widget{
			KeyValue: &KeyValue{
				TopLabel: "Ocorrências",
				Content:  issue.Count,
				Icon:     "MULTIPLE_PEOPLE",
			},
		})
	}

	widgets = append(widgets, Widget{
		KeyValue: &KeyValue{
			TopLabel: "Status",
			Content:  fmt.Sprintf("%s (%s)", issue.Status, issue.Substatus),
			Icon:     "FLAG",
		},
	})

	if !issue.FirstSeen.IsZero() {
		widgets = append(widgets, Widget{
			KeyValue: &KeyValue{
				TopLabel: "Primeira Ocorrência",
				Content:  issue.FirstSeen.Format("02/01/2006 15:04:05"),
				Icon:     "CLOCK",
			},
		})
	}

	// Botão para ver no Sentry
	buttonWidget := Widget{
		Buttons: []Button{
			{
				TextButton: TextButton{
					Text: "Ver no Sentry",
					OnClick: OnClick{
						OpenLink: OpenLink{
							URL: issue.WebURL,
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
					Subtitle: issue.Title,
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

	var webhook SentryWebhook

	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		log.Printf("Erro ao decodificar payload: %v", err)
		http.Error(w, "erro ao decodificar payload", http.StatusBadRequest)
		return
	}

	issue := webhook.Data.Issue
	log.Printf("Webhook recebido - Action: %s, Projeto: %s, Issue: %s, Nível: %s, Título: %s",
		webhook.Action, issue.Project.Name, issue.ShortID, issue.Level, issue.Title)

	// Só processa se for uma nova issue ou se for configurado para processar todas as ações
	processAllActions := os.Getenv("PROCESS_ALL_ACTIONS") == "true"
	if webhook.Action != "created" && !processAllActions {
		log.Printf("Ignorando ação: %s (apenas 'created' é processada por padrão)", webhook.Action)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK - Action ignored"))
		return
	}

	useCards := os.Getenv("USE_CARDS") == "true"

	var message interface{}
	if useCards {
		message = webhook.createCardMessage()
	} else {
		message = GChatMessage{
			Text: webhook.formatSimpleMessage(),
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

	if os.Getenv("PROCESS_ALL_ACTIONS") == "true" {
		log.Printf("🔄 Processando todas as ações do Sentry")
	} else {
		log.Printf("🆕 Processando apenas issues criadas (action: created)")
	}

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
