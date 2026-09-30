package services

import (
	"log"

	"linkforge-url-shortener/internal/store"
)

func StartClickWorkers(count int) {
	for i := 0; i < count; i++ {
		go func(workerID int) {
			for event := range ClickQueue {
				err := store.SaveClick(event)
				if err != nil {
					log.Printf("worker %d failed to save click: %v", workerID, err)
				}
			}
		}(i + 1)
	}
}
