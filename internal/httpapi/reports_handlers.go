package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/delivery"
	"chosenerp/internal/documents"
	"chosenerp/internal/finance"
	"chosenerp/internal/members"
	"chosenerp/internal/store"
)

// ---- Relatórios financeiros ----

func (a *App) handleMonthlyBalance(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	q := r.URL.Query()
	b := boundsFromClaims(claims)
	var points []finance.MonthlyPoint
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		points, err = a.Finance.MonthlySeries(r.Context(), tx, q.Get("from"), q.Get("to"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"series": points})
}

func (a *App) handleDRE(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	q := r.URL.Query()
	b := boundsFromClaims(claims)
	var dre finance.DRE
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		dre, err = a.Finance.BuildDRE(r.Context(), tx, q.Get("from"), q.Get("to"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dre)
}

// exportPeriod resolve o período a partir de query (from/to ou year).
func exportPeriod(q map[string][]string) (string, string) {
	year := first(q["year"])
	if year == "" {
		return first(q["from"]), first(q["to"])
	}
	n, _ := strconv.Atoi(year)
	return fmt.Sprintf("%d-01-01", n), fmt.Sprintf("%d-01-01", n+1)
}

func first(v []string) string {
	if len(v) > 0 {
		return v[0]
	}
	return ""
}

// money formata um valor para os relatórios exportados.
func money(v float64) string { return fmt.Sprintf("%.2f", v) }

func (a *App) handleExportBalance(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	q := r.URL.Query()
	from, to := exportPeriod(q)
	format := first(q["format"])
	b := boundsFromClaims(claims)
	var points []finance.MonthlyPoint
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		points, err = a.Finance.MonthlySeries(r.Context(), tx, from, to)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := make([][]string, 0, len(points))
	for _, p := range points {
		rows = append(rows, []string{p.Month, money(p.Income), money(p.Expense), money(p.Net)})
	}
	writeReport(w, "balancete-"+from+"-"+to, "Balancete mensal",
		"Período: "+from+" a "+to, format,
		[]exportSection{{Headers: []string{"Mês", "Entradas", "Saídas", "Saldo"}, Rows: rows}})
}

func (a *App) handleExportDRE(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	q := r.URL.Query()
	from, to := exportPeriod(q)
	format := first(q["format"])
	b := boundsFromClaims(claims)
	var dre finance.DRE
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		dre, err = a.Finance.BuildDRE(r.Context(), tx, from, to)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	lines := make([][]string, 0, len(dre.Lines))
	for _, l := range dre.Lines {
		tipo := "Entrada"
		if l.Type == "expense" {
			tipo = "Saída"
		}
		lines = append(lines, []string{tipo, l.Category, money(l.Total)})
	}
	totais := [][]string{
		{"Entradas", "", money(dre.Income)},
		{"Saídas", "", money(dre.Expense)},
		{"Resultado", "", money(dre.Net)},
	}
	if dre.Comparison != nil {
		totais = append(totais, []string{"Comparativo (período anterior)", fmt.Sprintf("%.1f%%", dre.Comparison.DeltaPct), ""})
	}
	writeReport(w, "dre-"+from+"-"+to, "DRE — Demonstração do Resultado",
		"Período: "+from+" a "+to, format,
		[]exportSection{
			{Title: "Por conta contábil", Headers: []string{"Tipo", "Conta contábil", "Total"}, Rows: lines},
			{Title: "Totais", Headers: []string{"Indicador", "Observação", "Valor"}, Rows: totais},
		})
}

// ---- Aniversariantes e demográficos (pessoas) ----

// handleBirthdays devolve os aniversariantes de nascimento e de casamento de um
// mês (1-12; padrão: mês corrente).
func (a *App) handleBirthdays(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	month := monthParam(r)
	b := boundsFromClaims(claims)
	var birthdays []members.Birthday
	var marriages []members.MarriageAnniversary
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		birthdays, err = a.Members.Birthdays(r.Context(), tx, month)
		if err != nil {
			return err
		}
		marriages, err = a.Members.Marriages(r.Context(), tx, month)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"month": month, "birthdays": birthdays, "marriages": marriages,
	})
}

