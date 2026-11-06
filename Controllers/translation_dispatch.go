package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/internal/translations"
	"ArmadaCMS/main/utils"
	"context"
	"net/http"
)

// DispatchTranslationJobs checks for runnable work without creating a Cloud Run
// execution for an empty queue. The route requires Google's scheduler identity.
// @Summary Dispatch runnable translation work
// @Tags translations
// @Security SchedulerOIDC
// @Success 202 "Worker start requested"
// @Success 204 "No runnable work or generation disabled"
// @Failure 401 {string} string "Google ID token required"
// @Failure 403 {string} string "Unexpected service account"
// @Failure 503 {string} string "Dispatcher unavailable"
// @Router /internal/translations/dispatch [post]
func DispatchTranslationJobs(w http.ResponseWriter, r *http.Request) {
	dispatchTranslationJobs(w, r, translations.New(db.DB), utils.LaunchTranslationWorker)
}
func dispatchTranslationJobs(w http.ResponseWriter, r *http.Request, s *translations.Service, launch func(context.Context) error) {
	started, err := s.Dispatch(r.Context(), launch)
	if err != nil {
		http.Error(w, "Translation dispatcher unavailable", http.StatusServiceUnavailable)
		return
	}
	if started {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
