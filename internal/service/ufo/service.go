package ufo

type UFO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// New creates a UFO service entity.
func NewService(id, name string) UFO {
	return UFO{
		ID:   id,
		Name: name,
	}
}

func (u UFO) GetIDService() string {
	return u.ID
}

func (u UFO) GetNameService() string {
	return u.Name
}
