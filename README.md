# 🔔 Sentry → Google Chat Webhook (Go)

Servidor em Go que recebe webhooks do Sentry e envia mensagens formatadas para o Google Chat.

---

## 🚀 Deploy na Render

### 1. Crie um novo serviço no [Render](https://render.com)

- Tipo: **Web Service**
- Build Command: `go build -o app`
- Start Command: `./app`
- Runtime: **Docker**
- Porta: `8080`

### 2. Adicione as variáveis de ambiente

No painel da Render, vá em **Environment > Environment Variables**:

- `GCHAT_WEBHOOK`: sua URL de webhook do Google Chat.
- `PORT`: `8080` (Render usa isso por padrão)

---

## 💻 Rodando localmente

### 1. Clone o projeto

```bash
git clone https://github.com/seu-usuario/sentry-to-gchat.git
cd sentry-to-gchat
```