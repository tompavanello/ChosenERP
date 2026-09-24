package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/auth"
	"chosenerp/internal/kids"
	"chosenerp/internal/store"
)

// kidsDo executa a operacao no escopo RLS da sessao e serializa o resultado,
// mapeando erros de validacao/not-found para os status corretos.
func (a *App) kidsDo(w http.ResponseWriter, r *http.Request, status int,
	fn func(ctx context.Context, tx pgx.Tx, claims *auth.Claims) (any, error)) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out any
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		out, e = fn(r.Context(), tx, claims)
		return e
	})
	if err != nil {
		switch {
		case store.IsNotFound(err):
			writeErr(w, http.StatusNotFound, "not found")
		case errors.Is(err, kids.ErrInvalidInput):
			writeErr(w, http.StatusBadRequest, err.Error())
		default:
			writeErr(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, status, out)
}

// ---- Trilhas / conteudo ----

func (a *App) handleListKidsTracks(w http.ResponseWriter, r *http.Request) {
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		rows, err := a.Kids.ListTracks(ctx, tx)
		return map[string]any{"tracks": rows}, err
	})
}

func (a *App) handleCreateKidsTrack(w http.ResponseWriter, r *http.Request) {
	var in kids.TrackInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	a.kidsDo(w, r, http.StatusCreated, func(ctx context.Context, tx pgx.Tx, c *auth.Claims) (any, error) {
		return a.Kids.CreateTrack(ctx, tx, c.TenantID, c.BranchID, in)
	})
}

func (a *App) handleUpdateKidsTrack(w http.ResponseWriter, r *http.Request) {
	var in kids.TrackInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	id := r.PathValue("id")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		return a.Kids.UpdateTrack(ctx, tx, id, in)
	})
}

func (a *App) handleDeleteKidsTrack(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		return map[string]any{"ok": true}, a.Kids.DeleteTrack(ctx, tx, id)
	})
}

func (a *App) handleListKidsLessons(w http.ResponseWriter, r *http.Request) {
	trackID := r.PathValue("id")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		rows, err := a.Kids.ListLessons(ctx, tx, trackID)
		return map[string]any{"lessons": rows}, err
	})
}

func (a *App) handleCreateKidsLesson(w http.ResponseWriter, r *http.Request) {
	var in kids.LessonInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	trackID := r.PathValue("id")
	a.kidsDo(w, r, http.StatusCreated, func(ctx context.Context, tx pgx.Tx, c *auth.Claims) (any, error) {
		return a.Kids.CreateLesson(ctx, tx, c.TenantID, c.BranchID, trackID, in)
	})
}

func (a *App) handleUpdateKidsLesson(w http.ResponseWriter, r *http.Request) {
	var in kids.LessonInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	id := r.PathValue("id")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		return a.Kids.UpdateLesson(ctx, tx, id, in)
	})
}

func (a *App) handleDeleteKidsLesson(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		return map[string]any{"ok": true}, a.Kids.DeleteLesson(ctx, tx, id)
	})
}

// ---- Turmas ----

func (a *App) handleListKidsClasses(w http.ResponseWriter, r *http.Request) {
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		rows, err := a.Kids.ListClasses(ctx, tx)
		return map[string]any{"classes": rows}, err
	})
}

func (a *App) handleCreateKidsClass(w http.ResponseWriter, r *http.Request) {
	var in kids.ClassInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	a.kidsDo(w, r, http.StatusCreated, func(ctx context.Context, tx pgx.Tx, c *auth.Claims) (any, error) {
		return a.Kids.CreateClass(ctx, tx, c.TenantID, c.BranchID, in)
	})
}

func (a *App) handleUpdateKidsClass(w http.ResponseWriter, r *http.Request) {
	var in kids.ClassInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	id := r.PathValue("id")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		return a.Kids.UpdateClass(ctx, tx, id, in)
	})
}

func (a *App) handleDeleteKidsClass(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		return map[string]any{"ok": true}, a.Kids.DeleteClass(ctx, tx, id)
	})
}

// ---- Matriculas / responsaveis ----

func (a *App) handleListKidsEnrollments(w http.ResponseWriter, r *http.Request) {
	classID := r.PathValue("id")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		rows, err := a.Kids.ListEnrollments(ctx, tx, classID)
		return map[string]any{"enrollments": rows}, err
	})
}

func (a *App) handleCreateKidsEnrollment(w http.ResponseWriter, r *http.Request) {
	var in kids.EnrollmentInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	classID := r.PathValue("id")
	a.kidsDo(w, r, http.StatusCreated, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		return a.Kids.CreateEnrollment(ctx, tx, classID, in)
	})
}

