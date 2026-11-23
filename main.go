// Servicio web para SiWeb UPTC (desplegable en Render).
// Una sola página: inicias sesión una vez y luego consultas con un formulario.
package main

import (
	"bytes"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"awesomeProject/internal/auth"
	"awesomeProject/internal/httpx"
	"awesomeProject/internal/siweb"
	"awesomeProject/internal/vista"
)

// version: cámbiala en cada despliegue para saber qué build está corriendo.
const version = "docente-views-1"

// sesion es un portal con su estado: el cliente OAuth, el token en vivo y el
// nombre de su cookie de login. Hay una por portal (estudiante y docente), así
// las dos sesiones conviven sin pisarse.
type sesion struct {
	portal *auth.Portal
	cookie string // nombre de la cookie que guarda el PKCE del login a medias
	nombre string // etiqueta para los mensajes
	mu     sync.Mutex
	token  *auth.Token
}

var (
	est = &sesion{portal: auth.Estudiante, cookie: "login_pkce_est", nombre: "estudiante"}
	doc = &sesion{portal: auth.Docente, cookie: "login_pkce_doc", nombre: "docente"}
)

func (s *sesion) actual() *auth.Token {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token
}

func (s *sesion) guardar(t *auth.Token) {
	s.mu.Lock()
	s.token = t
	s.mu.Unlock()
	_ = s.portal.Guardar(t)
}

