package api

// Snek defines the interactions of a Battlesnake with the API.
type Snek interface {
	Start(*State) error
	Move(*State) (string, string, error)
	End(*State) error
}

// State defines the game state on a given turn.
//
// Everything else in this file is wireformat types explained at
// https://docs.battlesnake.com/references/api.
type State struct {
	Game  Game
	Turn  int
	Board Board
	Me    Battlesnake
}

type InfoResponse struct {
	APIVersion string `json:"apiversion"`
	Author     string `json:"author,omitempty"`
	Color      string `json:"color,omitempty"`
	Head       string `json:"head,omitempty"`
	Tail       string `json:"tail,omitempty"`
	Version    string `json:"version,omitempty"`
}

type GameRequest struct {
	Game  Game        `json:"game"`
	Turn  int         `json:"turn"`
	Board Board       `json:"board"`
	You   Battlesnake `json:"you"`
}

type MoveResponse struct {
	Move  string `json:"move"`
	Shout string `json:"shout"`
}

type Game struct {
	ID      string                 `json:"id"`
	Ruleset map[string]interface{} `json:"ruleset"`
	Timeout int                    `json:"timeout"`
}

type Battlesnake struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Health  int     `json:"health"`
	Body    []Point `json:"body"`
	Latency string  `json:"latency"`
	Head    Point   `json:"head"`
	Length  int     `json:"length"`
	Shout   string  `json:"shout"`
	Squad   string  `json:"squad"`
}

type Board struct {
	Height  int           `json:"height"`
	Width   int           `json:"width"`
	Food    []Point       `json:"food"`
	Hazards []Point       `json:"hazards"`
	Snakes  []Battlesnake `json:"snakes"`
}

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}
