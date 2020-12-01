package api

import (
	"github.com/ut8ia/shortener/pkg/version"
	"net/http"
)

// Version godoc
// @Summary Version
// @Description returns podinfo version and git commit hash
// @Tags HTTP API
// @Produce json
// @Router /version [get]
// @Success 200 {object} api.MapResponse
func (s *Server) versionHandler(w http.ResponseWriter, r *http.Request) {
	result := map[string]string{
		"version": version.VERSION,
		"commit":  version.REVISION,
	}
	s.JSONResponse(w, r, result)
}
