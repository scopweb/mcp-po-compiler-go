Objetivo
- Construir un MCP en Go que compile archivos .po a .mo al estilo de Poedit para proyectos WordPress, de modo que Claude Desktop pueda usarlo sin requerir Poedit instalado.

Alcance inicial
- Ingesta de .po (UTF-8) y generación de .mo compatibles con gettext/WordPress.
- Validación básica: cabeceras obligatorias, conteo de mensajes, flags fuzzy, plural-forms.
- Opcional (fase 2): merge de plantillas .pot, normalización y ordenamiento determinista.

Entregables
- Servidor MCP en Go (binario y código) con manifest y capabilities declaradas para Claude Desktop.
- Herramientas expuestas: compilar .po a .mo, validar .po, mostrar resumen de catálogos.
- Pruebas automatizadas y muestras .po/.mo de referencia.

Diseño de alto nivel
- Protocolo MCP: definir manifest (tools, auth none, rate limits), schema de requests/responses y rutas de archivos de trabajo.
- Núcleo gettext: usar una librería Go probada para leer/escribir .po/.mo (p.ej. github.com/leonelquinteros/gotext o github.com/vorlif/spreak) o implementar escritor .mo sencillo si dependencias son pesadas.
- Flujo principal: recibir payload (contenido .po o ruta), parsear, validar, compilar a .mo en memoria y devolver blob/base64 o ruta en disco, con hash para deduplicación.
- Seguridad: limitar acceso a rutas fuera del workspace, tamaño máximo de archivos, rechazar inputs binarios.

Plan de trabajo
1) Investigación rápida (half-day)
	- Revisar librerías Go actuales para .po/.mo (lectura, escritura, plural forms).
	- Confirmar requisitos de compatibilidad WordPress (endianness, cabecera project-id-version, language-team, po-revision-date, plural-forms).
	- Definir si el output se entrega como archivo temporal o como bytes/base64 en la respuesta MCP.

2) Esqueleto MCP (1 día)
	- Generar estructura Go (cmd/server, internal/po, internal/mcp, testdata/).
	- Implementar manifest.json para Claude Desktop con tools: compile_po, validate_po, summarize_po.
	- Wiring de logging, manejo de errores y límites de tamaño.

3) Núcleo gettext (1-2 días)
	- Implementar parser/loader .po (o integrar librería) y escritor .mo determinista.
	- Validaciones: cabeceras requeridas, conteo plural vs. plural-forms, flags fuzzy, entradas sin msgstr.
	- Pruebas con fixtures reales (extraer de un .po WordPress y uno minimal propio).

4) Herramientas MCP (1 día)
	- compile_po: recibe contenido .po (texto) y devuelve .mo (bytes/base64) + metadata (messages, language, hash).
	- validate_po: devuelve lista de warnings/errores y métricas (total, traducidos, fuzzy, vacíos).
	- summarize_po: retorna cabeceras clave y progreso (para UI rápida en Claude).

5) Integración Claude Desktop (0.5 día)
	- Probar el MCP localmente con Claude Desktop, verificar tools y respuestas.
	- Ajustar manifest (descripciones, parámetros) y mensajes de error legibles.

6) Calidad y empaquetado (0.5-1 día)
	- Añadir pruebas adicionales y casos límite (plurals complejos, contextos msgctxt, cadenas vacías).
	- Licencias y README de uso; binario build reproducible; CI básico (go test ./...).

Riesgos y mitigaciones
- Parsers .po inestables o incompletos: preferir librería mantenida; si no, cubrir con tests propios.
- Variaciones de plural-forms en WordPress: incluir tabla mínima de ejemplos y validación flexible.
- Tamaño de archivos grandes: establecer límites y streaming opcional en el futuro.

Definición de listo (DoD)
- Todas las tools MCP funcionan desde Claude Desktop, compilando y validando .po reales de WordPress.
- .mo generado coincide con msgfmt/Poedit en hash para casos de prueba.
- Tests verdes y documentación mínima (uso, tools, ejemplos de requests/responses).
