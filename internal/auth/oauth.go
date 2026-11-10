// Flujo OAuth 2.0 (Authorization Code + PKCE) contra Keycloak de la UPTC.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	authURL     = "https://auth.uptc.edu.co/auth/realms/UPTC/protocol/openid-connect/auth"
	tokenURL    = "https://auth.uptc.edu.co/auth/realms/UPTC/protocol/openid-connect/token"
	clientID    = "SiWebEstudiante-publico"
	redirectURI = "https://apps3.uptc.edu.co/SiWebEstudianteFrontEnd/"
	scope       = "openid profile email"
)

var cliente = &http.Client{Timeout: 15 * time.Second}

func aleatorio(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func challengeDe(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// URLLogin arma la URL de autorización con PKCE y devuelve también el verifier
// y el state (hay que recordarlos para el intercambio del código).
func URLLogin() (urlLogin, verifier, state string) {
	verifier = aleatorio(48)
	state = aleatorio(16)
	q := url.Values{
		"client_id":             {clientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"scope":                 {scope},
		"state":                 {state},
		"code_challenge":        {challengeDe(verifier)},
		"code_challenge_method": {"S256"},
	}
	return authURL + "?" + q.Encode(), verifier, state
}

// ExtraerCode saca el 'code' de la URL de regreso. También lo busca cuando quedó
// escondido dentro del parámetro redirect_uri (el caso del 502 de SiWeb).
func ExtraerCode(pegada string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(pegada))
	if err != nil || strings.TrimSpace(pegada) == "" {
		return "", errors.New("eso no parece una URL válida")
	}
	actual := u
	for i := 0; i < 4 && actual != nil; i++ {
		datos := actual.Query()
		if frag, err := url.ParseQuery(actual.Fragment); err == nil {
			for k, v := range frag {
				datos[k] = v
			}
		}
		if e := datos.Get("error"); e != "" {
			return "", fmt.Errorf("%s: %s", e, datos.Get("error_description"))
		}
		if c := datos.Get("code"); c != "" {
			return c, nil
		}
		interna := datos.Get("redirect_uri")
		if interna == "" {
			break
		}
		actual, _ = url.Parse(interna)
	}
	return "", errors.New("no encontré 'code' en esa URL. ¿Copiaste la URL completa?")
}

// IntercambiarCode cambia el código por el token.
func IntercambiarCode(code, verifier string) (*Token, error) {
	return pedirToken(url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
	})
}

// Refrescar saca un token nuevo usando el refresh_token (sin volver a iniciar sesión).
func Refrescar(refreshToken string) (*Token, error) {
	return pedirToken(url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	})
}

// pedirToken hace el POST al endpoint de token con las credenciales en el body.
func pedirToken(params url.Values) (*Token, error) {
	params.Set("client_id", clientID)
	resp, err := cliente.PostForm(tokenURL, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var datos map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&datos)

	if resp.StatusCode != http.StatusOK {
		if d, ok := datos["error_description"].(string); ok && d != "" {
			return nil, errors.New(d)
		}
		if e, ok := datos["error"].(string); ok && e != "" {
			return nil, errors.New(e)
		}
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	crudo, _ := json.Marshal(datos)
	var t Token
	if err := json.Unmarshal(crudo, &t); err != nil {
		return nil, err
	}
	t.ObtenidoUnix = time.Now().Unix()
	return &t, nil
}
