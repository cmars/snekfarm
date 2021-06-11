package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
)

func Router(newSnek func() Snek) http.Handler {
	r := chi.NewRouter()
	h := &handler{newSnek: newSnek, gameSneks: map[string]Snek{}}
	r.Get("/", h.Info)
	r.Post("/start", h.Start)
	r.Post("/move", h.Move)
	r.Post("/end", h.End)
	return r
}

type handler struct {
	newSnek func() Snek

	mu        sync.RWMutex
	gameSneks map[string]Snek
}

func (*handler) Info(w http.ResponseWriter, r *http.Request) {
	err := json.NewEncoder(w).Encode(&InfoResponse{
		APIVersion: "1",
	})
	if err != nil {
		log.Printf("failed to write response: %v", err)
	}
}

func (h *handler) Start(w http.ResponseWriter, r *http.Request) {
	var startReq GameRequest
	err := json.NewDecoder(r.Body).Decode(&startReq)
	if err != nil {
		log.Printf("failed to decode request: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	gameSnek := h.newSnek()
	h.mu.Lock()
	h.gameSneks[startReq.Game.ID+startReq.You.ID] = gameSnek
	h.mu.Unlock()

	err = gameSnek.Start(&State{
		Game:  startReq.Game,
		Turn:  startReq.Turn,
		Board: startReq.Board,
		Me:    startReq.You,
	})
	if err != nil {
		http.Error(w, "snek cannot start game", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *handler) Move(w http.ResponseWriter, r *http.Request) {
	var moveReq GameRequest
	err := json.NewDecoder(r.Body).Decode(&moveReq)
	if err != nil {
		log.Printf("failed to decode request: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	h.mu.RLock()
	gameSnek, ok := h.gameSneks[moveReq.Game.ID+moveReq.You.ID]
	h.mu.RUnlock()
	if !ok {
		http.Error(w, "game not started", http.StatusBadRequest)
		return
	}

	move, shout, err := gameSnek.Move(&State{
		Game:  moveReq.Game,
		Turn:  moveReq.Turn,
		Board: moveReq.Board,
		Me:    moveReq.You,
	})
	if err != nil {
		log.Printf("snek cannot move: %v", err)
		http.Error(w, "snek cannot move", http.StatusBadRequest)
		return
	}

	err = json.NewEncoder(w).Encode(&MoveResponse{
		Move:  move,
		Shout: shout,
	})
	if err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func (h *handler) End(w http.ResponseWriter, r *http.Request) {
	var endReq GameRequest
	err := json.NewDecoder(r.Body).Decode(&endReq)
	if err != nil {
		log.Printf("failed to decode request: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	gameSnek, ok := h.gameSneks[endReq.Game.ID+endReq.You.ID]
	delete(h.gameSneks, endReq.Game.ID+endReq.You.ID)
	h.mu.Unlock()
	if !ok {
		http.Error(w, "game not started", http.StatusBadRequest)
		return
	}

	err = gameSnek.End(&State{
		Game:  endReq.Game,
		Turn:  endReq.Turn,
		Board: endReq.Board,
		Me:    endReq.You,
	})
	if err != nil {
		http.Error(w, "snek cannot end game", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
