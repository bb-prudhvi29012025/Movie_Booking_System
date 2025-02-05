package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"project/db"
	"project/models"
)

func GetMovies(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query("SELECT id, movie_name, description FROM movie")
	if err != nil {
		log.Println("Database query failed:", err)
		http.Error(w, "Failed to fetch movies", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var movies []models.Movie
	for rows.Next() {
		var movie models.Movie
		err := rows.Scan(&movie.ID, &movie.Movie_Name, &movie.Description)
		if err != nil {
			log.Println("Error scanning row:", err)
			continue
		}
		movies = append(movies, movie)
	}

	if err := rows.Err(); err != nil {
		log.Println("Row iteration error:", err)
		http.Error(w, "Failed to process movie details", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if len(movies) == 0 {
		w.Write([]byte("[]"))
		return
	}
	json.NewEncoder(w).Encode(movies)
}