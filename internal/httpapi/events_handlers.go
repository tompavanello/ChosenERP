package httpapi

import (
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/events"
	"chosenerp/internal/store"
)

// ---- Tipos de evento (catalogo SELADO - somente leitura, 000069) ----

func (a *App) handleListEventKinds(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []events.Kind
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Events.ListKinds(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"kinds": out})
}

// ---- Eventos ----

func (a *App) handleListEvents(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	q := r.URL.Query()
	b := boundsFromClaims(claims)
	var out []events.Event
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Events.ListEvents(r.Context(), tx, q.Get("from"), q.Get("to"), q.Get("kind"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": out})
}

func (a *App) handleCreateEvent(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in events.CreateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	if in.StartsAt == "" {
		writeErr(w, http.StatusBadRequest, "starts_at e obrigatorio")
		return
	}
	b := boundsFromClaims(claims)
	var e *events.Event
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		e, err = a.Events.CreateEvent(r.Context(), tx, claims.TenantID, branchID, claims.UserID, in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (a *App) handleGetEvent(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var e *events.Event
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		e, err = a.Events.GetEvent(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "evento nao encontrado")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (a *App) handleUpdateEvent(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in events.UpdateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	b := boundsFromClaims(claims)
	var e *events.Event
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		e, err = a.Events.UpdateEvent(r.Context(), tx, r.PathValue("id"), in)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "evento nao encontrado")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (a *App) handleDeleteEvent(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Events.DeleteEvent(r.Context(), tx, r.PathValue("id"))
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "evento nao encontrado")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- Convocados ----

func (a *App) handleListInvitees(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []events.Invitee
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Events.ListInvitees(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invitees": out})
}

func (a *App) handleSetInvitees(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		MemberIDs   []string `json:"member_ids"`
		MinistryIDs []string `json:"ministry_ids"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Events.SetInvitees(r.Context(), tx, r.PathValue("id"), in.MemberIDs, in.MinistryIDs)
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "evento nao encontrado")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- Chamada nominal ----

func (a *App) handleListEventAttendance(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []events.Attendance
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Events.ListAttendance(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"attendance": out})
}

func (a *App) handleSaveEventAttendance(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		ParticipantsCount int      `json:"participants_count"`
		PresentMemberIDs  []string `json:"present_member_ids"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Events.SaveAttendance(r.Context(), tx, r.PathValue("id"), in.ParticipantsCount, in.PresentMemberIDs)
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "evento nao encontrado")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- Frequencia do membro ----

func (a *App) handleListFrequency(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []events.Frequency
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Events.ListFrequency(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"frequency": out})
}

func (a *App) handleSetFrequency(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Frequency string `json:"frequency"`
		StartedAt string `json:"started_at"`
		Notes     string `json:"notes"`
	}
	if err := readJSON(r, &in); err != nil || in.Frequency == "" {
		writeErr(w, http.StatusBadRequest, "frequency e obrigatorio")
		return
	}
	if in.Frequency != "frequente" && in.Frequency != "pouco_frequente" && in.Frequency != "nao_frequente" {
		writeErr(w, http.StatusBadRequest, "frequencia invalida")
		return
	}
	b := boundsFromClaims(claims)
	var f *events.Frequency
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		if _, err := a.Members.Get(r.Context(), tx, r.PathValue("id")); err != nil {
			return err
		}
		var err error
		f, err = a.Events.SetFrequency(r.Context(), tx, r.PathValue("id"), in.Frequency, in.StartedAt, in.Notes, claims.UserID)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "membro nao encontrado")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

// handleEventRSVP lista as confirmacoes de presenca (RSVP) de um evento.
func (a *App) handleEventRSVP(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	eventID := r.PathValue("id")
	var res *events.RSVPResult
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		res, e = a.Events.RSVPsOfEvent(r.Context(), tx, eventID)
		return e
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}
