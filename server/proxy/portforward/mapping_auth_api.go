package portforward

import "net/http"

func handlePortMappingAuth(w http.ResponseWriter, r *http.Request) {
	handleMappingLegacyConfig(w, r, true)
}
