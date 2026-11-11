// Paquete httpx: cliente HTTP compartido que arregla el problema de la UPTC
// (su servidor no envía el certificado intermedio completo). El programa
// descarga ese intermedio solo, siguiendo la pista que trae el certificado,
// y verifica normalmente contra las raíces del sistema. La seguridad queda intacta.
package httpx

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	compartido *http.Client
	iniciar    sync.Once

	mu          sync.Mutex
	raices      *x509.CertPool       // raíces de confianza (sistema + extras)
	intermedios = x509.NewCertPool() // intermedios que vamos descargando (cache)

	descargador = &http.Client{Timeout: 10 * time.Second} // para bajar certificados
)

// Cliente devuelve (y reutiliza) el cliente HTTP ya configurado.
func Cliente() *http.Client {
	iniciar.Do(construir)
	return compartido
}

func construir() {
	p, err := x509.SystemCertPool()
	if err != nil || p == nil {
		p = x509.NewCertPool()
	}
	// Extras opcionales: variable UPTC_CA_PEM y archivos en certs/ (no obligatorios).
	if pemEnv := os.Getenv("UPTC_CA_PEM"); pemEnv != "" {
		p.AppendCertsFromPEM([]byte(pemEnv))
	}
	if entradas, err := os.ReadDir("certs"); err == nil {
		for _, e := range entradas {
			if ext := filepath.Ext(e.Name()); ext == ".pem" || ext == ".crt" {
				if b, err := os.ReadFile(filepath.Join("certs", e.Name())); err == nil {
					p.AppendCertsFromPEM(b)
				}
			}
		}
	}
	raices = p

	compartido = &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// Hacemos la verificación nosotros (abajo), completando los
				// certificados intermedios que la UPTC no envía. NO es inseguro:
				// seguimos validando el dominio y la cadena contra raíces reales.
				InsecureSkipVerify: true,
				VerifyConnection:   verificar,
			},
		},
	}
}

func verificar(cs tls.ConnectionState) error {
	if len(cs.PeerCertificates) == 0 {
		return fmt.Errorf("el servidor no envió certificado")
	}
	hoja := cs.PeerCertificates[0]

	// intenta verificar con los intermedios que ya tenemos.
	intentar := func() error {
		mu.Lock()
		inter := intermedios.Clone()
		mu.Unlock()
		for _, c := range cs.PeerCertificates[1:] {
			inter.AddCert(c)
		}
		_, err := hoja.Verify(x509.VerifyOptions{
			DNSName:       cs.ServerName, // valida el dominio (seguridad intacta)
			Roots:         raices,
			Intermediates: inter,
		})
		return err
	}

	if err := intentar(); err == nil {
		return nil
	}

	// Falta algún intermedio: lo descargamos siguiendo la extensión AIA
	// (una URL que el propio certificado indica) y reintentamos.
	actual := hoja
	for i := 0; i < 6; i++ {
		if len(actual.IssuingCertificateURL) == 0 {
			break
		}
		c, err := bajarCert(actual.IssuingCertificateURL[0])
		if err != nil {
			break
		}
		mu.Lock()
		intermedios.AddCert(c)
		mu.Unlock()
		if err := intentar(); err == nil {
			return nil
		}
		if c.Subject.String() == c.Issuer.String() {
			break // llegamos a la raíz
		}
		actual = c
	}
	return fmt.Errorf("no se pudo verificar el certificado de %s (cadena incompleta)", cs.ServerName)
}

// bajarCert descarga un certificado (DER o PEM) desde una URL y lo parsea.
func bajarCert(url string) (*x509.Certificate, error) {
	resp, err := descargador.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	datos, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if blk, _ := pem.Decode(datos); blk != nil {
		datos = blk.Bytes
	}
	return x509.ParseCertificate(datos)
}
