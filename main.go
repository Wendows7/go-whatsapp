package main

import (
	"context"
	"fmt"
	"go-whatsapp/config"
	"go-whatsapp/database"
	api "go-whatsapp/http"
	whatsappWebhook "go-whatsapp/whatsapp"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	env  = config.LoadEnv()
	port = ":" + env["APP_PORT"]
)

func main() {
	ctx := context.Background()

	client := whatsappWebhook.InitClient(ctx)

	db, err := database.InitDb()
	if err != nil {
		panic(fmt.Sprintf("Gagal koneksi DB: %v", err))
	}

	err = whatsappWebhook.SaveGroup(db, client) // init API Endpoint
	if err != nil {
		panic(fmt.Sprintf("Gagal simpan grup: %v", err))
	}
	api.SendAlerting(client, ctx) // init API Endpoint
	api.SendMessage(client, ctx)  // init API Endpoint
	//api.SendGroup(client, ctx)    // init API Endpoint

	// Init HTTP server
	go func() {
		fmt.Println("HTTP server running di", port)
		err := http.ListenAndServe(port, nil)
		if err != nil {
			panic(err)
		}
	}()

	// Tunggu signal untuk graceful shutdown
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch

	client.Disconnect()
	fmt.Println("Client WhatsApp terputus. Program selesai.")

}

//func getGroupList() {
//	client.Connect()
//	groups, err := client.GetJoinedGroups()
//	if err != nil {
//		panic(fmt.Sprintf("Gagal ambil grup: %v", err))
//	}
//
//	for _, group := range groups {
//		fmt.Println("Group JID:", group.JID.String())
//		fmt.Println("Group Name:", group.Name)
//	}
//}
