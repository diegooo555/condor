package siweb

import "encoding/json"

// Clase es una franja del horario, ya aplanada y lista para mostrar.
type Clase struct {
	Materia     string
	Dia         int // 1=Lunes ... 7=Domingo
	HoraInicio  int
	HoraFin     int
	Aula        string
	Edificio    int
	Descripcion string
	Curso       int
}

// Duracion en horas (para el alto del bloque en la grilla).
func (c Clase) Duracion() int {
	d := c.HoraFin - c.HoraInicio
	if d < 1 {
		return 1
	}
	return d
}

// estructura cruda que devuelve el BackEnd de SiWeb.
type itemHorario struct {
	ID struct {
		NroAsignatura struct {
			Nombre string `json:"nombreAsignatura"`
		} `json:"nroAsignatura"`
		Curso  int `json:"curso"`
		NroDia int `json:"nroDia"`
	} `json:"id"`
	HoraInicio int `json:"horaInicio"`
	HoraFin    int `json:"horaFin"`
	Aula       struct {
		CodigoExterno string `json:"codigoExterno"`
		Edificio      int    `json:"edificio"`
		Descripcion   string `json:"descripcion"`
	} `json:"aula"`
}

// ParsearHorario convierte el JSON de la respuesta en una lista de Clase.
func ParsearHorario(body []byte) ([]Clase, error) {
	var items []itemHorario
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}
	clases := make([]Clase, 0, len(items))
	for _, it := range items {
		clases = append(clases, Clase{
			Materia:     it.ID.NroAsignatura.Nombre,
			Dia:         it.ID.NroDia,
			HoraInicio:  it.HoraInicio,
			HoraFin:     it.HoraFin,
			Aula:        it.Aula.CodigoExterno,
			Edificio:    it.Aula.Edificio,
			Descripcion: it.Aula.Descripcion,
			Curso:       it.ID.Curso,
		})
	}
	return clases, nil
}
