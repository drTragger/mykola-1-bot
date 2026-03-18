package utils

type Torrent struct {
	Hash       string  `json:"hash"`
	Name       string  `json:"name"`
	Progress   float64 `json:"progress"`
	State      string  `json:"state"`
	Dlspeed    int64   `json:"dlspeed"`
	Upspeed    int64   `json:"upspeed"`
	Size       int64   `json:"size"`
	TotalSize  int64   `json:"total_size"`
	Eta        int64   `json:"eta"`
	NumSeeds   int64   `json:"num_seeds"`
	NumLeechs  int64   `json:"num_leechs"`
	Downloaded int64   `json:"downloaded"`
	Uploaded   int64   `json:"uploaded"`
}
