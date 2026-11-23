package vista

import (
	"fmt"
	"html"
	"strings"

	"awesomeProject/internal/siweb"
)

// CalificacionesHTML dibuja la tabla de notas de un curso.
func CalificacionesHTML(filas []siweb.Calificacion, titulo string) string {
	if len(filas) == 0 {
		return `<p>No hay calificaciones para mostrar.</p>`
	}

	var b strings.Builder
	b.WriteString(cssCalif)
	if titulo != "" {
		fmt.Fprintf(&b, `<p class="sub">%s · %d estudiantes</p>`, html.EscapeString(titulo), len(filas))
	}

	b.WriteString(`<table class="calif"><thead><tr>
<th>#</th><th>Estudiante</th><th>1er 50%</th><th>2do 50%</th><th>Habilitación</th><th>Definitiva</th>
</tr></thead><tbody>`)
	for i, f := range filas {
		fmt.Fprintf(&b, `<tr><td class="num">%d</td><td class="est">%s<small>%d</small></td><td>%s</td><td>%s</td><td>%s</td><td class="fin">%s</td></tr>`,
			i+1,
			html.EscapeString(tituloCaso(f.Nombre)), f.Codigo,
			notaTexto(f.Primer50), notaTexto(f.Segundo50), notaTexto(f.Habilitacion), notaTexto(f.Final))
	}
	b.WriteString(`</tbody></table>`)
	return b.String()
}

// notaTexto muestra el centinela "99" (o vacío) como un guion, para que la tabla
// no se llene de noventainueves donde todavía no hay nota.
func notaTexto(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || v == "99" {
		return "—"
	}
	return html.EscapeString(v)
}

const cssCalif = `<style>
table.calif{border-collapse:collapse;width:100%;font-size:.88rem}
table.calif th,table.calif td{border:1px solid #e6e6e6;padding:.45rem .6rem;text-align:center}
table.calif th{background:#f5f6f8;font-size:.8rem}
table.calif td.num{color:#999;width:36px}
table.calif td.est{text-align:left}
table.calif td.est small{display:block;color:#999;font-size:.72rem}
table.calif td.fin{font-weight:600}
</style>`
