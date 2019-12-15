package calendar

import (
    "fmt"
)

type Calendar struct {
    Name string
    IsPrivate bool
}

func (c Calendar) String() string {
    return fmt.Sprintf("%s(%v)", c.Name, c.IsPrivate)
}