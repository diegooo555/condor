package siweb

import "encoding/json"

// Perfil son los datos del estudiante, ya legibles.
type Perfil struct {
	Nombre          string
	Codigo          int
	TipoIdent       string
	NumeroIdent     string
	Sexo            string
	EstadoCivil     string
	Estrato         string
	Direccion       string
	Telefono        string
	Correo          string
	CorreoUptc      string
	Programa        string
	Facultad        string
	Sede            string
	Modalidad       string
	Jornada         string
	Plan            int
	EstadoMatricula string
}

type itemPerfil struct {
	NroEstudiante int    `json:"nroEstudiante"`
	NroIdenEst    string `json:"nroIdenEst"`
	TipoIdenEst   string `json:"tipoIdenEst"`
	Apellido      string `json:"apellidoEstudiante"`
	Nombre        string `json:"nombreEstudiante"`
	Ecivil        string `json:"ecivilEstudiante"`
	Direccion     string `json:"direccion"`
	Sexo          string `json:"sexo"`
	Telefono      string `json:"telefono"`
	Mail          string `json:"direccionMail"`
	Estrato       *int   `json:"estrato"`
	MailUptc      string `json:"direccionMailUptc"`
	Programa      struct {
		NroPlanVigente int `json:"nroPlanVigente"`
		Nomprog        struct {
			NombrePrograma string `json:"nombrePrograma"`
		} `json:"nomprog"`
		Facultades struct {
			NombreFacultad string `json:"nombreFacultad"`
		} `json:"facultades"`
		Sede struct {
			Nombre string `json:"nmbreSede"`
		} `json:"sede"`
		Modalidad string `json:"modalidad"`
		Jornada   string `json:"jornada"`
	} `json:"programa"`
	Estado struct {
		NombreEstado string `json:"nombreEstado"`
	} `json:"estado"`
}

// diccionarios para pasar códigos a palabras.
var (
	sexos       = map[string]string{"M": "Masculino", "F": "Femenino"}
	eciviles    = map[string]string{"S": "Soltero(a)", "C": "Casado(a)", "U": "Unión libre", "D": "Divorciado(a)", "V": "Viudo(a)"}
	modalidades = map[string]string{"P": "Presencial", "D": "Distancia", "V": "Virtual"}
	jornadas    = map[string]string{"D": "Diurna", "N": "Nocturna", "M": "Mixta"}
	tiposIdent  = map[string]string{"C": "Cédula de ciudadanía", "T": "Tarjeta de identidad", "E": "Cédula de extranjería", "P": "Pasaporte"}
)

func traducir(m map[string]string, k string) string {
	if v, ok := m[k]; ok {
		return v
	}
	return k
}

// ParsearPerfil convierte la respuesta (un arreglo de 1) en un Perfil.
func ParsearPerfil(body []byte) (*Perfil, error) {
	var items []itemPerfil
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	it := items[0]

	estrato := "-"
	if it.Estrato != nil {
		estrato = itoa(*it.Estrato)
	}

	return &Perfil{
		Nombre:          it.Nombre + " " + it.Apellido,
		Codigo:          it.NroEstudiante,
		TipoIdent:       traducir(tiposIdent, it.TipoIdenEst),
		NumeroIdent:     it.NroIdenEst,
		Sexo:            traducir(sexos, it.Sexo),
		EstadoCivil:     traducir(eciviles, it.Ecivil),
		Estrato:         estrato,
		Direccion:       it.Direccion,
		Telefono:        it.Telefono,
		Correo:          it.Mail,
		CorreoUptc:      it.MailUptc,
		Programa:        it.Programa.Nomprog.NombrePrograma,
		Facultad:        it.Programa.Facultades.NombreFacultad,
		Sede:            it.Programa.Sede.Nombre,
		Modalidad:       traducir(modalidades, it.Programa.Modalidad),
		Jornada:         traducir(jornadas, it.Programa.Jornada),
		Plan:            it.Programa.NroPlanVigente,
		EstadoMatricula: it.Estado.NombreEstado,
	}, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	if neg {
		d = append([]byte{'-'}, d...)
	}
	return string(d)
}
