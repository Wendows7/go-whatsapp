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
	"time"
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

// GenerateNewQR membuat device baru, connect, dan mengembalikan string QR Code.
// Fungsi ini juga menjalankan goroutine background untuk menangani event success/timeout.
func GenerateNewQR(container *sqlstore.Container) (string, *whatsmeow.Client, error) {
	// 1. Buat slot device baru di database (Wajib untuk Multi-Device)
	device := container.NewDevice()

	// 2. Buat instance client baru
	client := whatsmeow.NewClient(device, log.Noop)

	// 3. Dapatkan channel QR
	qrChan, _ := client.GetQRChannel(context.Background())

	// 4. Connect ke WhatsApp
	if err := client.Connect(); err != nil {
		return "", nil, fmt.Errorf("gagal connect client baru: %v", err)
	}

	// 5. Tunggu QR Code pertama muncul (Blocking sementara dengan timeout)
	// Kita gunakan Select agar API tidak hang selamanya jika ada error
	var qrCode string

	select {
	case evt := <-qrChan:
		if evt.Event == "code" {
			qrCode = evt.Code
		} else {
			return "", nil, fmt.Errorf("event pertama bukan code: %s", evt.Event)
		}
	case <-time.After(10 * time.Second):
		client.Disconnect()
		return "", nil, fmt.Errorf("timeout menunggu QR Code dari WhatsApp")
	}

	// 6. Jalankan Listener di Background (Goroutine)
	// PENTING: Kita harus tetap mendengarkan channel ini sampai 'success' atau 'timeout'
	// meskipun kita sudah return QR Code ke API.
	go func() {
		for evt := range qrChan {
			if evt.Event == "success" {
				fmt.Printf("Login Berhasil untuk Device JID: %s\n", client.Store.ID)

				// TODO: Di sini Anda bisa simpan client ke map global agar bisa dipakai kirim pesan
				// Contoh: globalClients[client.Store.ID.User] = client

				break // Keluar dari loop listener
			} else if evt.Event == "timeout" {
				fmt.Println("Login Timeout (User tidak scan)")
				client.Disconnect() // Matikan koneksi biar hemat resource
				break
			}
		}
	}()

	// Kembalikan QR Code string dan object client
	return qrCode, client, nil
}
