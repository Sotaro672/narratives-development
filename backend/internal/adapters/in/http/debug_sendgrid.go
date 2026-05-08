package httpin

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	sendgrid "github.com/sendgrid/sendgrid-go"
	sgmail "github.com/sendgrid/sendgrid-go/helpers/mail"
)

func DebugSendGridHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := os.Getenv("SENDGRID_API_KEY")
	from := os.Getenv("SENDGRID_FROM")

	if apiKey == "" || from == "" {
		http.Error(w, "SendGrid env not set", http.StatusInternalServerError)
		return
	}

	client := sendgrid.NewSendClient(apiKey)

	fromMail := sgmail.NewEmail("Debug", from)
	toMail := sgmail.NewEmail("You", from) // 自分宛に送信
	subject := "SendGrid Debug Test"
	plainText := "This is a debug email from Narratives backend."

	message := sgmail.NewSingleEmail(fromMail, subject, toMail, plainText, "")

	resp, err := client.Send(message)
	if err != nil {
		log.Println("SendGrid error:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out := map[string]any{
		"status":  resp.StatusCode,
		"body":    resp.Body,
		"headers": resp.Headers,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
