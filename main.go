package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/cmars/snekfarm/api"
	"github.com/cmars/snekfarm/lucky"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Mount("/lucky", api.Router(lucky.New))
	r.Mount("/luckydocile", api.Router(lucky.NewDocile))

	log.Fatal(http.ListenAndServe(":3000", r))
}
