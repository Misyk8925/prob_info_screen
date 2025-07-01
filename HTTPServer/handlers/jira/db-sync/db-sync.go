package db_sync

import (
	"gorm.io/gorm"
	"log"
	"net/http"
	"prob_info_screen/storage"
)

func New(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := storage.SyncProjectDB(db, "TKP"); err != nil {
			log.Fatalf("Синхронизация провалилась: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := storage.SyncProjectDB(db, "TKP"); err != nil {
			http.Error(w, "Failed to sync project database: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte("Project database synced successfully"))
		log.Println("Project database synced successfully")

	}
}
