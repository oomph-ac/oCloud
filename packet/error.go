package packet

import "fmt"

var (
	ErrEndOfBatchRead = fmt.Errorf("no pending batches remaining to read")
)
