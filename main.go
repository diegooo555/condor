// Servicio web para SiWeb UPTC (desplegable en Render).
// Una sola página: inicias sesión una vez y luego consultas con un formulario.
package main

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"awesomeProject/internal/auth"
	"awesomeProject/internal/siweb"
	"awesomeProject/internal/vista"
)

var (
	mu          sync.Mutex
	pendientes  = map[string]string{} // state -> verifier (logins a medias)
	ultimoToken *auth.Token
)

func main() {
	// Si quedó un token guardado de antes, lo cargamos al arrancar.
	if t, err := auth.Cargar(); err == nil {
		ultimoToken = t
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", inicio)
	mux.HandleFunc("/login", login)
	mux.HandleFunc("/callback", callback)
	mux.HandleFunc("/horario", horario)
	mux.HandleFunc("/notas", notas)
	mux.HandleFunc("/perfil", perfil)

	log.Printf("Servicio escuchando en el puerto %s", port)
	log.Fatal(http.ListenAndServe(":"+port, protegido(os.Getenv("APP_PASSWORD"), mux)))
}

// --------- Páginas ---------

func inicio(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	mu.Lock()
	t := ultimoToken
	mu.Unlock()

	estado := `<p>Todavía no has iniciado sesión.</p>`
	formulario := ""
	if t != nil {
		vence := "desconocido"
		if t.ObtenidoUnix > 0 && t.ExpiresIn > 0 {
			vence = time.Unix(t.ObtenidoUnix, 0).Add(time.Duration(t.ExpiresIn) * time.Second).Format("15:04:05")
		}
		estado = fmt.Sprintf(`<p class="ok">Sesión activa. El token de acceso vence ~%s (se renueva solo).</p>`, vence)
		formulario = `
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
  </form></div>`
	}

	pagina(w, http.StatusOK, fmt.Sprintf(`
<div class="caja"><h2>1. Iniciar sesión</h2>
  %s
  <p>Se abre el login de la UPTC en otra pestaña. Entra con tu usuario de SiWeb,
  y cuando termines (aunque salga un 502) copia la URL COMPLETA de la barra y pégala aquí:</p>
  <p><a class="boton" href="/login" target="_blank">Abrir login UPTC</a></p>
  <form method="post" action="/callback">
    <input type="text" name="url" placeholder="https://apps3.uptc.edu.co/SiWebEstudianteFrontEnd/?...code=..." required>
    <p><button>Guardar sesión</button></p>
  </form></div>%s`, estado, formulario))
}

// login: arma la URL con PKCE, recuerda el verifier y redirige a la UPTC.
func login(w http.ResponseWriter, r *http.Request) {
	urlLogin, verifier, state := auth.URLLogin()
	mu.Lock()
	pendientes[state] = verifier
	mu.Unlock()
	http.Redirect(w, r, urlLogin, http.StatusFound)
}

// callback: recibe la URL pegada, saca el code y lo cambia por el token.
func callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	code, err := auth.ExtraerCode(r.FormValue("url"))
	if err != nil {
		paginaError(w, err.Error())
		return
	}
	// Tomamos cualquier verifier pendiente (solo hay uno a la vez en uso normal).
	mu.Lock()
	var verifier string
	for s, v := range pendientes {
		verifier = v
		delete(pendientes, s)
		break
	}
	mu.Unlock()
	if verifier == "" {
		paginaError(w, "No hay un login en curso. Da clic en 'Abrir login UPTC' primero.")
		return
	}

	t, err := auth.IntercambiarCode(code, verifier)
	if err != nil {
		paginaError(w, "No se pudo obtener el token: "+err.Error()+" (el código dura ~1 minuto; repite el login).")
		return
	}
	mu.Lock()
	ultimoToken = t
	mu.Unlock()
	_ = auth.Guardar(t)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// horario: renueva el token si venció y dibuja el horario del periodo.
func horario(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	t, err := tokenValido()
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
	t, err := tokenValido()
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
	t, err := tokenValido()
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

// tokenValido devuelve un token usable, renovándolo si hace falta.
func tokenValido() (*auth.Token, error) {
	mu.Lock()
	t := ultimoToken
	mu.Unlock()
	if t == nil {
		return nil, fmt.Errorf("todavía no has iniciado sesión")
	}
	if !t.Expirado() {
		return t, nil
	}
	if t.RefreshToken == "" {
		return nil, fmt.Errorf("la sesión venció. Inicia sesión de nuevo")
	}
	nuevo, err := auth.Refrescar(t.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("no se pudo renovar la sesión (ya caducó en la UPTC). Inicia sesión de nuevo")
	}
	mu.Lock()
	ultimoToken = nuevo
	mu.Unlock()
	_ = auth.Guardar(nuevo)
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
