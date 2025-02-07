package handlers

import (
	"fmt"
	"encoding/json"
	"net/http"
	"project/db"
	"project/models"
	"time"
	"strings"
	"strconv"
)

func authenticateSystem(w http.ResponseWriter, r *http.Request) bool {
	username, password, ok := r.BasicAuth()
	if !ok || username != "System" || password != "Test@123" {	
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

func InsertTheatre(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if !authenticateSystem(w, r) {
		return
	}

	var theatre struct {
		Theatre_Name string `json:"theatre_name"`
		Total_Rooms  int    `json:"total_rooms"`
	}

	err := json.NewDecoder(r.Body).Decode(&theatre)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}

	result, err := tx.Exec(`
		INSERT INTO theatre (theatre_name, total_rooms, created_by, created_on)
		VALUES (?, ?, 'System', ?)`,
		theatre.Theatre_Name, theatre.Total_Rooms, time.Now(),
	)

	if err != nil {
		tx.Rollback()
		http.Error(w, "Failed to insert theatre", http.StatusInternalServerError)
		return
	}

	theatreID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		http.Error(w, "Failed to retrieve theatre ID", http.StatusInternalServerError)
		return
	}

	var roomIDs []int64
	for i := 0; i < theatre.Total_Rooms; i++ {
		result, err := tx.Exec(`
			INSERT INTO room (theatre_id, created_by, created_on)
			VALUES (?, 'System', ?)`,
			theatreID, time.Now(),
		)

		if err != nil {
			tx.Rollback()
			http.Error(w, "Failed to insert rooms", http.StatusInternalServerError)
			return
		}

		roomID, err := result.LastInsertId()
		if err != nil {
			tx.Rollback()
			http.Error(w, "Failed to retrieve room ID", http.StatusInternalServerError)
			return
		}

		roomIDs = append(roomIDs, roomID)
	}

	err = tx.Commit()
	if err != nil {
		http.Error(w, "Transaction commit failed", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":    "Theatre and rooms added successfully",
		"theatre_id": theatreID,
		"room_ids":   roomIDs,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func UpdateTheatre(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if !authenticateSystem(w, r) {
		return
	}

	var updateData map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&updateData)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	theatreID, ok := updateData["theatre_id"].(float64)
	if !ok {
		http.Error(w, "Theatre ID is required", http.StatusBadRequest)
		return
	}

	var oldTotalRooms int
	err = db.DB.QueryRow("SELECT total_rooms FROM theatre WHERE id = ?", int(theatreID)).Scan(&oldTotalRooms)
	if err != nil {
		http.Error(w, "Failed to fetch current total_rooms", http.StatusInternalServerError)
		return
	}

	var createdRoomIDs []int
	var deletedRoomIDs []int

	newTotalRooms, ok := updateData["total_rooms"].(float64)
	if ok && int(newTotalRooms) != oldTotalRooms {
		difference := int(newTotalRooms) - oldTotalRooms

		if difference < 0 {
			rows, err := db.DB.Query("SELECT id FROM room WHERE theatre_id = ?", int(theatreID))
			if err != nil {
				http.Error(w, "Failed to fetch room IDs", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var roomIDs []int
			for rows.Next() {
				var roomID int
				if err := rows.Scan(&roomID); err != nil {
					http.Error(w, "Failed to scan room IDs", http.StatusInternalServerError)
					return
				}
				roomIDs = append(roomIDs, roomID)
			}

			roomIDsStr := r.URL.Query().Get("room_ids")
			if roomIDsStr == "" {
				http.Error(w, "Room IDs to delete must be provided", http.StatusBadRequest)
				return
			}

			ids := strings.Split(roomIDsStr, ",")
			if len(ids) != -difference {
				http.Error(w, fmt.Sprintf("You must provide exactly %d room IDs to delete", -difference), http.StatusBadRequest)
				return
			}

			for _, idStr := range ids {
				roomID, err := strconv.Atoi(idStr)
				if err != nil {
					http.Error(w, "Invalid room ID format", http.StatusBadRequest)
					return
				}
				_, err = db.DB.Exec("DELETE FROM room WHERE id = ?", roomID)
				if err != nil {
					http.Error(w, "Failed to delete room", http.StatusInternalServerError)
					return
				}
				deletedRoomIDs = append(deletedRoomIDs, roomID)
			}
		} else if difference > 0 {
			tx, err := db.DB.Begin()
			if err != nil {
				http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
				return
			}

			for i := 0; i < difference; i++ {
				result, err := tx.Exec(`
					INSERT INTO room (theatre_id, created_by, created_on)
					VALUES (?, 'System', ?)`,
					theatreID, time.Now(),
				)
				if err != nil {
					tx.Rollback()
					http.Error(w, "Failed to add new rooms", http.StatusInternalServerError)
					return
				}

				roomID, err := result.LastInsertId()
				if err != nil {
					tx.Rollback()
					http.Error(w, "Failed to get last inserted room ID", http.StatusInternalServerError)
					return
				}
				createdRoomIDs = append(createdRoomIDs, int(roomID))
			}

			err = tx.Commit()
			if err != nil {
				http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
				return
			}
		}
	}

	query := "UPDATE theatre SET "
	params := []interface{}{}
	paramCount := 0

	for key, value := range updateData {
		if key == "theatre_id" {
			continue
		}
		query += fmt.Sprintf("%s = ?, ", key)
		params = append(params, value)
		paramCount++
	}

	if paramCount == 0 {
		http.Error(w, "No attributes to update", http.StatusBadRequest)
		return
	}

	query = query[:len(query)-2]
	query += ", updated_by = 'System', updated_on = ? WHERE id = ?"
	params = append(params, time.Now(), int(theatreID))

	_, err = db.DB.Exec(query, params...)
	if err != nil {
		http.Error(w, "Failed to update theatre", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":          "Theatre updated successfully",
		"created_room_ids": createdRoomIDs,
		"deleted_room_ids": deletedRoomIDs,
	}

	json.NewEncoder(w).Encode(response)
}

func DeleteTheatre(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if !authenticateSystem(w, r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		ID int `json:"theatre_id"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var totalBookedSeats int
	err = tx.QueryRow("SELECT COALESCE(SUM(seats_booked), 0) FROM room WHERE theatre_id = ?", req.ID).Scan(&totalBookedSeats)
	if err != nil {
		http.Error(w, "Failed to check booked seats", http.StatusInternalServerError)
		return
	}

	if totalBookedSeats > 0 {
		http.Error(w, "Cannot delete theatre with booked seats", http.StatusConflict)
		return
	}

	_, err = tx.Exec("DELETE FROM room WHERE theatre_id = ?", req.ID)
	if err != nil {
		http.Error(w, "Failed to delete related rooms", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM theatre WHERE id = ?", req.ID)
	if err != nil {
		http.Error(w, "Failed to delete theatre", http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Theatre and related rooms deleted successfully"})
}

func GetTheatreByTheatreName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if !authenticateSystem(w, r) {
		return
	}

	theatreName := r.URL.Query().Get("theatre_name")
	if theatreName == "" {
		http.Error(w, "Theatre name is required", http.StatusBadRequest)
		return
	}

	rows, err := db.DB.Query("SELECT id, theatre_name, total_rooms FROM theatre WHERE theatre_name = ?", theatreName)
	if err != nil {
		http.Error(w, "Failed to retrieve theatres", http.StatusInternalServerError)
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
			http.Error(w, "Failed to read theatre data", http.StatusInternalServerError)
			return
		}
		theatres = append(theatres, theatre)
	}

	if len(theatres) == 0 {
		http.Error(w, "No theatres found with the given theatre name", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.MarshalIndent(theatres, "", "  ")
	if err != nil {
		http.Error(w, "Failed to encode data", http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)
	w.Write([]byte("\n"))
}


func InsertMovie(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if !authenticateSystem(w, r) {
		return
	}

	var movie models.Movie
	err := json.NewDecoder(r.Body).Decode(&movie)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec(`
		INSERT INTO movie (movie_name, description, created_by, created_on) 
		VALUES (?, ?, 'System', NOW())`,
		movie.Movie_Name, movie.Description,
	)
	if err != nil {
		http.Error(w, "Failed to insert movie", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Movie added successfully"})
}

func UpdateMovie(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if !authenticateSystem(w, r) {
		return
	}

	var updateData map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&updateData)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	movieID, ok := updateData["movie_id"].(float64)
	if !ok {
		http.Error(w, "movie_id is required", http.StatusBadRequest)
		return
	}

	query := "UPDATE movie SET "
	params := []interface{}{}
	paramCount := 0

	for key, value := range updateData {
		if key == "movie_id" {
			continue
		}

		query += fmt.Sprintf("%s = ?, ", key)
		params = append(params, value)
		paramCount++
	}

	if paramCount == 0 {
		http.Error(w, "No attributes to update", http.StatusBadRequest)
		return
	}

	query += "updated_by = ?, updated_on = NOW() WHERE id = ?"
	params = append(params, "System", int(movieID))

	_, err = db.DB.Exec(query, params...)
	if err != nil {
		http.Error(w, "Failed to update movie", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Movie updated successfully"})
}

func DeleteMovie(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if !authenticateSystem(w, r) {
		return
	}

	var req struct {
		MovieID int `json:"movie_id"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec("DELETE FROM movie WHERE id = ?", req.MovieID)
	if err != nil {
		http.Error(w, "Failed to delete movie", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Movie deleted successfully"})
}

func GetMovieByMovieName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if !authenticateSystem(w, r) {
		return
	}

	movieName := r.URL.Query().Get("movie_name")
	if movieName == "" {
		http.Error(w, "Movie name is required", http.StatusBadRequest)
		return
	}

	rows, err := db.DB.Query("SELECT id, movie_name, description FROM movie WHERE LOWER(movie_name) = LOWER(?)", movieName)
	if err != nil {
		http.Error(w, "Failed to retrieve movies", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var movies []struct {
		ID          int    `json:"id"`
		Movie_Name  string `json:"movie_name"`
		Description string `json:"description"`
	}

	for rows.Next() {
		var movie struct {
			ID          int    `json:"id"`
			Movie_Name  string `json:"movie_name"`
			Description string `json:"description"`
		}
		err := rows.Scan(&movie.ID, &movie.Movie_Name, &movie.Description)
		if err != nil {
			http.Error(w, "Failed to read movie data", http.StatusInternalServerError)
			return
		}
		movies = append(movies, movie)
	}

	if len(movies) == 0 {
		http.Error(w, "No movies found with the given movie name", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.MarshalIndent(movies, "", "  ")
	if err != nil {
		http.Error(w, "Failed to encode data", http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)
	w.Write([]byte("\n"))
}

