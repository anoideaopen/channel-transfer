package healthcheck

import (
	"net/http"

	"go.uber.org/atomic"
)

type HealthCheck struct {
	isReady atomic.Bool
}

func (hc *HealthCheck) Ready() {
	hc.isReady.Store(true)
}

func (hc *HealthCheck) LivenessProbe(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (hc *HealthCheck) ReadinessProbe(w http.ResponseWriter, _ *http.Request) {
	if !hc.isReady.Load() {
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func New() *HealthCheck {
	return &HealthCheck{}
}
