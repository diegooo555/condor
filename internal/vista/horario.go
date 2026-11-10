// Paquete vista: dibuja el horario como una grilla semanal en HTML.
package vista

import (
	"fmt"
	"html"
	"sort"
	"strings"

	"awesomeProject/internal/siweb"
)

var nombreDia = map[int]string{1: "Lunes", 2: "Martes", 3: "Miércoles", 4: "Jueves", 5: "Viernes", 6: "Sábado", 7: "Domingo"}

// paleta de colores para las materias (se asigna uno distinto a cada materia).
var paleta = []string{"#1f77b4", "#2ca02c", "#d62728", "#9467bd", "#e67e22", "#17a2b8", "#8e44ad", "#16a085", "#c0392b", "#2c3e50"}

// HorarioHTML devuelve el fragmento HTML (con su propio CSS) de la grilla.
func HorarioHTML(clases []siweb.Clase, titulo string) string {
	if len(clases) == 0 {
		return `<p>No hay clases para mostrar.</p>`
	}

	// Color por materia.
	colores := map[string]string{}
	for _, c := range clases {
		if _, ok := colores[c.Materia]; !ok {
			colores[c.Materia] = paleta[len(colores)%len(paleta)]
		}
	}

	// Rango de horas y días presentes.
	minH, maxH := 24, 0
	diaConClase := map[int]bool{}
	inicio := map[[2]int][]siweb.Clase{} // (día,hora) -> clases que empiezan ahí
	for _, c := range clases {
		if c.HoraInicio < minH {
			minH = c.HoraInicio
		}
		if c.HoraFin > maxH {
			maxH = c.HoraFin
		}
		diaConClase[c.Dia] = true
		k := [2]int{c.Dia, c.HoraInicio}
		inicio[k] = append(inicio[k], c)
	}

	// Columnas: Lunes a Viernes siempre; sábado/domingo solo si hay clase.
	dias := []int{1, 2, 3, 4, 5}
	if diaConClase[6] {
		dias = append(dias, 6)
	}
	if diaConClase[7] {
		dias = append(dias, 7)
	}

	var b strings.Builder
	b.WriteString(css)
	if titulo != "" {
		fmt.Fprintf(&b, `<p class="sub">%s</p>`, html.EscapeString(titulo))
	}

	// Encabezado.
	b.WriteString(`<table class="horario"><thead><tr><th class="hora">Hora</th>`)
	for _, d := range dias {
		fmt.Fprintf(&b, `<th>%s</th>`, nombreDia[d])
	}
	b.WriteString(`</tr></thead><tbody>`)

	// Filas: una por hora. Un bloque ocupa varias filas con rowspan.
	ocupado := map[[2]int]bool{}
	for h := minH; h < maxH; h++ {
		fmt.Fprintf(&b, `<tr><td class="hora">%d:00</td>`, h)
		for _, d := range dias {
			if ocupado[[2]int{d, h}] {
				continue // esta celda la cubre un rowspan de arriba
			}
			clasesAqui := inicio[[2]int{d, h}]
			if len(clasesAqui) == 0 {
				b.WriteString(`<td></td>`)
				continue
			}
			dur := 1
			for _, c := range clasesAqui {
				if c.Duracion() > dur {
					dur = c.Duracion()
				}
			}
			for k := 1; k < dur; k++ {
				ocupado[[2]int{d, h + k}] = true
			}
			fmt.Fprintf(&b, `<td rowspan="%d">`, dur)
			for _, c := range clasesAqui {
				b.WriteString(bloque(c, colores[c.Materia]))
			}
			b.WriteString(`</td>`)
		}
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</tbody></table>`)

	b.WriteString(leyenda(clases, colores))
	return b.String()
}

// bloque dibuja una clase dentro de una celda.
func bloque(c siweb.Clase, color string) string {
	salon := c.Aula
	if c.Edificio > 0 {
		salon = fmt.Sprintf("%s · Ed. %d", c.Aula, c.Edificio)
	}
	return fmt.Sprintf(`<div class="clase" style="background:%s">
<b>%s</b><small>%d:00–%d:00</small><small>%s</small></div>`,
		color, html.EscapeString(tituloCaso(c.Materia)),
		c.HoraInicio, c.HoraFin, html.EscapeString(salon))
}

// leyenda lista las materias con su color y el total de horas semanales.
func leyenda(clases []siweb.Clase, colores map[string]string) string {
	horas := map[string]int{}
	for _, c := range clases {
		horas[c.Materia] += c.Duracion()
	}
	materias := make([]string, 0, len(colores))
	for m := range colores {
		materias = append(materias, m)
	}
	sort.Strings(materias)

	var b strings.Builder
	b.WriteString(`<div class="leyenda">`)
	for _, m := range materias {
		fmt.Fprintf(&b, `<span class="chip"><i style="background:%s"></i>%s <b>(%dh)</b></span>`,
			colores[m], html.EscapeString(tituloCaso(m)), horas[m])
	}
	b.WriteString(`</div>`)
	return b.String()
}

// tituloCaso pasa "LENGUAJES FORMALES" a "Lenguajes Formales" (se ve más suave).
func tituloCaso(s string) string {
	palabras := strings.Fields(strings.ToLower(s))
	for i, p := range palabras {
		r := []rune(p)
		if len(r) > 0 {
			r[0] = []rune(strings.ToUpper(string(r[0])))[0]
		}
		palabras[i] = string(r)
	}
	return strings.Join(palabras, " ")
}

const css = `<style>
.sub{color:#666;margin:.2rem 0 1rem}
table.horario{border-collapse:collapse;width:100%;table-layout:fixed}
table.horario th,table.horario td{border:1px solid #e6e6e6;vertical-align:top}
table.horario th{background:#f5f6f8;padding:.45rem;font-size:.85rem}
table.horario td.hora,table.horario th.hora{width:58px;background:#fafafa;color:#888;text-align:center;font-size:.78rem;vertical-align:middle}
.clase{border-radius:7px;padding:.4rem .5rem;margin:2px;color:#fff;line-height:1.25}
.clase b{display:block;font-size:.8rem}
.clase small{display:block;font-size:.72rem;opacity:.92}
.leyenda{display:flex;flex-wrap:wrap;gap:.5rem .9rem;margin-top:1rem}
.chip{display:inline-flex;align-items:center;font-size:.82rem;color:#333}
.chip i{width:12px;height:12px;border-radius:3px;display:inline-block;margin-right:.35rem}
</style>`
