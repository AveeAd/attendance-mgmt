package handlers

import (
	"net"
	"net/http"

	"attendance-mgmt/backend/internal/version"
)

type serverInfoResponse struct {
	Addresses []string `json:"addresses"`
	Port      string   `json:"port"`
	Version   string   `json:"version"`
}

// ServerInfo handles GET /api/server-info (public — no login required, so
// the QR check-in page can be displayed/printed without signing in first).
// Returns the machine's non-loopback local IPv4 addresses, so the QR code
// page can suggest a real LAN URL instead of "localhost" (which only
// resolves on the laptop itself, not on staff phones over WiFi).
func ServerInfo(port string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, serverInfoResponse{
			Addresses: localIPv4Addresses(),
			Port:      port,
			Version:   version.Version,
		})
	}
}

func localIPv4Addresses() []string {
	addrs := []string{}

	ifaces, err := net.InterfaceAddrs()
	if err != nil {
		return addrs
	}

	for _, addr := range ifaces {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		ip4 := ipNet.IP.To4()
		if ip4 == nil {
			continue
		}
		addrs = append(addrs, ip4.String())
	}

	return addrs
}
