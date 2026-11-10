package siweb

import "encoding/json"

// Nota para aprobar en la UPTC (escala 0.0 - 5.0).
const NotaMinima = 3.0

// Materia es una asignatura cursada con su nota, ya lista para mostrar.
type Materia struct {
	Nombre     string
	Periodo    string // "PRIMER SEMESTRE DE 2022"
	NroPeriodo int    // para ordenar cronológicamente
	Creditos   int
	P50        *float64
	S50        *float64
	Habilitada *float64
	Final      *float64
	Estado     string
}

// Aprobada dice si pasó la materia.
func (m Materia) Aprobada() bool {
	return m.Final != nil && *m.Final >= NotaMinima
}

// Estudiante: solo lo académico (sin datos personales sensibles).
type Estudiante struct {
	Nombre   string
	Codigo   int
	Programa string
	Facultad string
	Sede     string
}

// HistoriaAcademica es el resultado completo.
type HistoriaAcademica struct {
	Estudiante Estudiante
	Materias   []Materia
}

// PromedioGeneral: promedio ponderado por créditos de todas las materias con nota.
func (h *HistoriaAcademica) PromedioGeneral() float64 {
	var suma float64
	var creditos int
	for _, m := range h.Materias {
		if m.Final != nil {
			suma += *m.Final * float64(m.Creditos)
			creditos += m.Creditos
		}
	}
	if creditos == 0 {
		return 0
	}
	return suma / float64(creditos)
}

// CreditosAprobados suma los créditos de las materias aprobadas.
func (h *HistoriaAcademica) CreditosAprobados() int {
	var total int
	for _, m := range h.Materias {
		if m.Aprobada() {
			total += m.Creditos
		}
	}
	return total
}

// CreditosCursados suma los créditos de todas las materias.
func (h *HistoriaAcademica) CreditosCursados() int {
	var total int
	for _, m := range h.Materias {
		total += m.Creditos
	}
	return total
}

// --- parseo del JSON crudo ---

type itemNota struct {
	ID struct {
		NroAsignatura struct {
			Nombre string `json:"nombreAsignatura"`
		} `json:"nroAsignatura"`
		Periodo struct {
			NroPeriodo  int    `json:"nroPeriodo"`
			Descripcion string `json:"descripcion"`
		} `json:"periodo"`
		NroEstudiante struct {
			Nombre        string `json:"nombreEstudiante"`
			Apellido      string `json:"apellidoEstudiante"`
			NroEstudiante int    `json:"nroEstudiante"`
			Programa      struct {
				Nomprog struct {
					NombrePrograma string `json:"nombrePrograma"`
				} `json:"nomprog"`
				Facultades struct {
					NombreFacultad string `json:"nombreFacultad"`
				} `json:"facultades"`
				Sede struct {
					Nombre string `json:"nmbreSede"`
				} `json:"sede"`
			} `json:"programa"`
		} `json:"nroEstudiante"`
	} `json:"id"`
	P50      *int   `json:"notaParcialP50"`
	S50      *int   `json:"notaParcialS50"`
	Hab      *int   `json:"notaHabilitacion"`
	Final    *int   `json:"notaFinal"`
	Creditos int    `json:"totalCreditos"`
	Estado   string `json:"estado"`
}

// sobre10 convierte la nota entera (0-50) a escala 0.0-5.0.
func sobre10(v *int) *float64 {
	if v == nil {
		return nil
	}
	f := float64(*v) / 10.0
	return &f
}

// ParsearNotas convierte el JSON de la respuesta en una HistoriaAcademica.
func ParsearNotas(body []byte) (*HistoriaAcademica, error) {
	var items []itemNota
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}
	h := &HistoriaAcademica{}
	for i, it := range items {
		if i == 0 {
			e := it.ID.NroEstudiante
			h.Estudiante = Estudiante{
				Nombre:   e.Nombre + " " + e.Apellido,
				Codigo:   e.NroEstudiante,
				Programa: e.Programa.Nomprog.NombrePrograma,
				Facultad: e.Programa.Facultades.NombreFacultad,
				Sede:     e.Programa.Sede.Nombre,
			}
		}
		h.Materias = append(h.Materias, Materia{
			Nombre:     it.ID.NroAsignatura.Nombre,
			Periodo:    it.ID.Periodo.Descripcion,
			NroPeriodo: it.ID.Periodo.NroPeriodo,
			Creditos:   it.Creditos,
			P50:        sobre10(it.P50),
			S50:        sobre10(it.S50),
			Habilitada: sobre10(it.Hab),
			Final:      sobre10(it.Final),
			Estado:     it.Estado,
		})
	}
	return h, nil
}
