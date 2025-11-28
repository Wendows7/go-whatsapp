package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/mdp/qrterminal/v3"
	"go-whatsapp/database"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	log "go.mau.fi/whatsmeow/util/log"
	"os"
)

func InitClient(ctx context.Context) *whatsmeow.Client {

	// Ganti dengan konfigurasi database PostgreSQL mu
	dbURI := database.DbUri()

	container, err := sqlstore.New(ctx, "postgres", dbURI, log.Noop)
	if err != nil {
		panic(fmt.Sprintf("Gagal koneksi DB: %v", err))
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		panic(fmt.Sprintf("Gagal ambil device: %v", err))
	}

	client := whatsmeow.NewClient(deviceStore, log.Noop)

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(ctx)
		err := client.Connect()
		if err != nil {
			panic(err)
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("Scan QR Code ini dengan WhatsApp:")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				//test, err := client.PairPhone(ctx, "6282165241668", true, whatsmeow.PairClientChrome, "Google Chrome (macOS)")
				//if err != nil {
				//	fmt.Println("Gagal pair:", err)
				//} else {
				//	fmt.Println("Pair berhasil:", test)
				//}
			} else if evt.Event == "success" {
				fmt.Println("Login berhasil!")
				break
			} else if evt.Event == "timeout" || evt.Event == "error" {
				fmt.Println("Login gagal:", evt.Event)
				os.Exit(1)
			}
		}
	} else {
		err := client.Connect()
		if err != nil {
			panic(err)
		}
		fmt.Println("Berhasil terkoneksi ke WhatsApp!")
	}

	//	return variable client and ctx
	return client
}

