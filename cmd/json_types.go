package cmd

// actionForJSON is a struct used for marshaling an action to JSON for machine-readable output.
type actionForJSON struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Details     []string `json:"details"`
}
