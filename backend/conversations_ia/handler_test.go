package conversations_ia

import (
	"strings"
	"testing"
)

func TestNormalizeMarkdownFull(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Headers pegados a texto",
			input:    "El tumor de Frantz es raro.## Definición",
			expected: "El tumor de Frantz es raro.\n\n## Definición\n",
		},
		{
			name:     "Headers consecutivos sin espacio",
			input:    "# Frantz Tumor## Resumen## Análisis",
			expected: "# Frantz Tumor\n\n## Resumen\n\n## Análisis\n",
		},
		{
			name:     "Bullets pegados a texto",
			input:    "Características:- Representa el 1%-3%- Es benigno",
			expected: "Características:\n- Representa el 1%-3%\n- Es benigno\n",
		},
		{
			name:     "Preservar guiones internos",
			input:    "El tratamiento médico-quirúrgico es efectivo.",
			expected: "El tratamiento médico-quirúrgico es efectivo.\n",
		},
		{
			name:     "Listas numeradas pegadas",
			input:    "Pasos:1. Diagnóstico2. Tratamiento",
			expected: "Pasos:\n1. Diagnóstico\n2. Tratamiento\n",
		},
		{
			name:     "Headers con saltos después",
			input:    "## Resumen\nEl tumor de Frantz",
			expected: "## Resumen\n\nEl tumor de Frantz\n",
		},
		{
			name:     "Normalizar múltiples saltos",
			input:    "Texto 1\n\n\n\nTexto 2",
			expected: "Texto 1\n\nTexto 2\n",
		},
		{
			name:     "Caso completo realista",
			input:    "# Frantz Tumor: Definición## Resumen\n- El tumor de Frantz es raro.- Representa 1%-3%.## Análisis\nEl tumor se origina en células pancreáticas.## Fuentes",
			expected: "# Frantz Tumor: Definición\n\n## Resumen\n\n- El tumor de Frantz es raro.\n- Representa 1%-3%.\n\n## Análisis\n\nEl tumor se origina en células pancreáticas.\n\n## Fuentes\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeMarkdownFull(tt.input)
			if result != tt.expected {
				t.Errorf("\nNombre: %s\nInput:\n%q\n\nEsperado:\n%q\n\nObtenido:\n%q\n\nDiff visual:\nEsperado: %s\nObtenido:  %s",
					tt.name,
					tt.input,
					tt.expected,
					result,
					strings.ReplaceAll(tt.expected, "\n", "⏎"),
					strings.ReplaceAll(result, "\n", "⏎"),
				)
			}
		})
	}
}

func TestNormalizeMarkdownFullPreservesValidFormatting(t *testing.T) {
	// Verificar que texto ya bien formateado no se rompa
	input := `# Título Principal

## Sección 1

Este es un párrafo normal.

- Item 1
- Item 2

## Sección 2

Otro párrafo.

1. Primera opción
2. Segunda opción

## Fuentes

Referencias aquí.
`
	result := normalizeMarkdownFull(input)

	// El resultado debe mantener la estructura (permitiendo limpieza final)
	if !strings.Contains(result, "# Título Principal") {
		t.Error("Se perdió el título principal")
	}
	if !strings.Contains(result, "## Sección 1") {
		t.Error("Se perdió una sección")
	}
	if !strings.Contains(result, "- Item 1") {
		t.Error("Se perdieron los bullets")
	}
	if !strings.Contains(result, "1. Primera opción") {
		t.Error("Se perdió la lista numerada")
	}

	// No debe tener más de 2 saltos consecutivos
	if strings.Contains(result, "\n\n\n") {
		t.Error("Hay más de 2 saltos de línea consecutivos")
	}
}
