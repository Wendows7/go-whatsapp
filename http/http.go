package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-whatsapp/config"
	"go-whatsapp/whatsapp" // Pastikan package ini berisi fungsi GenerateNewQR

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	env = config.LoadEnv()
)

// --- STRUCTS ---

type Response struct {
	Message string `json:"message"`
	Status  bool   `json:"status"`
}

type SendMessageRequest struct {
	Sender        string    `json:"sender"`         // OPTIONAL: No HP pengirim (jika bot punya banyak nomor)
	Number        string    `json:"number"`         // No HP tujuan
	Message       string    `json:"message"`        // Isi pesan
	PrivateKey    string    `json:"token_key"`      // Token auth
	MentionNumber *[]string `json:"mention_number"` // List nomor yang di-mention
}

type AlertPayload struct {
	Alerts []Alert `json:"alerts"`
}

type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

type QRRequest struct {
	PrivateKey string `json:"private_key"`
}

// --- HELPER FUNCTIONS ---

func numberFormatin(number string) string {
	var formattedNumber string
	if strings.HasPrefix(number, "08") {
		formattedNumber = "62" + number[1:]
	} else if strings.HasPrefix(number, "+62") {
		formattedNumber = number[1:]
	} else {
		formattedNumber = number
	}
	return formattedNumber
}

func protoString(s string) *string {
	return &s
}

// getClient memilih bot mana yang akan mengirim pesan.
// sender: nomor HP bot yang diinginkan (opsional).
func getClient(sender string, activeClients map[string]*whatsmeow.Client, mu *sync.RWMutex) (*whatsmeow.Client, error) {
	mu.RLock()
	defer mu.RUnlock()

	// 1. Jika user meminta sender spesifik
	if sender != "" {
		senderJID := numberFormatin(sender) // pastikan format 62xxx
		if client, ok := activeClients[senderJID]; ok {
			return client, nil
		}
		return nil, fmt.Errorf("sesi whatsapp untuk pengirim %s tidak ditemukan/belum login", sender)
	}

	// 2. Jika tidak minta spesifik, dan hanya ada 1 bot aktif -> Pakai itu
	if len(activeClients) == 1 {
		for _, client := range activeClients {
			return client, nil
		}
	}

	// 3. Jika bot banyak tapi user tidak memilih -> Error (Ambigu)
	if len(activeClients) > 1 {
		return nil, fmt.Errorf("terdapat %d akun aktif, harap isi parameter 'sender' di JSON", len(activeClients))
	}

	return nil, fmt.Errorf("tidak ada klien whatsapp yang terhubung")
}

// --- ROUTE REGISTRATIONS ---

// RegisterQRRoute mendaftarkan endpoint /api/get-qr
func RegisterQRRoute(container *sqlstore.Container, activeClients map[string]*whatsmeow.Client, mu *sync.RWMutex) {

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req QRRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if env["TOKEN_KEY"] != req.PrivateKey {
			http.Error(w, "Unauthorized", http.StatusForbidden)
			return
		}

		// Generate Device & Client Baru
		qrCodeString, newClient, err := whatsapp.GenerateNewQR(container)
		if err != nil {
			http.Error(w, "Error generating QR: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// LISTENER: Tunggu sampai user scan QR (Background Process)
		newClient.AddEventHandler(func(evt interface{}) {
			switch v := evt.(type) {
			case *events.PairSuccess:
				// Login Berhasil!
				fmt.Printf("[New Login] ID: %s, Device: %s\n", v.ID.User, v.BusinessName)

				// Simpan ke map activeClients dengan aman
				mu.Lock()
				activeClients[v.ID.User] = newClient
				mu.Unlock()

			case *events.Disconnected:
				// Jika timeout / putus sebelum scan
				fmt.Println("[New Login] Client disconnected/timeout")
			}
		})

		// Return QR Code ke API User
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "waiting_for_scan",
			"qr_code": qrCodeString,
			"message": "Silakan render string qr_code ini. Expired dalam 20-30 detik.",
			"expired": "60 seconds",
		})
	}

	http.HandleFunc("/api/qr", handler)
}

