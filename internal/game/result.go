package game

type Result struct {
	Revision uint64   `json:"revision"`
	Snapshot Snapshot `json:"snapshot"`
	Events   []Event  `json:"events"`
}
