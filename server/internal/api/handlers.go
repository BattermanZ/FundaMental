package api

import (
	"encoding/json"
	"fundamental/server/internal/database"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	db     *database.Database
	logger *logrus.Logger
}

func NewHandler(db *database.Database, logger *logrus.Logger) *Handler {
	return &Handler{
		db:     db,
		logger: logger,
	}
}

func (h *Handler) GetAllProperties(w http.ResponseWriter, r *http.Request) {
	properties, err := h.db.GetAllProperties()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get properties")
		http.Error(w, "Failed to get properties", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(properties)
}

func (h *Handler) GetPropertyStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.db.GetPropertyStats()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get property stats")
		http.Error(w, "Failed to get property stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *Handler) GetAreaStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postalPrefix := vars["postal_prefix"]

	stats, err := h.db.GetAreaStats(postalPrefix)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get area stats")
		http.Error(w, "Failed to get area stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *Handler) GetRecentSales(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default limit
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	sales, err := h.db.GetRecentSales(limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get recent sales")
		http.Error(w, "Failed to get recent sales", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sales)
}

// CORS middleware
func (h *Handler) EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
