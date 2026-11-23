package siweb

import "encoding/json"

// Calificacion es la fila de notas de un estudiante en un curso, ya legible.
type Calificacion struct {
	Codigo       int
	Nombre       string
	Curso        string
	Primer50     string
	Segundo50    string
	Habilitacion string
	Final        string
}

type itemCalificacion struct {
	NroEstudiante    int    `json:"nroEstudiante"`
	NombreEstudiante string `json:"nombreEstudiante"`
	NroCurso         string `json:"nroCurso"`
	NotaP50          string `json:"notaParcialP50"`
	NotaS50          string `json:"notaParcialS50"`
	NotaHab          string `json:"notaHabilitacion"`
	NotaFinal        string `json:"notaFinal"`
}

// ParsearCalificaciones convierte el JSON del curso en una lista de Calificacion.
func ParsearCalificaciones(body []byte) ([]Calificacion, error) {
	var items []itemCalificacion
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}
	filas := make([]Calificacion, 0, len(items))
	for _, it := range items {
		filas = append(filas, Calificacion{
			Codigo:       it.NroEstudiante,
			Nombre:       it.NombreEstudiante,
			Curso:        it.NroCurso,
			Primer50:     it.NotaP50,
			Segundo50:    it.NotaS50,
			Habilitacion: it.NotaHab,
			Final:        it.NotaFinal,
		})
	}
	return filas, nil
}
