package tier

type Id string

const (
	Id_Zero Id = ""
)

type (
	Name string
	Type string
)

const (
	Type_custom Type = "custom"
	Type_volume Type = "volume"
)
