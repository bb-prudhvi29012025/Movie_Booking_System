package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"project/db"
)

func GetTheatreDetails(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query("SELECT id, theatre_name, total_rooms FROM theatre")
	if err != nil {
		log.Println("Database query failed:", err)
		http.Error(w, "Failed to fetch theatre details", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var theatres []struct {
		ID           int    `json:"id"`
		Theatre_Name string `json:"theatre_name"`
		TotalRooms   int    `json:"total_rooms"`
	}

	for rows.Next() {
		var theatre struct {
			ID           int    `json:"id"`
			Theatre_Name string `json:"theatre_name"`
			TotalRooms   int    `json:"total_rooms"`
		}
		err := rows.Scan(&theatre.ID, &theatre.Theatre_Name, &theatre.TotalRooms)
		if err != nil {
			log.Println("Error scanning row:", err)
			http.Error(w, "Error reading theatre data", http.StatusInternalServerError)
			return
		}
		theatres = append(theatres, theatre)
	}

	if err := rows.Err(); err != nil {
		log.Println("Row iteration error:", err)
		http.Error(w, "Failed to process theatre details", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if len(theatres) == 0 {
		w.Write([]byte("[]"))
		return
	}

	if err := json.NewEncoder(w).Encode(theatres); err != nil {
		log.Println("Error encoding theatre data:", err)
		http.Error(w, "Failed to encode data", http.StatusInternalServerError)
		return
	}
}
