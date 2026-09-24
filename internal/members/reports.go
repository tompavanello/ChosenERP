package members

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// ---- Aniversariantes (requisito do módulo de relatórios) ----

// Birthday é um membro que faz aniversário no mês consultado.
type Birthday struct {
	ID        string  `json:"id"`
	FullName  string  `json:"full_name"`
	BirthDate string  `json:"birth_date"`
	Day       int     `json:"day"`
	Age       int     `json:"age"`
	WhatsApp  *string `json:"whatsapp,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	BranchID  string  `json:"branch_id"`
}

// MarriageAnniversary é um aniversário de casamento. Quando os dois cônjuges
// têm a mesma data, só um registro é devolvido (com o nome do cônjuge).
type MarriageAnniversary struct {
	ID           string  `json:"id"`
	FullName     string  `json:"full_name"`
	SpouseName   *string `json:"spouse_name,omitempty"`
	MarriageDate string  `json:"marriage_date"`
	Day          int     `json:"day"`
	Years        int     `json:"years"`
}

// Birthdays lista os aniversariantes (nascimento) do mês.
func (r *Repo) Birthdays(ctx context.Context, tx pgx.Tx, month int) ([]Birthday, error) {
	rows, err := tx.Query(ctx, `
		SELECT m.id::text, m.full_name, m.birth_date::text,
		       EXTRACT(DAY FROM m.birth_date)::int,
		       date_part('year', age(m.birth_date))::int,
		       m.whatsapp, m.phone, m.branch_id::text
		FROM members m
		WHERE m.birth_date IS NOT NULL
		  AND EXTRACT(MONTH FROM m.birth_date)::int = $1
		ORDER BY EXTRACT(DAY FROM m.birth_date), m.full_name`, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Birthday{}
	for rows.Next() {
		var b Birthday
		if err := rows.Scan(&b.ID, &b.FullName, &b.BirthDate, &b.Day, &b.Age,
			&b.WhatsApp, &b.Phone, &b.BranchID); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Marriages lista os aniversários de casamento do mês, deduplicando casais que
// compartilham a data (só a ponta de menor id entra).
func (r *Repo) Marriages(ctx context.Context, tx pgx.Tx, month int) ([]MarriageAnniversary, error) {
	rows, err := tx.Query(ctx, `
		SELECT m.id::text, m.full_name, m.marriage_date::text,
		       EXTRACT(DAY FROM m.marriage_date)::int,
		       date_part('year', age(m.marriage_date))::int,
		       sp.full_name
		FROM members m
		LEFT JOIN LATERAL (
			SELECT m2.full_name
			FROM member_relationships r
			JOIN members m2 ON m2.id = r.related_id
			WHERE r.member_id = m.id AND r.kind = 'spouse'
			LIMIT 1
		) sp ON true
		WHERE m.marriage_date IS NOT NULL
		  AND EXTRACT(MONTH FROM m.marriage_date)::int = $1
		  AND NOT EXISTS (
			SELECT 1
			FROM member_relationships r2
			JOIN members m3 ON m3.id = r2.related_id
			WHERE r2.member_id = m.id AND r2.kind = 'spouse'
			  AND m3.id < m.id
			  AND m3.marriage_date = m.marriage_date
		  )
		ORDER BY EXTRACT(DAY FROM m.marriage_date), m.full_name`, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MarriageAnniversary{}
	for rows.Next() {
		var a MarriageAnniversary
		if err := rows.Scan(&a.ID, &a.FullName, &a.MarriageDate, &a.Day, &a.Years,
			&a.SpouseName); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ---- Demográficos ----

// PyramidRow é uma faixa etária da pirâmide.
type PyramidRow struct {
	Bucket string `json:"bucket"`
	Male   int    `json:"male"`
	Female int    `json:"female"`
	Total  int    `json:"total"`
}

// CountRow é uma contagem genérica por chave.
type CountRow struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

// Demographics agrega o perfil da base de membros no escopo da sessão (RLS).
type Demographics struct {
	Total           int          `json:"total"`
	AgePyramid      []PyramidRow `json:"age_pyramid"`
	ByGender        []CountRow   `json:"by_gender"`
	ByMaritalStatus []CountRow   `json:"by_marital_status"`
	ByStatus        []CountRow   `json:"by_status"`
	ByState         []CountRow   `json:"by_state"`
	ByCity          []CountRow   `json:"by_city"`
}

// ageBuckets define a ordem fixa das faixas (inclusive as vazias).
var ageBuckets = []string{"0-12", "13-17", "18-25", "26-35", "36-45", "46-55", "56-65", "66+"}

// Demographics monta o painel demográfico.
func (r *Repo) Demographics(ctx context.Context, tx pgx.Tx) (Demographics, error) {
	var d Demographics
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM members`).Scan(&d.Total); err != nil {
		return d, err
	}

	// Pirâmide etária (por sexo).
	rows, err := tx.Query(ctx, `
		WITH base AS (
			SELECT
				CASE
					WHEN age(birth_date) < interval '13 years' THEN '0-12'
					WHEN age(birth_date) < interval '18 years' THEN '13-17'
					WHEN age(birth_date) < interval '26 years' THEN '18-25'
					WHEN age(birth_date) < interval '36 years' THEN '26-35'
					WHEN age(birth_date) < interval '46 years' THEN '36-45'
					WHEN age(birth_date) < interval '56 years' THEN '46-55'
					WHEN age(birth_date) < interval '66 years' THEN '56-65'
					ELSE '66+'
				END AS bucket,
				lower(COALESCE(gender, '')) AS g
			FROM members
			WHERE birth_date IS NOT NULL
		)
		SELECT bucket,
		       count(*) FILTER (WHERE g = 'male')::int,
		       count(*) FILTER (WHERE g = 'female')::int,
		       count(*)::int
		FROM base GROUP BY bucket`)
	if err != nil {
		return d, err
	}
	byBucket := map[string]PyramidRow{}
	for rows.Next() {
		var p PyramidRow
		if err := rows.Scan(&p.Bucket, &p.Male, &p.Female, &p.Total); err != nil {
			rows.Close()
			return d, err
		}
		byBucket[p.Bucket] = p
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return d, err
	}
	d.AgePyramid = make([]PyramidRow, 0, len(ageBuckets))
	for _, b := range ageBuckets {
		if p, ok := byBucket[b]; ok {
			d.AgePyramid = append(d.AgePyramid, p)
		} else {
			d.AgePyramid = append(d.AgePyramid, PyramidRow{Bucket: b})
		}
	}

	if d.ByGender, err = r.countRows(ctx, tx,
		`SELECT COALESCE(NULLIF(gender,''), 'nao_informado'), count(*)::int
		 FROM members GROUP BY 1 ORDER BY 2 DESC`); err != nil {
		return d, err
	}
	if d.ByMaritalStatus, err = r.countRows(ctx, tx,
		`SELECT COALESCE(NULLIF(marital_status,''), 'nao_informado'), count(*)::int
		 FROM members GROUP BY 1 ORDER BY 2 DESC`); err != nil {
		return d, err
	}
	if d.ByStatus, err = r.countRows(ctx, tx,
		`SELECT membership_status, count(*)::int
		 FROM members GROUP BY 1 ORDER BY 2 DESC`); err != nil {
		return d, err
	}
	if d.ByState, err = r.countRows(ctx, tx,
		`SELECT COALESCE(NULLIF(upper(m.address->>'state'),''), 'nao_informado'), count(*)::int
		 FROM members m GROUP BY 1 ORDER BY 2 DESC`); err != nil {
		return d, err
	}
	if d.ByCity, err = r.countRows(ctx, tx,
		`SELECT COALESCE(NULLIF(btrim(m.address->>'city'),''), 'nao_informado'), count(*)::int
		 FROM members m GROUP BY 1 ORDER BY 2 DESC LIMIT 20`); err != nil {
		return d, err
	}
	return d, nil
}

func (r *Repo) countRows(ctx context.Context, tx pgx.Tx, sql string) ([]CountRow, error) {
	rows, err := tx.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CountRow{}
	for rows.Next() {
		var c CountRow
		if err := rows.Scan(&c.Key, &c.Count); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
