package api

import (
    "net/http"
)

func NewRouter(h *Handler) http.Handler {
    mux := http.NewServeMux()

    // /buckets — GET и POST
    mux.HandleFunc("/buckets", func(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case http.MethodGet:
            h.handleListBuckets().ServeHTTP(w, r)
        case http.MethodPost:
            h.handleCreateBucket().ServeHTTP(w, r)
        default:
            http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
        }
    })

    // /buckets/{id} — GET, PUT, PATCH, DELETE
    mux.HandleFunc("/buckets/", func(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case http.MethodGet:
            h.handleGetBucket().ServeHTTP(w, r)
        case http.MethodPut:
            h.handleUpdateCapacity().ServeHTTP(w, r)
        case http.MethodPatch:
            h.handleUpdateTokens().ServeHTTP(w, r)
        case http.MethodDelete:
            h.handleDeleteBucket().ServeHTTP(w, r)
        default:
            http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
        }
    })

    return mux
}