func (a *App) handleUpdateKidsEnrollment(w http.ResponseWriter, r *http.Request) {
	var in kids.EnrollmentInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	id := r.PathValue("id")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		return a.Kids.UpdateEnrollment(ctx, tx, id, in)
	})
}

func (a *App) handleDeleteKidsEnrollment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		return map[string]any{"ok": true}, a.Kids.DeleteEnrollment(ctx, tx, id)
	})
}

func (a *App) handleListKidsGuardians(w http.ResponseWriter, r *http.Request) {
	enrollmentID := r.PathValue("id")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		rows, err := a.Kids.ListGuardians(ctx, tx, enrollmentID)
		return map[string]any{"guardians": rows}, err
	})
}

func (a *App) handleAddKidsGuardian(w http.ResponseWriter, r *http.Request) {
	var in kids.GuardianInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	enrollmentID := r.PathValue("id")
	a.kidsDo(w, r, http.StatusCreated, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		return a.Kids.AddGuardian(ctx, tx, enrollmentID, in)
	})
}

func (a *App) handleDeleteKidsGuardian(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		return map[string]any{"ok": true}, a.Kids.DeleteGuardian(ctx, tx, id)
	})
}

// ---- Encontros ----

func (a *App) handleListKidsSessions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	classID, from, to := q.Get("class_id"), q.Get("from"), q.Get("to")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		rows, err := a.Kids.ListSessions(ctx, tx, classID, from, to)
		return map[string]any{"sessions": rows}, err
	})
}

func (a *App) handleCreateKidsSession(w http.ResponseWriter, r *http.Request) {
	var in kids.SessionInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	a.kidsDo(w, r, http.StatusCreated, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		return a.Kids.CreateSession(ctx, tx, in)
	})
}

func (a *App) handleUpdateKidsSession(w http.ResponseWriter, r *http.Request) {
	var in kids.SessionInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	id := r.PathValue("id")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		return a.Kids.UpdateSession(ctx, tx, id, in)
	})
}

func (a *App) handleDeleteKidsSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		return map[string]any{"ok": true}, a.Kids.DeleteSession(ctx, tx, id)
	})
}

// ---- Check-in / check-out ----

func (a *App) handleKidsRoster(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		rows, err := a.Kids.Roster(ctx, tx, sessionID)
		return map[string]any{"roster": rows}, err
	})
}

func (a *App) handleKidsCheckin(w http.ResponseWriter, r *http.Request) {
	var in kids.CheckinInput
	if err := readJSON(r, &in); err != nil || in.EnrollmentID == "" {
		writeErr(w, http.StatusBadRequest, "enrollment_id required")
		return
	}
	sessionID := r.PathValue("id")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		return a.Kids.CheckIn(ctx, tx, sessionID, in)
	})
}

func (a *App) handleKidsCheckout(w http.ResponseWriter, r *http.Request) {
	var in struct {
		EnrollmentID     string `json:"enrollment_id"`
		SecurityCode     string `json:"security_code"`
		PickupGuardianID string `json:"pickup_guardian_id"`
	}
	if err := readJSON(r, &in); err != nil || in.EnrollmentID == "" {
		writeErr(w, http.StatusBadRequest, "enrollment_id required")
		return
	}
	sessionID := r.PathValue("id")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		return a.Kids.CheckOut(ctx, tx, sessionID, in.EnrollmentID, in.SecurityCode, in.PickupGuardianID)
	})
}

func (a *App) handleKidsAbsence(w http.ResponseWriter, r *http.Request) {
	var in struct {
		EnrollmentID string `json:"enrollment_id"`
		Absent       *bool  `json:"absent"`
	}
	if err := readJSON(r, &in); err != nil || in.EnrollmentID == "" {
		writeErr(w, http.StatusBadRequest, "enrollment_id required")
		return
	}
	sessionID := r.PathValue("id")
	absent := in.Absent == nil || *in.Absent
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		return map[string]any{"ok": true}, a.Kids.MarkAbsent(ctx, tx, sessionID, in.EnrollmentID, absent)
	})
}

// ---- Relatorio de evolucao ----

func (a *App) handleKidsEvolution(w http.ResponseWriter, r *http.Request) {
	classID := r.PathValue("id")
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	a.kidsDo(w, r, http.StatusOK, func(ctx context.Context, tx pgx.Tx, _ *auth.Claims) (any, error) {
		return a.Kids.Evolution(ctx, tx, classID, from, to)
	})
}
