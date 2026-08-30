package main

import (
	"context"
	"log"
	"net/http"

	"os"

	"github.com/Komanabelli-Balaji/booking-system/internal/adapters/redis"
	"github.com/Komanabelli-Balaji/booking-system/internal/booking"
	"github.com/Komanabelli-Balaji/booking-system/internal/utils"
	"github.com/Komanabelli-Balaji/booking-system/internal/ws"
)

func main() {
	mux := http.NewServeMux()

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	rdb := redis.NewClient(redisAddr)
	store := booking.NewRedisStore(rdb)
	svc := booking.NewService(store)

	hub := ws.NewHub()

	booking.ListenForExpirations(context.Background(), rdb, hub)

	bookingHandler := booking.NewHandler(svc, hub)

	mux.Handle("GET /", http.FileServer(http.Dir("static")))
	mux.HandleFunc("GET /ws", hub.HandleWebSocket)
	mux.HandleFunc("GET /movies", listMovies)

	mux.HandleFunc("GET /movies/{movieID}/seats", bookingHandler.ListSeats)
	mux.HandleFunc("POST /movies/{movieID}/seats/{seatID}/hold", bookingHandler.HoldSeat)

	mux.HandleFunc("PUT /sessions/{sessionID}/confirm", bookingHandler.ConfirmSession)
	mux.HandleFunc("DELETE /sessions/{sessionID}", bookingHandler.ReleaseSession)

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func listMovies(w http.ResponseWriter, r *http.Request) {
	utils.WriteJSON(w, http.StatusOK, movies)
}

var movies = []movieResponse{
	{ID: "paradise", Title: "Paradise", Rows: 6, SeatsPerRow: 10},
	{ID: "bnd", Title: "Spider-Man: Brand New Day", Rows: 7, SeatsPerRow: 10},
	{ID: "kgf", Title: "KGF", Rows: 8, SeatsPerRow: 10},
}

type movieResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Rows        int    `json:"rows"`
	SeatsPerRow int    `json:"seats_per_row"`
}
