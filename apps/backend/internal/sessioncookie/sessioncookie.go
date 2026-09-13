// Builds and clears the host-only cookies that carry server-managed auth sessions.
package sessioncookie

import (
	"net/http"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
)

const (
	TeacherName         = "__Host-nutka_teacher_session"
	LearnerName         = "__Host-nutka_learner_session"
	AuthRefreshDisabled = "auth_refresh_disabled"
)

func Build(name, token string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(authconfig.SessionDuration.Seconds()),
	}
}

func Clear(name string) *http.Cookie {
	cookie := Build(name, "")
	cookie.MaxAge = -1
	return cookie
}

// InjectingWriter adds a login cookie only for successful auth responses.
type InjectingWriter struct {
	http.ResponseWriter
	Cookie      *http.Cookie
	wroteHeader bool
}

func (w *InjectingWriter) WriteHeader(status int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		if status >= http.StatusOK && status < http.StatusMultipleChoices && w.Cookie != nil {
			http.SetCookie(w.ResponseWriter, w.Cookie)
		}
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *InjectingWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *InjectingWriter) Flush() {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *InjectingWriter) Written() bool {
	if tracker, ok := w.ResponseWriter.(interface{ Written() bool }); ok {
		return tracker.Written()
	}
	return w.wroteHeader
}

func (w *InjectingWriter) Status() int {
	if tracker, ok := w.ResponseWriter.(interface{ Status() int }); ok {
		return tracker.Status()
	}
	return 0
}

func (w *InjectingWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
