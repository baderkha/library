package err

import "fmt"

func Compose(er string) func(arg ...any) error {
	return func(args ...any) error {
		return fmt.Errorf(er, args...)
	}
}
