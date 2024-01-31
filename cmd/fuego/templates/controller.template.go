package templates

import (
	"github.com/go-fuego/fuego"
)

type newControllerRessources struct {
	// TODO add ressourses
}

type NewControllerRepository interface {
	// TODO add queries
}

func (rs newControllerRessources) Routes(s *fuego.Server) {
	newControllerRoutesGroupe := fuego.Group(s, "/new-controller")
	fuego.Get(newControllerRoutesGroupe, "/", rs.newControllerFunction)
}

// TODO replace the following with your own function
func (rs newControllerRessources) newControllerFunction(c fuego.Ctx[any]) (string, error) {
	return "🔥", nil
}