// handleDemographics devolve o painel demográfico (pirâmide etária, status,
// estado civil, sexo e distribuição geográfica) do escopo da sessão.
func (a *App) handleDemographics(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var d members.Demographics
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		d, err = a.Members.Demographics(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// ---- Exportação: aniversariantes e demográficos ----

// monthParam lê o mês (1-12) da query; padrão mês corrente.
func monthParam(r *http.Request) int {
	month := int(time.Now().Month())
	if m := r.URL.Query().Get("month"); m != "" {
		if n, err := strconv.Atoi(m); err == nil && n >= 1 && n <= 12 {
			month = n
		}
	}
	return month
}

var monthNamesPT = [...]string{"", "janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho", "agosto", "setembro", "outubro", "novembro", "dezembro"}

var statusLabels = map[string]string{
	"active": "Ativo (professo)", "member": "Não professo", "inactive": "Inativo",
	"dismissed": "Baixado do Rol", "transferred": "Transferido", "deceased": "Falecido", "other": "Outros",
}

var maritalLabels = map[string]string{
	"single": "Solteiro(a)", "married": "Casado(a)", "divorced": "Divorciado(a)", "widowed": "Viúvo(a)",
}

var genderLabels = map[string]string{"male": "Masculino", "female": "Feminino", "other": "Outro"}

func labelOf(m map[string]string, k string) string {
	if k == "" || k == "nao_informado" {
		return "Não informado"
	}
	if v, ok := m[k]; ok {
		return v
	}
	return k
}

func (a *App) handleExportBirthdays(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	month := monthParam(r)
	part := first(r.URL.Query()["part"]) // "", "nascimento" ou "casamento"
	format := first(r.URL.Query()["format"])
	b := boundsFromClaims(claims)
	var birthdays []members.Birthday
	var marriages []members.MarriageAnniversary
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		if birthdays, err = a.Members.Birthdays(r.Context(), tx, month); err != nil {
			return err
		}
		marriages, err = a.Members.Marriages(r.Context(), tx, month)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	brows := make([][]string, 0, len(birthdays))
	for _, p := range birthdays {
		brows = append(brows, []string{strconv.Itoa(p.Day), p.FullName, p.BirthDate, strconv.Itoa(p.Age)})
	}
	mrows := make([][]string, 0, len(marriages))
	for _, m := range marriages {
		casal := m.FullName
		if m.SpouseName != nil && *m.SpouseName != "" {
			casal += " & " + *m.SpouseName
		}
		mrows = append(mrows, []string{strconv.Itoa(m.Day), casal, m.MarriageDate, strconv.Itoa(m.Years)})
	}
	// Nascimento e casamento podem ser exportados juntos ou como relatórios
	// separados (part=nascimento|casamento).
	sections := []exportSection{}
	if part != "casamento" {
		sections = append(sections, exportSection{Title: "Aniversários de nascimento", Headers: []string{"Dia", "Nome", "Nascimento", "Idade"}, Rows: brows})
	}
	if part != "nascimento" {
		sections = append(sections, exportSection{Title: "Aniversários de casamento", Headers: []string{"Dia", "Casal", "Data", "Anos"}, Rows: mrows})
	}
	title := "Aniversariantes"
	fname := fmt.Sprintf("aniversariantes-%d-%02d", time.Now().Year(), month)
	switch part {
	case "nascimento":
		title = "Aniversariantes — Nascimento"
		fname += "-nascimento"
	case "casamento":
		title = "Aniversariantes — Casamento"
		fname += "-casamento"
	}
	writeReport(w, fname, title, "Mês: "+monthNamesPT[month], format, sections)
}

func (a *App) handleExportDemographics(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	format := first(r.URL.Query()["format"])
	b := boundsFromClaims(claims)
	var d members.Demographics
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		d, err = a.Members.Demographics(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	counts := func(rows []members.CountRow, labels map[string]string) [][]string {
		out := make([][]string, 0, len(rows))
		for _, c := range rows {
			out = append(out, []string{labelOf(labels, c.Key), strconv.Itoa(c.Count)})
		}
		return out
	}
	pyramid := make([][]string, 0, len(d.AgePyramid))
	for _, p := range d.AgePyramid {
		pyramid = append(pyramid, []string{p.Bucket, strconv.Itoa(p.Male), strconv.Itoa(p.Female), strconv.Itoa(p.Total)})
	}
	writeReport(w, fmt.Sprintf("demograficos-%d", time.Now().Year()),
		"Painel demográfico", fmt.Sprintf("Total de membros: %d", d.Total), format,
		[]exportSection{
			{Title: "Pirâmide etária", Headers: []string{"Faixa", "Masculino", "Feminino", "Total"}, Rows: pyramid},
			{Title: "Situação no Rol", Headers: []string{"Situação", "Total"}, Rows: counts(d.ByStatus, statusLabels)},
			{Title: "Estado civil", Headers: []string{"Estado civil", "Total"}, Rows: counts(d.ByMaritalStatus, maritalLabels)},
			{Title: "Sexo", Headers: []string{"Sexo", "Total"}, Rows: counts(d.ByGender, genderLabels)},
			{Title: "Distribuição por UF", Headers: []string{"UF", "Total"}, Rows: counts(d.ByState, nil)},
			{Title: "Cidades", Headers: []string{"Cidade", "Total"}, Rows: counts(d.ByCity, nil)},
		})
}

// ---- Demonstrativo Mensal (regime de caixa) ----

func (a *App) handleMonthlyStatement(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	month := r.URL.Query().Get("month")
	b := boundsFromClaims(claims)
	var st finance.MonthlyStatement
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		st, err = a.Finance.BuildMonthlyStatement(r.Context(), tx, month)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (a *App) handleExportMonthlyStatement(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	month := r.URL.Query().Get("month")
	format := first(r.URL.Query()["format"])
	b := boundsFromClaims(claims)
	var st finance.MonthlyStatement
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		st, err = a.Finance.BuildMonthlyStatement(r.Context(), tx, month)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	headers := []string{"Cód", "Descrição"}
	for _, wk := range st.Weeks {
		headers = append(headers, wk.Label)
	}
	headers = append(headers, "Total")
	lines := func(ls []finance.StatementLine) [][]string {
		out := make([][]string, 0, len(ls))
		for _, l := range ls {
			row := []string{l.Code, l.Name}
			for _, v := range l.Weeks {
				row = append(row, money(v))
			}
			row = append(row, money(l.Total))
			out = append(out, row)
		}
		return out
	}
	writeReport(w, "demonstrativo-"+st.Month, "Demonstrativo Mensal (Regime de Caixa)",
		"Período: "+st.From+" a "+st.To, format,
		[]exportSection{
			{Title: "Entradas", Headers: headers, Rows: lines(st.Income)},
			{Title: "Saídas", Headers: headers, Rows: lines(st.Expense)},
			{Title: "Resumo", Headers: []string{"Indicador", "Valor"}, Rows: [][]string{
				{"Saldo inicial", money(st.OpeningBalance)},
				{"Soma das entradas", money(st.TotalIncome)},
				{"Soma das saídas", money(st.TotalExpense)},
				{"Saldo final", money(st.ClosingBalance)},
			}},
		})
}

// ---- Recibos: renderização e envio ----

func (a *App) handleRenderReceipt(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	b := boundsFromClaims(claims)
	var body string
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		doc, err := a.Documents.GetByID(r.Context(), tx, id)
		if err != nil {
			return err
		}
		tenant, _ := a.Documents.TenantName(r.Context(), tx)
		body, err = documents.RenderReceiptHTML(doc, tenant)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "document not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(body))
}

func (a *App) handleSendDocument(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var in struct {
		Channel   string `json:"channel"`
		Recipient string `json:"recipient"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b := boundsFromClaims(claims)

	// Fase 1 (dentro da transação RLS): enfileira a entrega e monta a mensagem.
	var d *documents.Delivery
	var msg delivery.Message
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		doc, err := a.Documents.GetByID(r.Context(), tx, id)
		if err != nil {
			return err
		}
		tenant, _ := a.Documents.TenantName(r.Context(), tx)
		html, err := documents.RenderReceiptHTML(doc, tenant)
		if err != nil {
			return err
		}
		d, err = a.Documents.QueueDelivery(r.Context(), tx, claims.TenantID, claims.BranchID, id, in.Channel, in.Recipient)
		if err != nil {
			return err
		}
		msg = delivery.Message{
			Channel: in.Channel, Recipient: in.Recipient,
			Subject: doc.Title, HTML: html,
			Text:       "Código de validação: " + doc.QRToken,
			Link:       delivery.PublicLink(a.Config.AppBaseURL, doc.Kind, doc.QRToken),
			TenantName: tenant,
		}
		return nil
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "document not found")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	// Fase 2: envio real (fora da transação) + registro do resultado na outbox.
	if err := a.Dispatch.Send(r.Context(), msg); err != nil {
		_ = a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
			return a.Documents.MarkFailed(r.Context(), tx, d.ID, err.Error())
		})
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	_ = a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Documents.MarkSent(r.Context(), tx, d.ID)
	})
	now := time.Now().UTC().Format(time.RFC3339)
	d.Status = "sent"
	d.SentAt = &now
	d.Error = nil
	d.Attempts++
	writeJSON(w, http.StatusCreated, d)
}

func (a *App) handleListDeliveries(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	b := boundsFromClaims(claims)
	var out []documents.Delivery
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Documents.ListDeliveries(r.Context(), tx, id)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deliveries": out})
}

// ---- Demonstrativo para Assembleia ----

// handleAssemblyReport resume o período (entradas, saídas, resultado, por conta
// e evolução mensal) num formato pronto para apresentação em assembleia.
func (a *App) handleAssemblyReport(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	from, to := exportPeriod(r.URL.Query())
	b := boundsFromClaims(claims)
	var bal finance.Balance
	var months []finance.MonthlyPoint
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		if bal, err = a.Finance.SumBalance(r.Context(), tx, ""); err != nil {
			return err
		}
		months, err = a.Finance.MonthlySeries(r.Context(), tx, from, to)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from": from, "to": to, "balance": bal, "months": months,
	})
}

func assemblySections(bal finance.Balance, months []finance.MonthlyPoint, from, to string) []exportSection {
	var entradas, saidas [][]string
	for _, it := range bal.ByCategory {
		row := []string{it.Category, money(it.Total)}
		if it.Type == "income" {
			entradas = append(entradas, row)
		} else {
			saidas = append(saidas, row)
		}
	}
	resumo := [][]string{
		{"Entradas", money(bal.Income)},
		{"Saídas", money(bal.Expense)},
		{"Resultado do período", money(bal.Net)},
	}
	mrows := make([][]string, 0, len(months))
	for _, m := range months {
		mrows = append(mrows, []string{m.Month, money(m.Income), money(m.Expense), money(m.Net)})
	}
	return []exportSection{
		{Title: "Resumo do período (" + from + " a " + to + ")", Headers: []string{"Indicador", "Valor"}, Rows: resumo},
		{Title: "Entradas por conta", Headers: []string{"Conta contábil", "Total"}, Rows: entradas},
		{Title: "Saídas por conta", Headers: []string{"Conta contábil", "Total"}, Rows: saidas},
		{Title: "Evolução mensal", Headers: []string{"Mês", "Entradas", "Saídas", "Saldo"}, Rows: mrows},
		{Title: "Aprovação em assembleia", Headers: []string{"Assinatura", "Função", "Data"}, Rows: [][]string{
			{"", "Presidente da assembleia", ""},
			{"", "Tesoureiro(a)", ""},
			{"", "Secretário(a)", ""},
		}},
	}
}

func (a *App) handleExportAssembly(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	q := r.URL.Query()
	from, to := exportPeriod(q)
	format := first(q["format"])
	b := boundsFromClaims(claims)
	var bal finance.Balance
	var months []finance.MonthlyPoint
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		if bal, err = a.Finance.SumBalance(r.Context(), tx, ""); err != nil {
			return err
		}
		months, err = a.Finance.MonthlySeries(r.Context(), tx, from, to)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeReport(w, "demonstrativo-assembleia-"+from+"-"+to,
		"Demonstrativo financeiro para assembleia", "Período: "+from+" a "+to, format,
		assemblySections(bal, months, from, to))
}
