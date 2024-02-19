package errorHandling

import "fmt"

func CatchInputError(n int, err error) *string {
	if err != nil {
		result := fmt.Sprintf("Error: %d (%v)", n, err)
		return &result
	}
	return nil
}
