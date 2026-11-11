package vista

import (
	"encoding/base64"
	"fmt"
	"html"
)

// FotoHTML dibuja la foto del carné (JPEG) embebida como imagen.
// Se incrusta en base64 para no exponer el token en una URL de imagen.
func FotoHTML(estudiante string, jpeg []byte) string {
	datos := base64.StdEncoding.EncodeToString(jpeg)
	return fmt.Sprintf(`<style>
.foto-carne{max-width:280px;border-radius:12px;border:1px solid #ddd;display:block;margin:.6rem 0}
</style>
<p>Estudiante <b>%s</b></p>
<img class="foto-carne" src="data:image/jpeg;base64,%s" alt="Foto del carné de %s">
<p><a class="boton" download="%s.jpeg" href="data:image/jpeg;base64,%s">Descargar</a></p>`,
		html.EscapeString(estudiante), datos,
		html.EscapeString(estudiante), html.EscapeString(estudiante), datos)
}
