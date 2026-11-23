// Guarda y lee el token en un archivo local (token.json), con permisos restringidos.
package auth

import (
	"encoding/json"
	"os"
	"strings"
	"time"
)

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

// Guardar escribe el token en el archivo del portal. 0600 = solo tu usuario lo lee.
func (p *Portal) Guardar(t *Token) error {
	datos, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.Archivo, datos, 0o600)
}

// Cargar lee el token del archivo del portal. Devuelve error si no existe.
func (p *Portal) Cargar() (*Token, error) {
	datos, err := os.ReadFile(p.Archivo)
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
