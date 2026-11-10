package vista

import (
	"fmt"
	"html"
	"sort"
	"strings"

	"awesomeProject/internal/siweb"
)

// HistoriaHTML dibuja la historia académica agrupada por semestre.
func HistoriaHTML(h *siweb.HistoriaAcademica) string {
	if h == nil || len(h.Materias) == 0 {
		return `<p>No hay materias para mostrar.</p>`
	}

	var b strings.Builder
	b.WriteString(cssNotas)

	// Encabezado del estudiante (solo lo académico).
	e := h.Estudiante
	fmt.Fprintf(&b, `<div class="est"><b>%s</b> · Código %d<br><span>%s · %s · %s</span></div>`,
		html.EscapeString(tituloCaso(e.Nombre)), e.Codigo,
		html.EscapeString(tituloCaso(e.Programa)), html.EscapeString(tituloCaso(e.Facultad)), html.EscapeString(tituloCaso(e.Sede)))

	// Tarjetas de resumen.
	fmt.Fprintf(&b, `<div class="tiles">
<div class="tile"><span>Promedio general</span><b>%.2f</b></div>
<div class="tile"><span>Créditos aprobados</span><b>%d</b></div>
<div class="tile"><span>Créditos cursados</span><b>%d</b></div>
<div class="tile"><span>Materias</span><b>%d</b></div></div>`,
		h.PromedioGeneral(), h.CreditosAprobados(), h.CreditosCursados(), len(h.Materias))

	// Agrupar por periodo, en orden cronológico.
	materias := make([]siweb.Materia, len(h.Materias))
	copy(materias, h.Materias)
	sort.SliceStable(materias, func(i, j int) bool {
		return materias[i].NroPeriodo < materias[j].NroPeriodo
	})

	var periodoActual string
	abierto := false
	var sumaPeriodo float64
	var credPeriodo int

	cerrarPeriodo := func() {
		if abierto {
			prom := 0.0
			if credPeriodo > 0 {
				prom = sumaPeriodo / float64(credPeriodo)
			}
			fmt.Fprintf(&b, `</tbody><tfoot><tr><td colspan="4">Promedio del semestre</td><td class="nota">%.2f</td></tr></tfoot></table>`, prom)
			sumaPeriodo, credPeriodo = 0, 0
		}
	}

	for _, m := range materias {
		if m.Periodo != periodoActual {
			cerrarPeriodo()
			periodoActual = m.Periodo
			abierto = true
			fmt.Fprintf(&b, `<h3 class="periodo">%s</h3><table class="notas"><thead>
<tr><th>Materia</th><th>Créd.</th><th>P50</th><th>S50</th><th>Final</th></tr></thead><tbody>`,
				html.EscapeString(tituloCaso(m.Periodo)))
		}
		if m.Final != nil {
			sumaPeriodo += *m.Final * float64(m.Creditos)
			credPeriodo += m.Creditos
		}
		clase := "rep"
		if m.Aprobada() {
			clase = "apr"
		}
		fmt.Fprintf(&b, `<tr><td>%s</td><td class="c">%d</td><td class="c">%s</td><td class="c">%s</td><td class="nota %s">%s</td></tr>`,
			html.EscapeString(tituloCaso(m.Nombre)), m.Creditos,
			nota(m.P50), nota(m.S50), clase, nota(m.Final))
	}
	cerrarPeriodo()

	b.WriteString(`<p class="pie">Aprueba con ` + fmt.Sprintf("%.1f", siweb.NotaMinima) + ` o más. Verde = aprobada, rojo = reprobada.</p>`)
	return b.String()
}

// nota formatea una nota (o "-" si no hay).
func nota(n *float64) string {
	if n == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f", *n)
}

const cssNotas = `<style>
.est{margin:.2rem 0 1rem;line-height:1.35}
.est span{color:#666;font-size:.9rem}
.tiles{display:flex;flex-wrap:wrap;gap:.7rem;margin:0 0 1.5rem}
.tile{flex:1;min-width:130px;border:1px solid #e6e6e6;border-radius:9px;padding:.7rem .9rem;text-align:center}
.tile span{display:block;color:#777;font-size:.78rem}
.tile b{font-size:1.5rem;color:#0b5394}
h3.periodo{margin:1.3rem 0 .4rem;font-size:1rem;color:#333;border-left:4px solid #0b5394;padding-left:.5rem}
table.notas{border-collapse:collapse;width:100%;margin-bottom:.5rem}
table.notas th,table.notas td{border-bottom:1px solid #eee;padding:.4rem .5rem;text-align:left;font-size:.88rem}
table.notas th{background:#f7f8fa;font-size:.8rem;color:#555}
table.notas td.c,table.notas th:nth-child(n+2){text-align:center}
table.notas td.nota{text-align:center;font-weight:600}
table.notas td.apr{color:#0a7a2f}
table.notas td.rep{color:#b00020}
table.notas tfoot td{border-top:2px solid #ddd;border-bottom:0;font-weight:600;color:#333}
table.notas tfoot td.nota{color:#0b5394;text-align:center}
.pie{color:#888;font-size:.8rem;margin-top:1rem}
</style>`
