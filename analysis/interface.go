package analysis

import "github.com/developerek/fingerprint/spectral"

type SpectralAnalyser func(samples []float64, silenceThreshold float64) spectral.Spectra

type NewSpectralAnalyser interface {
}
type ReverseComplexAnalyzer interface {
}
