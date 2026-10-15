package httpapi

import (
	"errors"
	"net/http"

	"chosenerp/internal/bible"
)

// ---- Biblia (API externa) ----

func (a *App) handleBibleVersions(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("language")
	if !a.Bible.Enabled() {
		writeJSON(w, http.StatusOK, map[string]any{"configured": false, "versions": []bible.Version{}})
		return
	}
	vs, err := a.Bible.Versions(r.Context(), lang)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "falha ao consultar a Biblia")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"configured": true, "versions": vs})
}

func (a *App) handleBibleBooks(w http.ResponseWriter, r *http.Request) {
	if !a.Bible.Enabled() {
		writeErr(w, http.StatusServiceUnavailable, "bible_disabled")
		return
	}
	books, err := a.Bible.Books(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadGateway, "falha ao consultar a Biblia")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"books": books})
}

func (a *App) handleBibleChapters(w http.ResponseWriter, r *http.Request) {
	if !a.Bible.Enabled() {
		writeErr(w, http.StatusServiceUnavailable, "bible_disabled")
		return
	}
	chs, err := a.Bible.Chapters(r.Context(), r.PathValue("id"), r.PathValue("bookId"))
	if err != nil {
		writeErr(w, http.StatusBadGateway, "falha ao consultar a Biblia")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"chapters": chs})
}

func (a *App) handleBibleChapter(w http.ResponseWriter, r *http.Request) {
	if !a.Bible.Enabled() {
		writeErr(w, http.StatusServiceUnavailable, "bible_disabled")
		return
	}
	p, err := a.Bible.Chapter(r.Context(), r.PathValue("id"), r.PathValue("chapterId"))
	if err != nil {
		if errors.Is(err, bible.ErrDisabled) {
			writeErr(w, http.StatusServiceUnavailable, "bible_disabled")
			return
		}
		writeErr(w, http.StatusBadGateway, "falha ao consultar a Biblia")
		return
	}
	writeJSON(w, http.StatusOK, p)
}
