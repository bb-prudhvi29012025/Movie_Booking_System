package main

import (
	"log"
	"net/http"
	"project/db"
	"project/handlers"
)

func main() {
	db.InitDB()

	http.HandleFunc("/login", handlers.Login)
	http.HandleFunc("/theatres", handlers.GetTheatreDetails)
	http.HandleFunc("/movies", handlers.GetMovies)
	http.HandleFunc("/book-seat", handlers.Authenticate(handlers.BookSeat))

	http.HandleFunc("/theatre/add", handlers.InsertTheatre)
	http.HandleFunc("/theatre/update", handlers.UpdateTheatre)
	http.HandleFunc("/theatre/delete", handlers.DeleteTheatre)
	http.HandleFunc("/theatre/search", handlers.GetTheatreByTheatreName)

	http.HandleFunc("/movie/add", handlers.InsertMovie)
	http.HandleFunc("/movie/update", handlers.UpdateMovie)
	http.HandleFunc("/movie/delete", handlers.DeleteMovie)
	http.HandleFunc("/movie/search", handlers.GetMovieByMovieName)

	log.Println("Server running on http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
