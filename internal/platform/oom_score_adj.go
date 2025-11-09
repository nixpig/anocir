package platform

import (
	"fmt"
	"os"
	"strconv"
)

// AdjustOOMScore adjusts the OOM score of the current (container) process.
func AdjustOOMScore(oomScoreAdj int) error {
	if err := os.WriteFile(
		"/proc/self/oom_score_adj",
		[]byte(strconv.Itoa(oomScoreAdj)),
		0o644,
	); err != nil {
		return fmt.Errorf("create oom score adj file: %w", err)
	}

	return nil
}