// RegisterSendMessage mendaftarkan endpoint /api/send-message
func RegisterSendMessage(activeClients map[string]*whatsmeow.Client, mu *sync.RWMutex) {

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req SendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Validasi Token
		if env["TOKEN_KEY"] != req.PrivateKey {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(Response{Message: "Token key tidak valid", Status: false})
			return
		}

		// Pilih Bot Pengirim
		client, err := getClient(req.Sender, activeClients, mu)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(Response{Message: err.Error(), Status: false})
			return
		}

		// Format Nomor Tujuan
		var phoneNumber string
		if strings.Contains(req.Number, "@g.us") {
			phoneNumber = req.Number
		} else {
			phoneNumber = numberFormatin(req.Number) + "@s.whatsapp.net"
		}

		jid, err := types.ParseJID(phoneNumber)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(Response{Message: "Format nomor tujuan salah", Status: false})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Logic Kirim Pesan (Text Biasa / Mention)
		if req.MentionNumber != nil && len(*req.MentionNumber) > 0 {
			mentionedJIDs := []string{}
			mentionText := req.Message + "\n"

			for _, num := range *req.MentionNumber {
				formatedNum := numberFormatin(num) + "@s.whatsapp.net"
				mentionFinalJID, err := types.ParseJID(formatedNum)
				if err != nil {
					continue
				}
				mentionText += "@" + mentionFinalJID.User + " "
				mentionedJIDs = append(mentionedJIDs, formatedNum)
			}

			_, err = client.SendMessage(ctx, jid, &proto.Message{
				ExtendedTextMessage: &proto.ExtendedTextMessage{
					Text: protoString(mentionText),
					ContextInfo: &proto.ContextInfo{
						MentionedJID: mentionedJIDs,
					},
				},
			})
		} else {
			// Kirim Text Biasa
			_, err = client.SendMessage(ctx, jid, &proto.Message{
				Conversation: protoString(req.Message),
			})
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(Response{Message: "Gagal kirim: " + err.Error(), Status: false})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{Message: "Pesan terkirim via " + client.Store.ID.User, Status: true})
	}

	http.HandleFunc("/api/send-message", handler)
	http.HandleFunc("/send-message", handler)
}

// RegisterAlertRoute mendaftarkan endpoint /api/send-alert
func RegisterAlertRoute(activeClients map[string]*whatsmeow.Client, mu *sync.RWMutex) {

	handler := func(w http.ResponseWriter, r *http.Request) {
		// Validasi via Query Params
		tokenKey := r.URL.Query().Get("token_key")
		targetNumber := r.URL.Query().Get("number")
		senderNum := r.URL.Query().Get("sender") // Opsional: pilih bot pengirim

		if tokenKey != env["TOKEN_KEY"] {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(Response{Message: "Unauthorized", Status: false})
			return
		}

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Pilih Bot Pengirim
		client, err := getClient(senderNum, activeClients, mu)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(Response{Message: err.Error(), Status: false})
			return
		}

		// Format Nomor Tujuan
		var phoneNumber string
		if strings.Contains(targetNumber, "@g.us") {
			phoneNumber = targetNumber
		} else {
			phoneNumber = numberFormatin(targetNumber) + "@s.whatsapp.net"
		}

		jid, err := types.ParseJID(phoneNumber)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(Response{Message: "Nomor tujuan invalid", Status: false})
			return
		}

		// Decode Payload AlertManager/Prometheus
		var reqAlerting AlertPayload
		if err := json.NewDecoder(r.Body).Decode(&reqAlerting); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(Response{Message: "Invalid JSON Payload", Status: false})
			return
		}

		// Construct Pesan Alert
		if len(reqAlerting.Alerts) > 0 {
			alert := reqAlerting.Alerts[0]
			alertingMessage := fmt.Sprintf(
				"*Alertname:* %s\n*Severity:* %s\n*Title:* %s\n*Description:* %s",
				alert.Labels["alertname"],
				alert.Labels["severity"],
				alert.Annotations["title"],
				alert.Annotations["description"],
			)

			_, err = client.SendMessage(context.Background(), jid, &proto.Message{
				Conversation: protoString(alertingMessage),
			})

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(Response{Message: "Gagal kirim alert: " + err.Error(), Status: false})
				return
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Response{Message: "Alert berhasil dikirim via " + client.Store.ID.User, Status: true})
		} else {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Response{Message: "Empty alerts list, nothing sent.", Status: true})
		}
	}

	http.HandleFunc("/api/send-alert", handler)
	http.HandleFunc("/alert-devops", handler)
}

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "http://127.0.0.1:5500")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
