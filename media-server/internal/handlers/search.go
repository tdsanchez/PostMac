package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/tdsanchez/PostMac/internal/models"
	"github.com/tdsanchez/PostMac/internal/search"
	"github.com/tdsanchez/PostMac/internal/state"
)

// HandleSearchQuery executes a search query and returns file results
// Used by synthetic categories
func HandleSearchQuery(query string) ([]models.FileInfo, error) {
	// Parse the query
	queryNode, err := search.Parse(query)
	if err != nil {
		return nil, err
	}

	// Get current state (lock-free read)
	current := state.GetCurrent()
	filesByTag := current.FilesByTag

	// Execute the query
	results := queryNode.Evaluate(filesByTag)

	return results, nil
}

// HandleSearch executes a boolean search query
func HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	// Execute search using shared function
	results, err := HandleSearchQuery(query)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
			"query": query,
		})
		return
	}

	// Return results as JSON
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"files": results,
		"count": len(results),
		"query": query,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding search results: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
