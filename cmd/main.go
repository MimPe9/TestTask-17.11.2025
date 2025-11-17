package main

import (
	"context"
	"linkchecker/handlers"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {

	server := &http.Server{
		Addr:    ":8020",
		Handler: nil,
	}

	http.HandleFunc("/check", handlers.CheckHandler)
	http.HandleFunc("/list", handlers.ListHandler)

	var wg sync.WaitGroup
	handlers.SetWaitGroup(&wg)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Ошибка сервера: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Ошибка при остановке сервера: %v\n", err)
	}

	log.Println("Ожидание завершения операций")
	wg.Wait()
	handlers.SaveState()
	log.Println("Сервер остановлен")
}