func main() {
	// Comando aparte: descarga los certificados que la UPTC no envía completos.
	if len(os.Args) > 1 && os.Args[1] == "cacert" {
		cacert()
		return
	}

	// Si quedó un token guardado de antes, lo cargamos al arrancar (cada portal
	// tiene su propio archivo; si uno no existe, simplemente queda sin sesión).
	if t, err := est.portal.Cargar(); err == nil {
		est.token = t
	}
	if t, err := doc.portal.Cargar(); err == nil {
		doc.token = t
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", inicio)
	// Estudiante
	mux.HandleFunc("/login", est.login)
	mux.HandleFunc("/callback", est.callback)
	mux.HandleFunc("/horario", horario)
	mux.HandleFunc("/notas", notas)
	mux.HandleFunc("/perfil", perfil)
	mux.HandleFunc("/foto", foto)
	// Docente (mismo motor OAuth, otro client_id/redirect_uri)
	mux.HandleFunc("/login-docente", doc.login)
	mux.HandleFunc("/callback-docente", doc.callback)
	mux.HandleFunc("/horario-docente", horarioDocente)
	mux.HandleFunc("/calificaciones", calificaciones)
	mux.HandleFunc("/salud", salud)

	log.Printf("Servicio escuchando en el puerto %s (build %s)", port, version)
	log.Fatal(http.ListenAndServe(":"+port, protegido(os.Getenv("APP_PASSWORD"), mux)))
}

// --------- Páginas ---------

func inicio(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// --- Portal estudiante ---
	tEst := est.actual()
	pagHTML := cajaLogin("1. Iniciar sesión — estudiante", "/login", "/callback",
		"https://apps3.uptc.edu.co/SiWebEstudianteFrontEnd/?...code=...", estadoSesion(tEst))
	if tEst != nil {
		pagHTML += formulariosEstudiante
	}

	// --- Portal docente ---
	tDoc := doc.actual()
	pagHTML += cajaLogin("Iniciar sesión — docente", "/login-docente", "/callback-docente",
		"https://apps3.uptc.edu.co/SiWebDocenteFrontEnd/#/...code=...", estadoSesion(tDoc))
	if tDoc != nil {
		pagHTML += formulariosDocente
	}

	pagina(w, http.StatusOK, pagHTML)
}

// estadoSesion resume si el portal tiene sesión activa y cuándo vence el token.
func estadoSesion(t *auth.Token) string {
	if t == nil {
		return `<p>Todavía no has iniciado sesión.</p>`
	}
	vence := "desconocido"
	if t.ObtenidoUnix > 0 && t.ExpiresIn > 0 {
		vence = time.Unix(t.ObtenidoUnix, 0).Add(time.Duration(t.ExpiresIn) * time.Second).Format("15:04:05")
	}
	return fmt.Sprintf(`<p class="ok">Sesión activa. El token de acceso vence ~%s (se renueva solo).</p>`, vence)
}

// cajaLogin dibuja el recuadro de "abrir login + pegar URL" de un portal.
func cajaLogin(titulo, loginPath, callbackPath, placeholder, estado string) string {
	return fmt.Sprintf(`
<div class="caja"><h2>%s</h2>
  %s
  <p>Se abre el login de la UPTC en otra pestaña. Entra con tu usuario de SiWeb,
  y cuando termines (aunque salga un 502) copia la URL COMPLETA de la barra y pégala aquí:</p>
  <p><a class="boton" href="%s" target="_blank">Abrir login UPTC</a></p>
  <form method="post" action="%s">
    <input type="text" name="url" placeholder="%s" required>
    <p><button>Guardar sesión</button></p>
  </form></div>`, titulo, estado, loginPath, callbackPath, placeholder)
}

// formulariosEstudiante son las consultas disponibles una vez con sesión de estudiante.
const formulariosEstudiante = `
<div class="caja"><h2>2. Horario</h2>
  <form method="post" action="/horario">
    <p>Periodo: <input type="text" name="periodo" placeholder="263" required></p>
    <p>Código del estudiante: <input type="text" name="estudiante" placeholder="202214307" required></p>
    <p><button>Ver horario</button></p>
  </form></div>
<div class="caja"><h2>3. Historia académica (notas)</h2>
  <form method="post" action="/notas">
    <p>Código del estudiante: <input type="text" name="estudiante" placeholder="202214307" required></p>
    <p><button>Ver notas</button></p>
  </form></div>
<div class="caja"><h2>4. Mis datos (perfil)</h2>
  <form method="post" action="/perfil">
    <p>Correo UPTC: <input type="text" name="correo" placeholder="diego.sanchez20" required></p>
    <p><button>Ver datos</button></p>
  </form></div>
<div class="caja"><h2>5. Foto del carné</h2>
  <form method="post" action="/foto">
    <p>Código del estudiante: <input type="text" name="estudiante" placeholder="202322204" required></p>
    <p><button>Ver foto</button></p>
  </form></div>`

// formulariosDocente son las consultas disponibles una vez con sesión de docente.
const formulariosDocente = `
<div class="caja"><h2>Horario docente</h2>
  <form method="post" action="/horario-docente">
    <p>Periodo (idPeriodo): <input type="text" name="periodo" placeholder="253" required></p>
    <p>ID del profesor (idProfesor): <input type="text" name="profesor" placeholder="6775483" required></p>
    <p><button>Ver horario</button></p>
  </form></div>
<div class="caja"><h2>Calificaciones de un curso</h2>
  <form method="post" action="/calificaciones">
    <p>Asignatura (pAsignatura): <input type="text" name="asignatura" placeholder="8108267" required></p>
    <p>Curso (pCurso): <input type="text" name="curso" placeholder="1" required></p>
    <p>Periodo (pPeriodo): <input type="text" name="periodo" placeholder="263" required></p>
    <p>Cualitativa (cualitativa): <input type="text" name="cualitativa" placeholder="N" value="N" required></p>
    <p>Parcial (pNroParcial): <input type="text" name="parcial" placeholder="Nota 2do 50%" required>
       <br><small>Escríbelo tal cual, con espacios y %. Ej: Nota 2do 50%</small></p>
    <p><button>Ver calificaciones</button></p>
  </form></div>`

// login: arma la URL con PKCE, recuerda el verifier en una cookie y redirige.
func (s *sesion) login(w http.ResponseWriter, r *http.Request) {
	urlLogin, verifier, state := s.portal.URLLogin()
	http.SetCookie(w, &http.Cookie{
		Name:     s.cookie,
		Value:    state + "." + verifier,
		Path:     "/",
		MaxAge:   900, // 15 min para completar el login
		HttpOnly: true,
		Secure:   esHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, urlLogin, http.StatusFound)
}

// esHTTPS detecta TLS incluso detrás del proxy de Render (termina TLS y reenvía
// por HTTP con la cabecera X-Forwarded-Proto). En local (http) devuelve false.
func esHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

// callback: recibe la URL pegada, saca el code y lo cambia por el token.
func (s *sesion) callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	urlPegada := r.FormValue("url")
	code, err := auth.ExtraerCode(urlPegada)
	if err != nil {
		paginaError(w, err.Error())
		return
	}

	c, err := r.Cookie(s.cookie)
	if err != nil || c.Value == "" {
		paginaError(w, "No hay un login en curso (o ya expiró). Da clic en 'Abrir login UPTC' y pega la URL dentro de los 15 minutos.")
		return
	}
	// El code es de un solo uso: borramos la cookie apenas la tomamos.
	http.SetCookie(w, &http.Cookie{Name: s.cookie, Path: "/", MaxAge: -1})

	state, verifier, ok := strings.Cut(c.Value, ".")
	if !ok || verifier == "" {
		paginaError(w, "La sesión de login quedó incompleta. Inicia el login de nuevo.")
		return
	}
	// El state que generamos debe venir de regreso en la URL (anti-CSRF).
	if !strings.Contains(urlPegada, state) {
		paginaError(w, "La URL no corresponde a este login. Abre 'Abrir login UPTC' y pega la URL de esa misma pestaña.")
		return
	}

	t, err := s.portal.IntercambiarCode(code, verifier)
	if err != nil {
		paginaError(w, "No se pudo obtener el token: "+err.Error()+" (el código dura ~1 minuto; repite el login).")
		return
	}
	s.guardar(t)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// horario: renueva el token si venció y dibuja el horario del periodo.
func horario(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	t, err := est.tokenValido()
	if err != nil {
		paginaError(w, err.Error())
		return
	}

	periodo := r.FormValue("periodo")
	estudiante := r.FormValue("estudiante")
	cliente := siweb.Nuevo(t.AccessToken, "")
	body, codigoHTTP, err := cliente.AsignaturasInscritas(periodo, estudiante)
	if err != nil {
		paginaError(w, "Error al consultar: "+err.Error())
		return
	}

	if codigoHTTP == 401 {
		paginaError(w, "El token no fue aceptado. Inicia sesión de nuevo.")
		return
	}

	// Intentamos dibujar el horario; si el JSON no encaja, mostramos el crudo.
	clases, err := siweb.ParsearHorario(body)
	if err != nil || len(clases) == 0 {
		pagina(w, http.StatusOK, fmt.Sprintf(`
<div class="caja"><h2>Resultado (HTTP %d)</h2>
  <textarea readonly style="height:360px">%s</textarea>
  <p><a href="/">Volver</a></p></div>`, codigoHTTP, html.EscapeString(bonito(body))))
		return
	}

	titulo := fmt.Sprintf("Periodo %s · Estudiante %s", periodo, estudiante)
	pagina(w, http.StatusOK, fmt.Sprintf(`
<div class="caja"><h2>Horario</h2>%s
  <details style="margin-top:1rem"><summary>Ver JSON original</summary>
    <textarea readonly style="height:300px">%s</textarea></details>
  <p><a href="/">Volver / consultar otro</a></p></div>`,
		vista.HorarioHTML(clases, titulo), html.EscapeString(bonito(body))))
}

// notas: renueva el token si venció y dibuja la historia académica.
func notas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	t, err := est.tokenValido()
	if err != nil {
		paginaError(w, err.Error())
		return
	}

	estudiante := r.FormValue("estudiante")
	cliente := siweb.Nuevo(t.AccessToken, "")
	body, codigoHTTP, err := cliente.HistoriaAcademica(estudiante)
	if err != nil {
		paginaError(w, "Error al consultar: "+err.Error())
		return
	}
	if codigoHTTP == 401 {
		paginaError(w, "El token no fue aceptado. Inicia sesión de nuevo.")
		return
	}

	historia, err := siweb.ParsearNotas(body)
	if err != nil || len(historia.Materias) == 0 {
		pagina(w, http.StatusOK, fmt.Sprintf(`
<div class="caja"><h2>Resultado (HTTP %d)</h2>
  <textarea readonly style="height:360px">%s</textarea>
  <p><a href="/">Volver</a></p></div>`, codigoHTTP, html.EscapeString(bonito(body))))
		return
	}

	pagina(w, http.StatusOK, fmt.Sprintf(`
<div class="caja"><h2>Historia académica</h2>%s
  <p><a href="/">Volver / consultar otro</a></p></div>`,
		vista.HistoriaHTML(historia)))
}

// perfil: renueva el token si venció y dibuja los datos del estudiante.
func perfil(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	t, err := est.tokenValido()
	if err != nil {
		paginaError(w, err.Error())
		return
	}

	correo := r.FormValue("correo")
	cliente := siweb.Nuevo(t.AccessToken, "")
	body, codigoHTTP, err := cliente.Perfil(correo)
	if err != nil {
		paginaError(w, "Error al consultar: "+err.Error())
		return
	}
	if codigoHTTP == 401 {
		paginaError(w, "El token no fue aceptado. Inicia sesión de nuevo.")
		return
	}

	p, err := siweb.ParsearPerfil(body)
	if err != nil || p == nil {
		pagina(w, http.StatusOK, fmt.Sprintf(`
<div class="caja"><h2>Resultado (HTTP %d)</h2>
  <textarea readonly style="height:360px">%s</textarea>
  <p><a href="/">Volver</a></p></div>`, codigoHTTP, html.EscapeString(bonito(body))))
		return
	}

	pagina(w, http.StatusOK, fmt.Sprintf(`
<div class="caja"><h2>Datos del estudiante</h2>%s
  <p><a href="/">Volver / consultar otro</a></p></div>`,
		vista.PerfilHTML(p)))
}

// foto: renueva el token si venció y muestra la foto del carné del estudiante.
func foto(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	t, err := est.tokenValido()
	if err != nil {
		paginaError(w, err.Error())
		return
	}

	estudiante := r.FormValue("estudiante")
	cliente := siweb.Nuevo(t.AccessToken, "")
	body, codigoHTTP, err := cliente.FotoCarne(estudiante)
	if err != nil {
		paginaError(w, "Error al consultar: "+err.Error())
		return
	}
	if codigoHTTP == 401 {
		paginaError(w, "El token no fue aceptado. Inicia sesión de nuevo.")
		return
	}

	// Una respuesta válida es un JPEG (empieza con FF D8). Si no lo es (un error
	// en JSON, un vacío, etc.), mostramos el cuerpo crudo para poder depurar.
	esJPEG := len(body) >= 2 && body[0] == 0xFF && body[1] == 0xD8
	if codigoHTTP != http.StatusOK || !esJPEG {
		pagina(w, http.StatusOK, fmt.Sprintf(`
<div class="caja"><h2>Resultado (HTTP %d)</h2>
  <textarea readonly style="height:360px">%s</textarea>
  <p><a href="/">Volver</a></p></div>`, codigoHTTP, html.EscapeString(bonito(body))))
		return
	}

	pagina(w, http.StatusOK, fmt.Sprintf(`
<div class="caja"><h2>Foto del carné</h2>%s
  <p><a href="/">Volver / consultar otro</a></p></div>`,
		vista.FotoHTML(estudiante, body)))
}

// horarioDocente: renueva el token docente si venció y muestra el horario del profesor.
// Por ahora devuelve el JSON crudo; cuando conozcamos su forma armamos la vista.
func horarioDocente(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	t, err := doc.tokenValido()
	if err != nil {
		paginaError(w, err.Error())
		return
	}

	idPeriodo := r.FormValue("periodo")
	idProfesor := r.FormValue("profesor")
	cliente := siweb.Nuevo(t.AccessToken, "")
	body, codigoHTTP, err := cliente.HorarioDocente(idPeriodo, idProfesor)
	if err != nil {
		paginaError(w, "Error al consultar: "+err.Error())
		return
	}
	if codigoHTTP == 401 {
		paginaError(w, "El token no fue aceptado. Inicia sesión de nuevo.")
		return
	}

	// Reusamos la misma grilla del estudiante; si el JSON no encaja, mostramos el crudo.
	clases, err := siweb.ParsearHorarioDocente(body)
	if err != nil || len(clases) == 0 {
		pagina(w, http.StatusOK, fmt.Sprintf(`
<div class="caja"><h2>Resultado (HTTP %d)</h2>
  <textarea readonly style="height:360px">%s</textarea>
  <p><a href="/">Volver</a></p></div>`, codigoHTTP, html.EscapeString(bonito(body))))
		return
	}

	titulo := fmt.Sprintf("Periodo %s · Profesor %s", idPeriodo, idProfesor)
	pagina(w, http.StatusOK, fmt.Sprintf(`
<div class="caja"><h2>Horario docente</h2>%s
  <details style="margin-top:1rem"><summary>Ver JSON original</summary>
    <textarea readonly style="height:300px">%s</textarea></details>
  <p><a href="/">Volver / consultar otro</a></p></div>`,
		vista.HorarioHTML(clases, titulo), html.EscapeString(bonito(body))))
}

// calificaciones: renueva el token docente si venció y muestra las notas del curso.
// Por ahora devuelve el JSON crudo; cuando conozcamos su forma armamos la vista.
func calificaciones(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	t, err := doc.tokenValido()
	if err != nil {
		paginaError(w, err.Error())
		return
	}

	asignatura := r.FormValue("asignatura")
	curso := r.FormValue("curso")
	periodo := r.FormValue("periodo")
	cualitativa := r.FormValue("cualitativa")
	parcial := r.FormValue("parcial")

	cliente := siweb.Nuevo(t.AccessToken, "")
	body, codigoHTTP, err := cliente.Calificaciones(asignatura, curso, periodo, cualitativa, parcial)
	if err != nil {
		paginaError(w, "Error al consultar: "+err.Error())
		return
	}
	if codigoHTTP == 401 {
		paginaError(w, "El token no fue aceptado. Inicia sesión de nuevo.")
		return
	}

	filas, err := siweb.ParsearCalificaciones(body)
	if err != nil || len(filas) == 0 {
		pagina(w, http.StatusOK, fmt.Sprintf(`
<div class="caja"><h2>Resultado (HTTP %d)</h2>
  <textarea readonly style="height:360px">%s</textarea>
  <p><a href="/">Volver</a></p></div>`, codigoHTTP, html.EscapeString(bonito(body))))
		return
	}

	titulo := fmt.Sprintf("Asignatura %s · curso %s · periodo %s · %s", asignatura, curso, periodo, parcial)
	pagina(w, http.StatusOK, fmt.Sprintf(`
<div class="caja"><h2>Calificaciones</h2>%s
  <details style="margin-top:1rem"><summary>Ver JSON original</summary>
    <textarea readonly style="height:300px">%s</textarea></details>
  <p><a href="/">Volver / consultar otro</a></p></div>`,
		vista.CalificacionesHTML(filas, titulo), html.EscapeString(bonito(body))))
}

// tokenValido devuelve un token usable del portal, renovándolo si hace falta.
func (s *sesion) tokenValido() (*auth.Token, error) {
	t := s.actual()
	if t == nil {
		return nil, fmt.Errorf("todavía no has iniciado sesión como %s", s.nombre)
	}
	if !t.Expirado() {
		return t, nil
	}
	if t.RefreshToken == "" {
		return nil, fmt.Errorf("la sesión de %s venció. Inicia sesión de nuevo", s.nombre)
	}
	nuevo, err := s.portal.Refrescar(t.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("no se pudo renovar la sesión de %s (ya caducó en la UPTC). Inicia sesión de nuevo", s.nombre)
	}
	s.guardar(nuevo)
	return nuevo, nil
}

// --------- Utilidades ---------

const estilos = `body{font-family:system-ui,sans-serif;max-width:760px;margin:2rem auto;padding:0 1rem;line-height:1.5;color:#222}
.caja{border:1px solid #ddd;border-radius:10px;padding:1rem 1.2rem;margin:1rem 0}
a.boton,button{background:#0b5394;color:#fff;border:0;border-radius:6px;padding:.6rem 1rem;font-size:1rem;cursor:pointer;text-decoration:none;display:inline-block}
input[type=text]{width:100%;padding:.5rem;font-size:.95rem;box-sizing:border-box}
textarea{width:100%;height:110px;font-family:monospace;font-size:.8rem;box-sizing:border-box}
.error{color:#b00020}.ok{color:#0a7a2f}`

func pagina(w http.ResponseWriter, status int, contenido string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, `<!doctype html><html lang="es"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1"><title>SiWeb UPTC</title>
<style>%s</style></head><body><h1>Consulta SiWeb UPTC</h1>%s</body></html>`, estilos, contenido)
}

