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
)

type Response struct {
	Message string `json:"message"`
	Status  bool   `json:"status"`
}

type SendMessageRequest struct {
	Number        string    `json:"number"`         // nomor tujuan, misal "6281234567890"
	Message       string    `json:"message"`        // isi pesan
	PrivateKey    string    `json:"token_key"`      // kunci privat untuk otorisasi
	MentionNumber *[]string `json:"mention_number"` // nomor yang di mention dalam grup
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
			w.WriteHeader(http.StatusMethodNotAllowed)
			response := Response{
				Message: "Gagal mengirim pesan",
				Status:  false,
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		var req SendMessageRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if env["TOKEN_KEY"] != req.PrivateKey {
			w.WriteHeader(http.StatusForbidden)
			response := Response{
				Message: "Gagal mengirim pesan",
				Status:  false,
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response := Response{
				Message: "Gagal mengirim pesan",
				Status:  false,
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if req.Number == "" || req.Message == "" {
			w.WriteHeader(http.StatusBadRequest)
			response := Response{
				Message: "Gagal mengirim pesan",
				Status:  false,
			}
			json.NewEncoder(w).Encode(response)
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
			w.WriteHeader(http.StatusBadRequest)
			response := Response{
				Message: "Gagal mengirim pesan",
				Status:  false,
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Handle mention numbers if provided
		if req.MentionNumber != nil && len(*req.MentionNumber) > 0 {

			mentionedJIDs := []string{}
			mentionText := req.Message + "\n"

			for _, num := range *req.MentionNumber {
				//formatted := numberFormatin(num) + "@s.whatsapp.net"
				formatedNum := num + "@s.whatsapp.net"
				mentionFinalJID, err := types.ParseJID(formatedNum)

				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(Response{
						Message: "Nomor mention tidak valid: " + num,
						Status:  false,
					})
					return
				}

				mentionText += "@" + mentionFinalJID.User + " "
				mentionedJIDs = append(mentionedJIDs, formatedNum)
			}

			_, err = client.SendMessage(ctx, jid, &proto.Message{
				ExtendedTextMessage: &proto.ExtendedTextMessage{
					Text: config.ProtoString(mentionText),
					ContextInfo: &proto.ContextInfo{
						MentionedJID: mentionedJIDs,
					},
				},
			})

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(Response{
					Message: "Gagal mengirim pesan",
					Status:  false,
				})
				return
			}

			json.NewEncoder(w).Encode(Response{
				Message: "Pesan berhasil dikirim",
				Status:  true,
			})
			return

		} else {
			_, err = client.SendMessage(ctx, jid, &proto.Message{
				Conversation: config.ProtoString(req.Message),
			})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				response := Response{
					Message: "Gagal mengirim pesan",
					Status:  false,
				}
				json.NewEncoder(w).Encode(response)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// return json response
			response := Response{
				Message: "Pesan berhasil dikirim",
				Status:  true,
			}
			// Encode struct menjadi JSON dan kirim ke client
			json.NewEncoder(w).Encode(response)
		}

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
			w.WriteHeader(http.StatusForbidden)
			response := Response{
				Message: "Gagal mengirim pesan",
				Status:  false,
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			response := Response{
				Message: "Gagal mengirim pesan: ",
				Status:  false,
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		var reqAlerting AlertPayload
		err := json.NewDecoder(r.Body).Decode(&reqAlerting)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response := Response{
				Message: "Gagal mengirim pesan",
				Status:  false,
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		jid, err := types.ParseJID(phoneNumber)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response := Response{
				Message: "Gagal mengirim pesan",
				Status:  false,
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		alertingMessage := "*Alertname:* " + reqAlerting.Alerts[0].Labels["alertname"] + "\n*Severity:* " + reqAlerting.Alerts[0].Labels["severity"] +
			"\n*Title:* " + reqAlerting.Alerts[0].Annotations["title"] + "\n*Description:* " + reqAlerting.Alerts[0].Annotations["description"]

		_, err = client.SendMessage(ctx, jid, &proto.Message{
			Conversation: config.ProtoString(alertingMessage),
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := Response{
				Message: "Gagal mengirim pesan",
				Status:  false,
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusOK)
		response := Response{
			Message: "Berhasil Mengirim Pesan",
			Status:  true,
		}
		json.NewEncoder(w).Encode(response)
	}
	http.HandleFunc("/api/send-alert", h1)
	http.HandleFunc("/alert-devops", h1)

}
