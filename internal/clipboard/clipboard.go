package clipboard

import (
	"os"

	"github.com/aymanbagabas/go-osc52/v2"
)

func Set(s string) {
	osc52.New(s).WriteTo(os.Stderr)
}