func SaveGroup(db *sql.DB, client *whatsmeow.Client) error {

	// create table whatsapp_groups if not exists
	_, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS whatsapp_groups (
			id SERIAL PRIMARY KEY,
			jid TEXT UNIQUE NOT NULL,
			name TEXT);
	`)
	if err != nil {
		return fmt.Errorf("Gagal buat tabel whatsapp_groups: %v", err)
	}

	groups, err := client.GetJoinedGroups(context.Background())
	if err != nil {
		return fmt.Errorf("Gagal ambil grup: %v", err)
	}

	for _, group := range groups {
		_, err := db.Exec(`
            INSERT INTO whatsapp_groups (jid, name)
            VALUES ($1, $2)
            ON CONFLICT (jid) DO UPDATE SET name = EXCLUDED.name
        `, group.JID.String(), group.Name)

		if err != nil {
			return fmt.Errorf("gagal insert grup %s: %w", group.JID.String(), err)
		}
	}

	fmt.Println("Berhasil menyimpan grup ke database")
	return nil
}

//func ConnectClient(client *whatsmeow.Client, ctx context.Context) {
//
//	if client.Store.ID == nil {
//		qrChan, _ := client.GetQRChannel(ctx)
//		err := client.Connect()
//		if err != nil {
//			panic(err)
//		}
//
//		for evt := range qrChan {
//			if evt.Event == "code" {
//				fmt.Println("Scan QR Code ini dengan WhatsApp:")
//				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
//			} else if evt.Event == "success" {
//				fmt.Println("Login berhasil!")
//				break
//			} else if evt.Event == "timeout" || evt.Event == "error" {
//				fmt.Println("Login gagal:", evt.Event)
//				os.Exit(1)
//			}
//		}
//	} else {
//		err := client.Connect()
//		if err != nil {
//			panic(err)
//		}
//		fmt.Println("Berhasil terkoneksi ke WhatsApp!")
//	}
//}

//const (
//	messageAPIAddr = ":8080" // Port HTTP server untuk terima perintah kirim pesan
//)
//
//type Handler struct {
//	client *whatsmeow.Client
//}
//
//type SendMessageRequest struct {
//	To         string `json:"to"`          // nomor tujuan, misal "6281234567890"
//	Message    string `json:"message"`     // isi pesan
//	PrivateKey string `json:"private_key"` // kunci privat untuk otorisasi
//}
//
//type AlertPayload struct {
//	Alerts []Alert `json:"alerts"`
//}
//
//type Alert struct {
//	Labels      map[string]string `json:"labels"`
//	Annotations map[string]string `json:"annotations"`
//}
//
//func main() {
//	ctx := context.Background()
//
//	// Ganti dengan konfigurasi database PostgreSQL mu
//	dbURI := "postgres://" + env["DB_USERNAME"] + ":" + env["DB_PASSWORD"] + "@" + env["DB_HOSTNAME"] + ":" + env["DB_PORT"] + "/" + env["DB_NAME"] + "?sslmode=disable"
//
//	container, err := sqlstore.New(ctx, "postgres", dbURI, log.Noop)
//	if err != nil {
//		panic(fmt.Sprintf("Gagal koneksi DB: %v", err))
//	}
//
//	deviceStore, err := container.GetFirstDevice(ctx)
//	if err != nil {
//		panic(fmt.Sprintf("Gagal ambil device: %v", err))
//	}
//
//	client := whatsmeow.NewClient(deviceStore, log.Noop)
//	//handler := &Handler{client: client}
//	//client.AddEventHandler(handler.HandleEvent)
//
//	if client.Store.ID == nil {
//		qrChan, _ := client.GetQRChannel(ctx)
//		err = client.Connect()
//		if err != nil {
//			panic(err)
//		}
//
//		for evt := range qrChan {
//			if evt.Event == "code" {
//				fmt.Println("Scan QR Code ini dengan WhatsApp:")
//				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
//			} else if evt.Event == "success" {
//				fmt.Println("Login berhasil!")
//				break
//			} else if evt.Event == "timeout" || evt.Event == "error" {
//				fmt.Println("Login gagal:", evt.Event)
//				os.Exit(1)
//			}
//		}
//	} else {
//		err = client.Connect()
//		if err != nil {
//			panic(err)
//		}
//		fmt.Println("Berhasil terkoneksi ke WhatsApp!")
//	}
//
//	// HTTP server untuk menerima perintah kirim pesan via webhook
//	http.HandleFunc("/api/webhook/send-message", func(w http.ResponseWriter, r *http.Request) {
//
//		if r.Method != http.MethodPost {
//			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
//			return
//		}
//
//		var req SendMessageRequest
//		err := json.NewDecoder(r.Body).Decode(&req)
//
//		if env["PRIVATE_KEY"] != req.PrivateKey {
//			http.Error(w, "Unauthorized", http.StatusForbidden)
//			return
//		}
//
//		if err != nil {
//			http.Error(w, "Invalid JSON", http.StatusBadRequest)
//			return
//		}
//
//		if req.To == "" || req.Message == "" {
//			http.Error(w, "Field 'to' and 'message' required", http.StatusBadRequest)
//			return
//		}
//		phoneNumber := req.To + "@s.whatsapp.net"
//
//		jid, err := types.ParseJID(phoneNumber)
//		if err != nil {
//			http.Error(w, "Nomor tujuan tidak valid", http.StatusBadRequest)
//			return
//		}
//
//		_, err = client.SendMessage(ctx, jid, &proto.Message{
//			Conversation: protoString(req.Message),
//		})
//		if err != nil {
//			http.Error(w, "Gagal kirim pesan: "+err.Error(), http.StatusInternalServerError)
//			return
//		}
//
//		w.WriteHeader(http.StatusOK)
//		w.Write([]byte("Pesan berhasil dikirim"))
//	})
//
//	// Endpoint for alerting
//	http.HandleFunc("/api/webhook/send-alert", func(w http.ResponseWriter, r *http.Request) {
//
//		phoneNumber := r.URL.Query().Get("number") + "@s.whatsapp.net"
//		if r.URL.Query().Get("private_key") != env["PRIVATE_KEY"] {
//			http.Error(w, "Unauthorized", http.StatusForbidden)
//			return
//		}
//
//		if r.Method != http.MethodPost {
//			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
//			return
//		}
//
//		var reqAlerting AlertPayload
//		err := json.NewDecoder(r.Body).Decode(&reqAlerting)
//
//		if err != nil {
//			http.Error(w, "Invalid JSON", http.StatusBadRequest)
//			return
//		}
//
//		jid, err := types.ParseJID(phoneNumber)
//		if err != nil {
//			http.Error(w, "Nomor tujuan tidak valid", http.StatusBadRequest)
//			return
//		}
//
//		alertingMessage := "*Alertname:* " + reqAlerting.Alerts[0].Labels["alertname"] + "\n*Severity:* " + reqAlerting.Alerts[0].Labels["severity"] + "\n*Title:* " + reqAlerting.Alerts[0].Annotations["title"] + "\n*Description:* " + reqAlerting.Alerts[0].Annotations["description"]
//
//		_, err = client.SendMessage(ctx, jid, &proto.Message{
//			Conversation: protoString(alertingMessage),
//		})
//		if err != nil {
//			http.Error(w, "Gagal kirim pesan: "+err.Error(), http.StatusInternalServerError)
//			return
//		}
//
//		w.WriteHeader(http.StatusOK)
//		w.Write([]byte("Pesan berhasil dikirim"))
//	})
//
//	go func() {
//		fmt.Println("HTTP server running di", messageAPIAddr)
//		err := http.ListenAndServe(messageAPIAddr, nil)
//		if err != nil {
//			panic(err)
//		}
//	}()
//
//	// Tunggu signal untuk graceful shutdown
//	ch := make(chan os.Signal, 1)
//	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
//	<-ch
//
//	client.Disconnect()
//	fmt.Println("Client WhatsApp terputus. Program selesai.")
//}
//
//func protoString(s string) *string {
//	return &s
//}
