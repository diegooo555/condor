package siweb

import (
	"encoding/json"
	"sort"
	"strings"
)

// estructura cruda del horario docente (servicio SiWebDocenteBackEnd/horario).
// El título viene como "MATERIA-GRUPO-SALON" y hay una fila por cada hora.
type itemHorarioDocente struct {
	Title      string `json:"title"`
	StartTime  string `json:"startTime"`
	DaysOfWeek []int  `json:"daysOfWeek"`
}

// ParsearHorarioDocente convierte el JSON del portal docente en []Clase, para
// reutilizar la misma grilla del estudiante. Une las horas contiguas de una
// misma materia/día en un solo bloque y descarta la fila "Hora Maxima y Minima".
func ParsearHorarioDocente(body []byte) ([]Clase, error) {
	var items []itemHorarioDocente
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}

	type clave struct {
		dia     int
		materia string
		aula    string
		curso   int
	}
	horas := map[clave][]int{}
	for _, it := range items {
		if len(it.DaysOfWeek) == 0 || it.StartTime == "" {
			continue // fila centinela (Hora Maxima y Minima)
		}
		dia := it.DaysOfWeek[0]
		if dia < 1 || dia > 7 {
			continue
		}
		h := horaDe(it.StartTime)
		if h < 0 {
			continue
		}
		materia, aula, curso := partesTitulo(it.Title)
		k := clave{dia, materia, aula, curso}
		horas[k] = append(horas[k], h)
	}

	var clases []Clase
	for k, hs := range horas {
		sort.Ints(hs)
		ini, prev := hs[0], hs[0]
		for i := 1; i <= len(hs); i++ {
			if i < len(hs) && hs[i] == prev+1 {
				prev = hs[i]
				continue
			}
			clases = append(clases, Clase{
				Materia:    k.materia,
				Dia:        k.dia,
				HoraInicio: ini,
				HoraFin:    prev + 1,
				Aula:       k.aula,
				Curso:      k.curso,
			})
			if i < len(hs) {
				ini, prev = hs[i], hs[i]
			}
		}
	}

	// Orden estable por día y hora.
	sort.Slice(clases, func(i, j int) bool {
		if clases[i].Dia != clases[j].Dia {
			return clases[i].Dia < clases[j].Dia
		}
		return clases[i].HoraInicio < clases[j].HoraInicio
	})
	return clases, nil
}

// horaDe saca la hora entera de "08:00:00" -> 8. Devuelve -1 si no puede.
func horaDe(s string) int {
	p := strings.SplitN(s, ":", 2)
	if len(p) == 0 || p[0] == "" {
		return -1
	}
	n := 0
	for _, r := range p[0] {
		if r < '0' || r > '9' {
			return -1
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// partesTitulo separa "ALGORITMOS Y PROGRAMACION-G1-RA304" en
// (materia, aula, curso). Los dos últimos segmentos son grupo y salón.
func partesTitulo(title string) (materia, aula string, curso int) {
	partes := strings.Split(title, "-")
	if len(partes) >= 3 {
		aula = strings.TrimSpace(partes[len(partes)-1])
		grupo := strings.TrimSpace(partes[len(partes)-2])
		materia = strings.TrimSpace(strings.Join(partes[:len(partes)-2], "-"))
		return materia, aula, soloDigitos(grupo)
	}
	return strings.TrimSpace(title), "", 0
}

// soloDigitos extrae el número de "G1" -> 1 (0 si no hay dígitos).
func soloDigitos(s string) int {
	n, hay := 0, false
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
			hay = true
		}
	}
	if !hay {
		return 0
	}
	return n
}
