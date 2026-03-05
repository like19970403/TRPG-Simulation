package realtime

import (
	"encoding/json"
	"log/slog"
)

// mustMarshal marshals v to JSON. If marshaling fails (which should never happen
// for the simple maps/structs used in this package), it logs the error and returns
// the JSON null literal to avoid propagating broken data.
func mustMarshal(logger *slog.Logger, v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		logger.Error("json marshal failed", "error", err)
		return []byte("null")
	}
	return data
}
