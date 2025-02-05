package handlers

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "time"

    "github.com/golang-jwt/jwt/v4"
    "project/db"
    "project/models"
    "project/utils"
)

func Login(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    var credentials struct {
        Username string `json:"username"`
        Password string `json:"password"`
    }

    err := json.NewDecoder(r.Body).Decode(&credentials)
    if err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    var storedPassword string
    err = db.DB.QueryRow("SELECT password FROM credentials WHERE username = ?", credentials.Username).Scan(&storedPassword)
    if err == sql.ErrNoRows {
        http.Error(w, "Invalid username or password", http.StatusUnauthorized)
        return
    } else if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    if credentials.Password != storedPassword {
        http.Error(w, "Invalid username or password", http.StatusUnauthorized)
        return
    }

    expirationTime := time.Now().Add(5 * time.Minute)
    claims := &models.Claims{
        Username: credentials.Username,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(expirationTime),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString(utils.JwtKey)
    if err != nil {
        http.Error(w, "Failed to generate token", http.StatusInternalServerError)
        return
    }

    _, err = db.DB.Exec("UPDATE credentials SET recent_login = CURRENT_TIMESTAMP WHERE username = ?", credentials.Username)
    if err != nil {
        http.Error(w, "Failed to update login timestamp", http.StatusInternalServerError)
        return
    }

    http.SetCookie(w, &http.Cookie{
        Name:    "token",
        Value:   tokenString,
        Expires: expirationTime,
    })
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Login successful\n"))
}

func Authenticate(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        cookie, err := r.Cookie("token")
        if err != nil {
            http.Error(w, "Unauthorized: Token not provided", http.StatusUnauthorized)
            return
        }

        claims := &models.Claims{}
        token, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
            return utils.JwtKey, nil
        })
        if err != nil || !token.Valid {
            http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
            return
        }

        next(w, r)
    }
}
