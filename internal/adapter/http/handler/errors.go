package handler

import "net/http"

func errorResponse(w http.ResponseWriter, status int, message any) {
	env := envelope{"error": message}

	// Write the response using the writeJSON() helper. If this happens to return an
	// error then log it, and fall back to sending the client an empty response with a
	// 500 Internal Server Error status code.
	if err := writeJSON(w, status, env, nil); err != nil {
		w.WriteHeader(500)
	}
}

func failedValidationResponse(w http.ResponseWriter, errors map[string]string) {
	errorResponse(w, http.StatusUnprocessableEntity, errors)
}

func internalErrorResponse(w http.ResponseWriter, message any) {
	errorResponse(w, http.StatusInternalServerError, message)
}
