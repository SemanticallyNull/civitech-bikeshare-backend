package station

import (
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Type int

const (
	Public Type = iota
	Private
)

type Station struct {
	ID           uuid.UUID
	Name         string
	Address      string
	OpeningHours string `db:"opening_hours"`
	Location     pgtype.Point
	Type         Type
}

func (t Type) String() string {
	return [...]string{"public", "private"}[t]
}

func (t Type) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *Type) Scan(i any) error {
	switch v := i.(type) {
	case string:
		switch v {
		case "public":
			*t = Public
			return nil
		case "private":
			*t = Private
			return nil
		}
	}
	panic("invalid scan type")
}
