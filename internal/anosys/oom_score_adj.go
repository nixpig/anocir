package anosys

import (
	"fmt"
	"os"
	"strconv"
)

func AdjustOOMScore(oomScoreAdj int) error {
	if err := os.WriteFile(
		"/proc/self/oom_score_adj",
		[]byte(strconv.Itoa(oomScoreAdj)),
		0644,
	); err != nil {
		return fmt.Errorf("create oom score adj file: %w", err)
	}

	return nil
}
