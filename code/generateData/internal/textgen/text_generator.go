package textgen

import (
    "math/rand"
    "strings"
)

type Generator struct {
    RNG *rand.Rand
}

var openings = []string{
    "Presento una queja",
    "Interpongo un reclamo",
    "Deseo registrar una denuncia",
    "Quiero reportar un problema",
}

var requests = []string{
    "Solicito una investigación.",
    "Pido atención inmediata.",
    "Requiero solución urgente.",
}

func (g *Generator) GenerateNormal(entity string, materia string) string {

    a := openings[g.RNG.Intn(len(openings))]
    b := requests[g.RNG.Intn(len(requests))]

    text := a + " contra " + entity + " por problemas relacionados con " + materia + ". " + b

    return g.AddNoise(text)
}

func (g *Generator) AddNoise(text string) string {

    if g.RNG.Float64() < 0.15 {
        text = strings.ReplaceAll(text, "ción", "cion")
    }

    if g.RNG.Float64() < 0.10 {
        text += "!!"
    }

    if g.RNG.Float64() < 0.08 {
        text = strings.ToUpper(text)
    }

    return text
}

func (g *Generator) GenerateDuplicateVariant(base string) string {

    variants := []func(string) string{
        strings.ToUpper,
        strings.ToLower,
        func(s string) string {
            return s + " urgente"
        },
        func(s string) string {
            return strings.ReplaceAll(s, "Solicito", "Requiero")
        },
    }

    f := variants[g.RNG.Intn(len(variants))]

    return f(base)
}