// Paquete siweb: hace las peticiones al BackEnd de SiWeb de la UPTC.
package siweb

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"awesomeProject/internal/httpx"
)

const baseURL = "https://apps3.uptc.edu.co/SiWebEstudianteBackEnd"

// carnetURL es otro servicio (carnetización), que entrega la foto en JPEG.
const carnetURL = "https://servicios3.uptc.edu.co/SiCarnetizacion-service"

// docenteURL es el BackEnd del portal docente (horario del profesor).
const docenteURL = "https://servicios3.uptc.edu.co/SiWebDocenteBackEnd"

// Cliente guarda el token (y opcionalmente la cookie) para reusarlos en cada petición.
type Cliente struct {
	Token  string
	Cookie string
	http   *http.Client
}

// Nuevo crea un cliente con el token de sesión.
func Nuevo(token, cookie string) *Cliente {
	return &Cliente{
		Token:  strings.TrimSpace(token),
		Cookie: strings.TrimSpace(cookie),
		http:   httpx.Cliente(),
	}
}

// AsignaturasInscritas consulta las asignaturas inscritas de un estudiante en un periodo.
// Devuelve el cuerpo de la respuesta tal cual (JSON) y el código HTTP.
func (c *Cliente) AsignaturasInscritas(periodo, estudiante string) ([]byte, int, error) {
	// El servidor espera: busqueda=periodo==<periodo>;estudiante==<codigo>
	// url.Values se encarga de codificar los == y el ; correctamente.
	q := url.Values{}
	q.Set("busqueda", fmt.Sprintf("periodo==%s;estudiante==%s", periodo, estudiante))
	endpoint := baseURL + "/asignaturasInscritas/busqueda?" + q.Encode()

	return c.get(endpoint)
}

// HistoriaAcademica consulta las notas de todos los semestres de un estudiante.
func (c *Cliente) HistoriaAcademica(estudiante string) ([]byte, int, error) {
	q := url.Values{}
	q.Set("busqueda", estudiante)
	endpoint := baseURL + "/Inscritoscurso/busqueda/carhisnot?" + q.Encode()
	return c.get(endpoint)
}

// Perfil consulta los datos del estudiante por su correo.
// Acepta el correo completo o solo el usuario (se le agrega @uptc.edu.co).
func (c *Cliente) Perfil(correo string) ([]byte, int, error) {
	correo = strings.TrimSpace(correo)
	if !strings.Contains(correo, "@") {
		correo += "@uptc.edu.co"
	}
	q := url.Values{}
	q.Set("busqueda", correo)
	endpoint := baseURL + "/estudiante/EstudianteMail?" + q.Encode()
	return c.get(endpoint)
}

// FotoCarne descarga la foto del carné de un estudiante por su código.
// La respuesta es binaria (JPEG), no JSON: se devuelven los bytes tal cual.
func (c *Cliente) FotoCarne(estudiante string) ([]byte, int, error) {
	estudiante = strings.TrimSpace(estudiante)
	endpoint := carnetURL + "/imgFotCarne/veriFot/" + url.PathEscape(estudiante)
	return c.get(endpoint)
}

// HorarioDocente consulta el horario de un profesor en un periodo (portal docente).
func (c *Cliente) HorarioDocente(idPeriodo, idProfesor string) ([]byte, int, error) {
	q := url.Values{}
	q.Set("idPeriodo", strings.TrimSpace(idPeriodo))
	q.Set("idProfesor", strings.TrimSpace(idProfesor))
	endpoint := docenteURL + "/horario?" + q.Encode()
	return c.get(endpoint)
}

// Calificaciones consulta las notas de un curso en un parcial (portal docente).
// El parcial se pasa en limpio (ej. "Nota 2do 50%"); aquí se codifica solo.
func (c *Cliente) Calificaciones(asignatura, curso, periodo, cualitativa, parcial string) ([]byte, int, error) {
	q := url.Values{}
	q.Set("pAsignatura", strings.TrimSpace(asignatura))
	q.Set("pCurso", strings.TrimSpace(curso))
	q.Set("pPeriodo", strings.TrimSpace(periodo))
	q.Set("cualitativa", strings.TrimSpace(cualitativa))
	q.Set("pNroParcial", strings.TrimSpace(parcial))
	// La UPTC espera los espacios como %20, no como '+'. Encode() usa '+'; como
	// QueryEscape ya convierte un '+' literal en %2B, los únicos '+' que quedan
	// son espacios, así que pasarlos a %20 es seguro.
	raw := strings.ReplaceAll(q.Encode(), "+", "%20")
	endpoint := docenteURL + "/calificacion?" + raw
	return c.get(endpoint)
}

// get arma la petición con los encabezados de autenticación y la ejecuta.
func (c *Cliente) get(endpoint string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if c.Cookie != "" {
		req.Header.Set("Cookie", c.Cookie)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, res.StatusCode, err
	}
	return body, res.StatusCode, nil
}
