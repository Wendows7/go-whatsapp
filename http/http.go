package http

import (
	"context"
	"encoding/json"
	"fmt"
	"go-whatsapp/config"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"net/http"
	"strings"
)

var (
	env = config.LoadEnv()
	//client, ctx = whatsapp.InitClient()
)

type SendMessageRequest struct {
	To         string `json:"to"`        // nomor tujuan, misal "6281234567890"
	Message    string `json:"message"`   // isi pesan
	PrivateKey string `json:"token_key"` // kunci privat untuk otorisasi
}

type AlertPayload struct {
	Alerts []Alert `json:"alerts"`
}

type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

func numberFormatin(number string) string {
	var formattedNumber string
	if strings.HasPrefix(number, "08") {
		formattedNumber = "62" + number[1:]
	} else if strings.HasPrefix(number, "+62") {
		formattedNumber = number[1:]
	}
	return formattedNumber
}

func SendMessage(client *whatsmeow.Client, ctx context.Context) {
	//client.Connect()

	// HTTP server untuk menerima perintah kirim pesan via webhook
	http.HandleFunc("/api/webhook/send-message", func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req SendMessageRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		fmt.Println(env["TOKEN_KEY"], req.PrivateKey)
		if env["TOKEN_KEY"] != req.PrivateKey {
			http.Error(w, "Unauthorized", http.StatusForbidden)
			return
		}

		if err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.To == "" || req.Message == "" {
			http.Error(w, "Field 'to' and 'message' required", http.StatusBadRequest)
			return
		}
		phoneNumber := numberFormatin(req.To) + "@s.whatsapp.net"
		fmt.Println(phoneNumber)

		jid, err := types.ParseJID(phoneNumber)
		if err != nil {
			http.Error(w, "Nomor tujuan tidak valid", http.StatusBadRequest)
			return
		}

		_, err = client.SendMessage(ctx, jid, &proto.Message{
			Conversation: config.ProtoString(req.Message),
		})
		if err != nil {
			http.Error(w, "Gagal kirim pesan: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Pesan berhasil dikirim"))
	})

}

func SendAlerting(client *whatsmeow.Client, ctx context.Context) {
	//client.Connect()
	// Endpoint for alerting
	http.HandleFunc("/api/webhook/send-alert", func(w http.ResponseWriter, r *http.Request) {

		phoneNumber := numberFormatin(r.URL.Query().Get("number")) + "@s.whatsapp.net"
		if r.URL.Query().Get("token_key") != env["TOKEN_KEY"] {
			http.Error(w, "Unauthorized", http.StatusForbidden)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var reqAlerting AlertPayload
		err := json.NewDecoder(r.Body).Decode(&reqAlerting)

		if err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		jid, err := types.ParseJID(phoneNumber)
		if err != nil {
			http.Error(w, "Nomor tujuan tidak valid", http.StatusBadRequest)
			return
		}

		alertingMessage := "*Alertname:* " + reqAlerting.Alerts[0].Labels["alertname"] + "\n*Severity:* " + reqAlerting.Alerts[0].Labels["severity"] +
			"\n*Title:* " + reqAlerting.Alerts[0].Annotations["title"] + "\n*Description:* " + reqAlerting.Alerts[0].Annotations["description"]

		_, err = client.SendMessage(ctx, jid, &proto.Message{
			Conversation: config.ProtoString(alertingMessage),
		})
		if err != nil {
			http.Error(w, "Gagal kirim pesan: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Pesan berhasil dikirim"))
	})
}

func SendGroup(client *whatsmeow.Client, ctx context.Context) {
	// HTTP server untuk menerima perintah kirim pesan via webhook
	http.HandleFunc("/api/webhook/send-group", func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req SendMessageRequest
		err := json.NewDecoder(r.Body).Decode(&req)

		if env["TOKEN_KEY"] != req.PrivateKey {
			http.Error(w, "Unauthorized", http.StatusForbidden)
			return
		}

		if err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.To == "" || req.Message == "" {
			http.Error(w, "Field 'to' and 'message' required", http.StatusBadRequest)
			return
		}
		phoneNumber := req.To

		jid, err := types.ParseJID(phoneNumber)
		if err != nil {
			http.Error(w, "Nomor tujuan tidak valid", http.StatusBadRequest)
			return
		}

		_, err = client.SendMessage(ctx, jid, &proto.Message{
			Conversation: config.ProtoString(req.Message),
		})
		if err != nil {
			http.Error(w, "Gagal kirim pesan: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Pesan berhasil dikirim"))
	})
}
