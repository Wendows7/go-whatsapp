package http

import (
	"context"
	"encoding/json"
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
	Number     string `json:"number"`    // nomor tujuan, misal "6281234567890"
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
	h1 := func(w http.ResponseWriter, r *http.Request) {

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

		if req.Number == "" || req.Message == "" {
			http.Error(w, "Field 'number' and 'message' required", http.StatusBadRequest)
			return
		}
		var phoneNumber string
		if strings.Contains(req.Number, "@g.us") {
			phoneNumber = req.Number
		} else {
			phoneNumber = numberFormatin(req.Number) + "@s.whatsapp.net"
		}

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
	}
	http.HandleFunc("/api/send-message", h1)
	http.HandleFunc("/send-message", h1)

}

func SendAlerting(client *whatsmeow.Client, ctx context.Context) {
	//client.Connect()
	// Endpoint for alerting
	h1 := func(w http.ResponseWriter, r *http.Request) {

		var phoneNumber string
		numberData := r.URL.Query().Get("number")
		if strings.Contains(numberData, "@g.us") {
			phoneNumber = numberData
		} else {
			phoneNumber = numberFormatin(numberData) + "@s.whatsapp.net"
		}
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
	}
	http.HandleFunc("/api/send-alert", h1)
	http.HandleFunc("/alert-devops", h1)

}

//func SendGroup(client *whatsmeow.Client, ctx context.Context) {
//	// HTTP server untuk menerima perintah kirim pesan via webhook
//	http.HandleFunc("/api/webhook/send-group", func(w http.ResponseWriter, r *http.Request) {
//
//		if r.Method != http.MethodPost {
//			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
//			return
//		}
//
//		var req SendMessageRequest
//		err := json.NewDecoder(r.Body).Decode(&req)
//
//		if env["TOKEN_KEY"] != req.PrivateKey {
//			http.Error(w, "Unauthorized", http.StatusForbidden)
//			return
//		}
//
//		if err != nil {
//			http.Error(w, "Invalid JSON", http.StatusBadRequest)
//			return
//		}
//
//		if req.Number == "" || req.Message == "" {
//			http.Error(w, "Field 'to' and 'message' required", http.StatusBadRequest)
//			return
//		}
//		phoneNumber := req.Number
//
//		jid, err := types.ParseJID(phoneNumber)
//		if err != nil {
//			http.Error(w, "Nomor tujuan tidak valid", http.StatusBadRequest)
//			return
//		}
//
//		_, err = client.SendMessage(ctx, jid, &proto.Message{
//			Conversation: config.ProtoString(req.Message),
//		})
//		if err != nil {
//			http.Error(w, "Gagal kirim pesan: "+err.Error(), http.StatusInternalServerError)
//			return
//		}
//
//		w.WriteHeader(http.StatusOK)
//		w.Write([]byte("Pesan berhasil dikirim"))
//	})
//}