func paginaError(w http.ResponseWriter, msg string) {
	pagina(w, http.StatusBadRequest, `<p class="error">`+html.EscapeString(msg)+`</p><p><a href="/">Volver</a></p>`)
}

// salud dice qué build corre y si la conexión TLS con la UPTC verifica bien,
// sin necesidad de iniciar sesión. Útil para depurar despliegues en Render.
func salud(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "build: %s\n", version)

	url := "https://auth.uptc.edu.co/auth/realms/UPTC/.well-known/openid-configuration"
	resp, err := httpx.Cliente().Get(url)
	if err != nil {
		fmt.Fprintf(w, "TLS con la UPTC: ERROR\n%v\n", err)
		return
	}
	resp.Body.Close()
	fmt.Fprintf(w, "TLS con la UPTC: OK (HTTP %d)\n", resp.StatusCode)
}

// cacert se conecta a los servidores de la UPTC, sigue la cadena de certificados
// y guarda en certs/uptc.pem los intermedios/raíz que hacen falta para verificar.
func cacert() {
	hosts := []string{"auth.uptc.edu.co:443", "apps3.uptc.edu.co:443"}
	var buf bytes.Buffer
	vistos := map[string]bool{}

	for _, h := range hosts {
		conn, err := tls.Dial("tcp", h, &tls.Config{InsecureSkipVerify: true}) // solo para LEER el certificado
		if err != nil {
			fmt.Printf("No pude conectar a %s: %v\n", h, err)
			continue
		}
		cadena := conn.ConnectionState().PeerCertificates
		conn.Close()
		if len(cadena) == 0 {
			continue
		}

		// Seguimos la cadena hacia arriba usando la extensión AIA del certificado.
		actual := cadena[0]
		for i := 0; i < 6; i++ {
			if len(actual.IssuingCertificateURL) == 0 {
				break
			}
			inter, err := bajarCert(actual.IssuingCertificateURL[0])
			if err != nil {
				fmt.Printf("No pude bajar un certificado de %s: %v\n", h, err)
				break
			}
			if !vistos[string(inter.Raw)] {
				vistos[string(inter.Raw)] = true
				pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: inter.Raw})
				fmt.Println("Agregado:", inter.Subject.CommonName)
			}
			if inter.Subject.String() == inter.Issuer.String() {
				break // llegamos a la raíz (auto-firmada)
			}
			actual = inter
		}
	}

	if buf.Len() == 0 {
		fmt.Println("No se descargó ningún certificado. Revisa tu conexión y vuelve a intentar.")
		return
	}
	if err := os.MkdirAll("certs", 0o755); err != nil {
		fmt.Println("No pude crear la carpeta certs:", err)
		return
	}
	if err := os.WriteFile("certs/uptc.pem", buf.Bytes(), 0o644); err != nil {
		fmt.Println("No pude escribir certs/uptc.pem:", err)
		return
	}
	fmt.Println("\nListo: certs/uptc.pem creado. Súbelo al repositorio y vuelve a desplegar.")
}

// bajarCert descarga un certificado (DER o PEM) desde una URL y lo parsea.
func bajarCert(url string) (*x509.Certificate, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	datos, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if blk, _ := pem.Decode(datos); blk != nil {
		datos = blk.Bytes
	}
	return x509.ParseCertificate(datos)
}

func bonito(datos []byte) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, datos, "", "  "); err != nil {
		return string(datos)
	}
	return buf.String()
}

// protegido pide usuario/clave si definiste APP_PASSWORD.
func protegido(clave string, next http.Handler) http.Handler {
	if clave == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pass, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(pass), []byte(clave)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="uptc"`)
			http.Error(w, "Acceso restringido", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
