package httpapi

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/governance"
	"chosenerp/internal/store"
)

// ---- Atas (G1/G4) ----

func (a *App) handleListMinutes(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	q := r.URL.Query()
	b := boundsFromClaims(claims)
	var out []governance.Minute
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Governance.ListMinutes(r.Context(), tx, q.Get("from"), q.Get("to"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"minutes": out})
}

func (a *App) handleCreateMinute(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in governance.MinuteInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	if in.Title == "" || in.MeetingAt == "" {
		writeErr(w, http.StatusBadRequest, "title e meeting_at sao obrigatorios")
		return
	}
	b := boundsFromClaims(claims)
	var m *governance.Minute
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		m, err = a.Governance.CreateMinute(r.Context(), tx, claims.TenantID, branchID, claims.UserID, in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (a *App) handleGetMinute(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var m *governance.Minute
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		m, err = a.Governance.GetMinute(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "ata nao encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (a *App) handleUpdateMinute(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in governance.MinuteInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	b := boundsFromClaims(claims)
	var m *governance.Minute
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		m, err = a.Governance.UpdateMinute(r.Context(), tx, r.PathValue("id"), in)
		return err
	})
	if err != nil {
		switch {
		case errors.Is(err, governance.ErrMinuteSigned):
			writeErr(w, http.StatusConflict, err.Error())
		case store.IsNotFound(err):
			writeErr(w, http.StatusNotFound, "ata nao encontrada")
		default:
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (a *App) handleDeleteMinute(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Governance.DeleteMinute(r.Context(), tx, r.PathValue("id"))
	})
	if err != nil {
		switch {
		case errors.Is(err, governance.ErrMinuteSigned):
			writeErr(w, http.StatusConflict, err.Error())
		case store.IsNotFound(err):
			writeErr(w, http.StatusNotFound, "ata nao encontrada")
		default:
			writeErr(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handleSignMinute(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	prof, err := a.Auth.Me(r.Context(), claims.UserID, claims.TenantID, claims.BranchID, claims.Role)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "profile unavailable")
		return
	}
	b := boundsFromClaims(claims)
	var sig *governance.Signature
	err = a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		sig, err = a.Governance.SignMinute(r.Context(), tx, r.PathValue("id"), claims.UserID, prof.FullName, claims.Role)
		return err
	})
	if err != nil {
		switch {
		case errors.Is(err, governance.ErrMinuteSigned):
			writeErr(w, http.StatusConflict, err.Error())
		case store.IsNotFound(err):
			writeErr(w, http.StatusNotFound, "ata nao encontrada")
		default:
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusCreated, sig)
}

func (a *App) handleListSignatures(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []governance.Signature
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Governance.ListSignatures(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"signatures": out})
}

// ---- Votacoes (G2/G3) ----

func (a *App) handleListVotes(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []governance.Vote
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Governance.ListVotes(r.Context(), tx, r.URL.Query().Get("status"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"votes": out})
}

func (a *App) handleCreateVote(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in governance.VoteInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	if in.Title == "" {
		writeErr(w, http.StatusBadRequest, "title e obrigatorio")
		return
	}
	b := boundsFromClaims(claims)
	var v *governance.Vote
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		v, err = a.Governance.CreateVote(r.Context(), tx, claims.TenantID, branchID, claims.UserID, in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (a *App) handleGetVote(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var v *governance.Vote
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		v, err = a.Governance.GetVote(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "votacao nao encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (a *App) handleUpdateVote(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in governance.VoteInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	b := boundsFromClaims(claims)
	var v *governance.Vote
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		v, err = a.Governance.UpdateVote(r.Context(), tx, r.PathValue("id"), in)
		return err
	})
	if err != nil {
		switch {
		case errors.Is(err, governance.ErrVoteClosed):
			writeErr(w, http.StatusConflict, err.Error())
		case store.IsNotFound(err):
			writeErr(w, http.StatusNotFound, "votacao nao encontrada")
		default:
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (a *App) handleDeleteVote(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Governance.DeleteVote(r.Context(), tx, r.PathValue("id"))
	})
	if err != nil {
		if errors.Is(err, governance.ErrVoteClosed) {
			writeErr(w, http.StatusConflict, "so e possivel excluir uma votacao em rascunho")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handleOpenVote(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var v *governance.Vote
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		v, err = a.Governance.OpenVote(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		if errors.Is(err, governance.ErrVoteClosed) || store.IsNotFound(err) {
			writeErr(w, http.StatusConflict, "a votacao nao esta em rascunho")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (a *App) handleCloseVote(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var v *governance.Vote
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		v, err = a.Governance.CloseVote(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "votacao nao encontrada")
			return
		}
		if errors.Is(err, governance.ErrVoteClosed) {
			writeErr(w, http.StatusConflict, "votacao cancelada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (a *App) handleCastBallot(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		OptionID string `json:"option_id"`
	}
	if err := readJSON(r, &in); err != nil || in.OptionID == "" {
		writeErr(w, http.StatusBadRequest, "option_id e obrigatorio")
		return
	}
	b := boundsFromClaims(claims)
	var res *governance.Result
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		res, err = a.Governance.CastBallot(r.Context(), tx, r.PathValue("id"), claims.UserID, "", in.OptionID)
		return err
	})
	if err != nil {
		switch {
		case errors.Is(err, governance.ErrAlreadyVoted):
			writeErr(w, http.StatusConflict, err.Error())
		case errors.Is(err, governance.ErrVoteNotOpen):
			writeErr(w, http.StatusConflict, err.Error())
		case errors.Is(err, governance.ErrOptionInvalid):
			writeErr(w, http.StatusBadRequest, err.Error())
		case store.IsNotFound(err):
			writeErr(w, http.StatusNotFound, "votacao nao encontrada")
		default:
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	// A resposta devolve so a apuracao agregada - nunca o vinculo votoeleitor.
	writeJSON(w, http.StatusOK, res)
}

func (a *App) handleVoteResult(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var res *governance.Result
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		res, err = a.Governance.Tally(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "votacao nao encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ---- Convenios e documentacao legal (G6) ----

func (a *App) handleListLegalDocuments(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []governance.LegalDocument
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Governance.ListLegalDocuments(r.Context(), tx, r.URL.Query().Get("kind"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"documents": out})
}

func (a *App) handleCreateLegalDocument(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in governance.LegalInput
	if err := readJSON(r, &in); err != nil || in.Title == "" {
		writeErr(w, http.StatusBadRequest, "title e obrigatorio")
		return
	}
	b := boundsFromClaims(claims)
	var d *governance.LegalDocument
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		d, err = a.Governance.CreateLegalDocument(r.Context(), tx, claims.TenantID, branchID, claims.UserID, in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (a *App) handleUpdateLegalDocument(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in governance.LegalInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	b := boundsFromClaims(claims)
	var d *governance.LegalDocument
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		d, err = a.Governance.UpdateLegalDocument(r.Context(), tx, r.PathValue("id"), in)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "documento nao encontrado")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (a *App) handleDeleteLegalDocument(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Governance.DeleteLegalDocument(r.Context(), tx, r.PathValue("id"))
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "documento nao encontrado")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- Painel de mandatos (G5) ----

func (a *App) handleListMandates(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []governance.Mandate
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Governance.ListMandates(r.Context(), tx, 60)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"mandates": out})
}
