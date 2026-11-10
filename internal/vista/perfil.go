package vista

import (
	"fmt"
	"html"
	"strings"

	"awesomeProject/internal/siweb"
)

// PerfilHTML dibuja la tarjeta de datos del estudiante.
func PerfilHTML(p *siweb.Perfil) string {
	if p == nil {
		return `<p>No se encontró el estudiante.</p>`
	}

	var b strings.Builder
	b.WriteString(cssPerfil)

	// Encabezado: nombre, código y estado de matrícula.
	fmt.Fprintf(&b, `<div class="phead">
<div class="avatar">%s</div>
<div><b>%s</b><br><span>Código %d · %s</span></div>
<span class="badge">%s</span></div>`,
		html.EscapeString(iniciales(p.Nombre)),
		html.EscapeString(tituloCaso(p.Nombre)), p.Codigo,
		html.EscapeString(tituloCaso(p.Programa)),
		html.EscapeString(tituloCaso(p.EstadoMatricula)))

	// Secciones de datos.
	b.WriteString(seccion("Contacto", [][2]string{
		{"Correo UPTC", p.CorreoUptc},
		{"Correo personal", p.Correo},
		{"Teléfono", p.Telefono},
		{"Dirección", tituloCaso(p.Direccion)},
	}))
	b.WriteString(seccion("Académico", [][2]string{
		{"Programa", tituloCaso(p.Programa)},
		{"Facultad", tituloCaso(p.Facultad)},
		{"Sede", tituloCaso(p.Sede)},
		{"Modalidad", p.Modalidad},
		{"Jornada", p.Jornada},
		{"Plan de estudios", fmt.Sprintf("%d", p.Plan)},
	}))
	b.WriteString(seccion("Identificación", [][2]string{
		{"Documento", p.TipoIdent},
		{"Número", p.NumeroIdent},
		{"Sexo", p.Sexo},
		{"Estado civil", p.EstadoCivil},
		{"Estrato", p.Estrato},
	}))

	return b.String()
}

// seccion arma un bloque con título y una grilla de campo/valor.
func seccion(titulo string, campos [][2]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<h3 class="psec">%s</h3><div class="pgrid">`, html.EscapeString(titulo))
	for _, c := range campos {
		valor := c[1]
		if strings.TrimSpace(valor) == "" {
			valor = "-"
		}
		fmt.Fprintf(&b, `<div class="campo"><span>%s</span><b>%s</b></div>`,
			html.EscapeString(c[0]), html.EscapeString(valor))
	}
	b.WriteString(`</div>`)
	return b.String()
}

// iniciales saca las dos primeras iniciales del nombre para el avatar.
func iniciales(nombre string) string {
	ps := strings.Fields(nombre)
	ini := ""
	for _, p := range ps {
		if len(p) > 0 {
			ini += strings.ToUpper(string([]rune(p)[0]))
		}
		if len(ini) == 2 {
			break
		}
	}
	return ini
}

const cssPerfil = `<style>
.phead{display:flex;align-items:center;gap:.9rem;margin-bottom:1.2rem;flex-wrap:wrap}
.phead b{font-size:1.1rem}
.phead span{color:#666;font-size:.9rem}
.avatar{width:52px;height:52px;border-radius:50%;background:#0b5394;color:#fff;display:flex;align-items:center;justify-content:center;font-weight:600;font-size:1.1rem}
.badge{margin-left:auto;background:#e6f4ea;color:#0a7a2f;border-radius:20px;padding:.3rem .8rem;font-size:.8rem;font-weight:600}
h3.psec{margin:1.2rem 0 .5rem;font-size:.95rem;color:#333;border-left:4px solid #0b5394;padding-left:.5rem}
.pgrid{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:.6rem}
.campo{border:1px solid #eee;border-radius:8px;padding:.5rem .7rem}
.campo span{display:block;color:#888;font-size:.74rem}
.campo b{font-size:.92rem;font-weight:500;word-break:break-word}
</style>`
