package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"project/db"
	"project/models"
	"project/utils"
)

func BookSeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var request models.BookingRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.NoOfSeats <= 0 {
		http.Error(w, "Number of seats must be a positive integer", http.StatusBadRequest)
		return
	}

	username, err := authenticateUser(r)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	if request.MovieName != "" && request.TheatreName == "" {
		rows, err := db.DB.Query("SELECT DISTINCT theatre_name FROM theatre t JOIN room r ON t.id = r.theatre_id WHERE r.movie_name = ?", request.MovieName)
		if err != nil {
			http.Error(w, "Database error when fetching theatres", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var theatres []string
		for rows.Next() {
			var theatreName string
			err := rows.Scan(&theatreName)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error fetching theatre data: %v", err), http.StatusInternalServerError)
				return
			}
			theatres = append(theatres, theatreName)
		}

		if len(theatres) == 0 {
			http.Error(w, "No theatres found screening this movie", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":         "Multiple theatres available. Choose one.",
			"theatre_choices": theatres,
		})
		return
	}

	var movieID int
	err = db.DB.QueryRow("SELECT id FROM movie WHERE movie_name = ?", request.MovieName).Scan(&movieID)
	if err == sql.ErrNoRows {
		http.Error(w, "Movie is not screening anywhere", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Database error when fetching movie: %v", err), http.StatusInternalServerError)
		return
	}

	var theatreID int
	err = db.DB.QueryRow("SELECT id FROM theatre WHERE theatre_name = ?", request.TheatreName).Scan(&theatreID)
	if err == sql.ErrNoRows {
		http.Error(w, "Theatre not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Database error when fetching theatre: %v", err), http.StatusInternalServerError)
		return
	}

	if request.RoomName != "" {
		var roomID int
		err = db.DB.QueryRow("SELECT id FROM room WHERE room_name = ? AND theatre_id = ? AND movie_name = ?", request.RoomName, theatreID, request.MovieName).Scan(&roomID)
		if err == sql.ErrNoRows {
			http.Error(w, "Movie is not screening in the mentioned theatre and room", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, fmt.Sprintf("Database error when fetching room: %v", err), http.StatusInternalServerError)
			return
		}

		bookSeats(roomID, theatreID, username, request.NoOfSeats, w)
		return
	}

	rows, err := db.DB.Query("SELECT id, room_name FROM room WHERE theatre_id = ? AND movie_name = ?", theatreID, request.MovieName)
	if err != nil {
		http.Error(w, "Database error when fetching rooms", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var rooms []models.Room
	for rows.Next() {
		var room models.Room
		err := rows.Scan(&room.ID, &room.RoomName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching room data: %v", err), http.StatusInternalServerError)
			return
		}
		rooms = append(rooms, room)
	}

	if len(rooms) == 0 {
		http.Error(w, "Movie is not screening in this theatre", http.StatusNotFound)
		return
	}

	if len(rooms) == 1 {
		bookSeats(rooms[0].ID, theatreID, username, request.NoOfSeats, w)
		return
	}

	var roomChoices []string
	for _, room := range rooms {
		roomChoices = append(roomChoices, room.RoomName)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Multiple rooms available. Choose one.",
		"room_choices": roomChoices,
	})
}

func authenticateUser(r *http.Request) (string, error) {
	cookie, err := r.Cookie("token")
	if err != nil {
		return "", fmt.Errorf("token not provided")
	}

	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
		return utils.JwtKey, nil
	})
	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	return claims.Username, nil
}

func bookSeats(roomID, theatreID int, username string, noOfSeats int, w http.ResponseWriter) {
	var seatsVacant, seatsBooked int
	err := db.DB.QueryRow("SELECT seats_vacant, seats_booked FROM room WHERE id = ?", roomID).Scan(&seatsVacant, &seatsBooked)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching seat data: %v", err), http.StatusInternalServerError)
		return
	}

	if seatsVacant < noOfSeats {
		http.Error(w, "Not enough seats available", http.StatusConflict)
		return
	}

	_, err = db.DB.Exec("UPDATE room SET seats_vacant = seats_vacant - ?, seats_booked = seats_booked + ?, updated_by = ?, updated_on = ? WHERE id = ?", noOfSeats, noOfSeats, username, time.Now(), roomID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update room details: %v", err), http.StatusInternalServerError)
		return
	}

	_, err = db.DB.Exec("UPDATE theatre SET updated_by = ?, updated_on = ? WHERE id = ?", username, time.Now(), theatreID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update theatre details: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("%d seats booked successfully\n", noOfSeats)))
}
