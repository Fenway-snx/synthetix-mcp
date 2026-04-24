package repository

import "fmt"

type OrderFilter string

const (
	_createdAt_PlaceHolder = "created_at %s"

	OrderFilter_ASC  = "ASC"
	OrderFilter_DESC = "DESC"
)

var (
	CreatedAt_ASC  = fmt.Sprintf(_createdAt_PlaceHolder, OrderFilter_ASC)
	CreatedAt_DESC = fmt.Sprintf(_createdAt_PlaceHolder, OrderFilter_DESC)
)
