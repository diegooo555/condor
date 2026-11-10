// Guarda y lee el token en un archivo local (token.json), con permisos restringidos.
package auth

import (
	"encoding/json"
	"os"
	"strings"
	"time"
)

// Archivo donde se guarda el token, al lado del programa.
const Archivo = "token.json"

// Token es lo que devuelve la UPTC más la hora en que lo obtuvimos.
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	Scope        string `json:"scope,omitempty"`
	ObtenidoUnix int64  `json:"obtenido_unix,omitempty"`
}

// Expirado dice si el access_token ya venció (con 30 s de margen).
func (t *Token) Expirado() bool {
	if t.ObtenidoUnix == 0 || t.ExpiresIn == 0 {
		return true
	}
	vence := time.Unix(t.ObtenidoUnix, 0).Add(time.Duration(t.ExpiresIn) * time.Second)
	return time.Now().After(vence.Add(-30 * time.Second))
}

// Guardar escribe el token en token.json. 0600 = solo tu usuario puede leerlo.
func Guardar(t *Token) error {
	datos, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Archivo, datos, 0o600)
}

// Cargar lee el token de token.json. Devuelve error si el archivo no existe.
func Cargar() (*Token, error) {
	datos, err := os.ReadFile(Archivo)
	if err != nil {
		return nil, err
	}
	var t Token
	if err := json.Unmarshal(datos, &t); err != nil {
		return nil, err
	}
	t.AccessToken = strings.TrimSpace(t.AccessToken)
	return &t, nil
}
