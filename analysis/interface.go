package analysis

type SpectralAnalyser interface {
	Do(input *SimpleBuffer)
}

type ReverseComplexAnalyzer interface {
	Analyzer
	ReverseDo(input *ComplexBuffer)
}

type ProcessFunc func(input *SimpleBuffer)
