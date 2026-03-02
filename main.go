package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go-whatsapp/config"
	"go-whatsapp/database"
	api "go-whatsapp/http" // Pastikan package ini sesuai path project Anda
	whatsappWebhook "go-whatsapp/whatsapp"

	_ "github.com/lib/pq" // Driver Postgres
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	log "go.mau.fi/whatsmeow/util/log"
)

var (
	env  = config.LoadEnv()
	port = ":" + env["APP_PORT"]
)

func main() {
	// 1. Setup Context & Database Container (Whatsmeow)
	// Kita init DB Whatsmeow di sini agar bisa handle BANYAK device
	dbURI := database.DbUri() // Pastikan fungsi ini return string connection postgres yang benar

	container, err := sqlstore.New(context.Background(), "postgres", dbURI, log.Noop)
	if err != nil {
		panic(fmt.Sprintf("Gagal koneksi DB Whatsmeow: %v", err))
	}

	// 2. Siapkan Map untuk menyimpan Client yang aktif (Multi-Device)
	// Key: Nomor HP (JID User), Value: Pointer ke Client
	activeClients := make(map[string]*whatsmeow.Client)

	var mu sync.RWMutex // Mutex untuk mencegah crash saat akses map bersamaan

	// 3. Load Semua Device yang Sudah Ada di Database (Saat Restart)
	fmt.Println("Memuat sesi WhatsApp yang tersimpan...")
	devices, err := container.GetAllDevices(context.Background())
	if err != nil {
		panic(fmt.Sprintf("Gagal ambil device dari DB: %v", err))
	}

	for _, device := range devices {
		// Buat client untuk setiap device
		client := whatsmeow.NewClient(device, log.Noop)

		// Connect
		if err := client.Connect(); err != nil {
			fmt.Printf("Gagal connect device %s: %v\n", device.ID, err)
		} else {
			// Simpan ke map activeClients
			mu.Lock()
			activeClients[client.Store.ID.User] = client
			mu.Unlock()
			fmt.Printf("Berhasil connect kembali: %s\n", client.Store.ID.User)
		}
	}

	// 4. Background Task: Simpan Group (Opsional)
	// Kita jalankan untuk setiap client yang aktif
	go func() {
		// Beri jeda sedikit biar koneksi stabil
		time.Sleep(5 * time.Second)

		dbApp, err := database.InitDb() // DB untuk aplikasi/grup
		if err != nil {
			fmt.Printf("Gagal koneksi DB Aplikasi: %v\n", err)
			return
		}

		mu.RLock()
		defer mu.RUnlock()

		// Loop semua client aktif dan sync grup mereka
		for _, client := range activeClients {
			go func(c *whatsmeow.Client) {
				fmt.Println("Syncing groups for:", c.Store.ID.User)
				err := whatsappWebhook.SaveGroup(dbApp, c)
				if err != nil {
					fmt.Printf("Gagal simpan grup %s: %v\n", c.Store.ID.User, err)
				}
			}(client)
		}
	}()

	// 5. Init API Endpoints (Tanpa Handler Struct)
	// Kita pass container, map client, dan mutex ke fungsi register di package http

	// API: Get New QR (Multi Device)
	api.RegisterQRRoute(container, activeClients, &mu)

	// API: Send Message & Alert (Perlu disesuaikan di file http.go agar menerima map)
	// Asumsi: Anda sudah mengubah fungsi SendMessage & SendAlerting di http.go
	// untuk menerima (activeClients, &mu) bukannya (client).
	api.RegisterSendMessage(activeClients, &mu)
	api.RegisterAlertRoute(activeClients, &mu)

	// 6. Init HTTP Server
	server := &http.Server{Addr: port}

	go func() {
		fmt.Println("HTTP server running di", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	// 7. Graceful Shutdown
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch

	fmt.Println("Shutting down...")

	// Disconnect semua client
	mu.Lock()
	for _, client := range activeClients {
		client.Disconnect()
	}

	mu.Unlock()

	fmt.Println("Semua Client WhatsApp terputus. Program selesai.")
}
