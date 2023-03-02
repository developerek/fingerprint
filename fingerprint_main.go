package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/developerek/fingerprint/audiomatcher"
	"github.com/developerek/fingerprint/fingerprint"
	"github.com/developerek/fingerprint/lookup"
	"github.com/developerek/fingerprint/pcm"
	"github.com/developerek/fingerprint/spectral"
)

func loadFiles(filenames []string, analyser spectral.Analyser, optVerbose bool) (matches lookup.Matches, err error) {

	matches = lookup.New()

	for _, filename := range filenames {
		fmt.Printf("Processing fingerprints for %s...\n", filename)
		stream, err := pcm.NewFileStream(filename, fingerprint.SAMPLE_RATE, fingerprint.BLOCK_SIZE)
		if err != nil {
			return nil, err
		}

		matches, err = loadStream(filename, stream, matches, analyser, optVerbose)

		stream.Close()
	}

	return matches, nil
}

func listen(stream pcm.StartReader, matcher *audiomatcher.AudioMatcher, analyser spectral.Analyser, optVerbose bool) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)

	if err := stream.Start(); err != nil {
		return fmt.Errorf("Error starting microphone recording: %s", err)
	}

	fmt.Printf("Listening for %d fingerprints.  Press Ctrl-C to stop\n", len(matcher.FingerprintLib))
	var count int = 0
	for {
		count = count + 1
		frame, err := stream.Read()
		fmt.Printf("count:%d\n", count)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			log.Fatalf("Error reading microphone: %s", err)
			fmt.Printf("Error reading microphone: %s", err)
		}

		fp := fingerprint.Generate(analyser, frame.AsFloat64(), fingerprint.MIC_SILENCE_THRESHOLD)

		if fp != nil {

			printStatus(fp, frame, optVerbose)

			matcher.Register(fingerprint.Hash(fp.Fingerprint()), frame.Timestamp())

			// Check every second to see if they are certain enough to be a match
			if frame.BlockId()%fingerprint.BLOCKS_PER_SECOND == 0 {
				log.Printf("(%.2f) %s\n", frame.Timestamp(), matcher.Stats())
				fmt.Printf("(%.2f) %s\n", frame.Timestamp(), matcher.Stats())
				//hits := matcher.GetHits()
				//if len(hits) > 0 {
				//fmt.Println(hits)
				//}
			}
		}

		select {
		case <-sig:
			return nil
		default:
		}
	}

}
func readDir(dirname string, ext string) ([]string, error) {

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	fmt.Println(exPath)

	f, err := os.Open(exPath + "/mp3")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	files, err := f.Readdir(0)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	var fileList []string = make([]string, 0)

	for _, v := range files {

		if filepath.Ext(v.Name()) != ext {
			continue
		}
		fmt.Println(exPath + "/mp3/" + v.Name())
		fileList = append(fileList, exPath+"/mp3/"+v.Name())
	}
	return fileList, nil
}
func loadStream(filename string, stream pcm.Reader, matches lookup.Matches, analyser spectral.Analyser, optVerbose bool) (lookup.Matches, error) {
	clashCount, fpCount := 0, 0
	for {
		frame, err := stream.Read()
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return matches, err
		}

		fp := fingerprint.Generate(analyser, frame.AsFloat64(), fingerprint.FILE_SILENCE_THRESHOLD)

		printStatus(fp, frame, optVerbose)

		if fp != nil {
			fpCount++
			/*			if _, ok := matches[string(fp.Fingerprint())]; ok {
							clashCount++
						}
						matches[string(fp.Fingerprint())] = audiomatcher.Match{filename, frame.Timestamp()}
			*/
			key := fingerprint.Hash(fp.Fingerprint())
			if _, ok := matches.Lookup(key); ok {
				clashCount++
			}
			matches.Add(key, filename, frame.Timestamp())
		}
	}

	log.Printf("%s:\tFingerprints %d, hash clashes: %d\n", filename, fpCount, clashCount)

	return matches, nil
}

func printStatus(fp fmt.Stringer, frame *pcm.Frame, verbose bool) {
	if verbose {
		header := fmt.Sprintf("[%4d:%6.2f]", frame.BlockId(), frame.Timestamp())
		if fp == nil {
			fmt.Printf("%s fp: nil\n", header)
		} else {
			//fmt.Printf("%s %s\n", header, fp.Candidates)
			fmt.Printf("%s %s\n", header, fp)
			//fmt.Printf("%s -> Key: %v\n\n", header, fp.Fingerprint())
		}
	}
}

func main() {
	var optVerbose bool
	var optAnalyser, optInput string
	var analyser spectral.Analyser

	flag.BoolVar(&optVerbose, "verbose", false, "Verbose output of spectral analysis data")
	flag.StringVar(&optAnalyser, "analyser", "bespoke", "Spectral analyser to use (pwelch | bespoke)")
	flag.StringVar(&optInput, "input", "", "Input file to use instead of microphone")

	flag.Parse()

	switch optAnalyser {
	case "bespoke":
		analyser = spectral.Amplitude
	case "pwelch":
		analyser = spectral.Pwelch
	default:
		flag.PrintDefaults()
		log.Fatalf("Unrecognised spectral analyser requested: '%s'", optAnalyser)

	}

	if len(flag.Args()) == 10 {
		log.Println("Error: No audio files found to match against")
		flag.PrintDefaults()
		os.Exit(1)
	}

	filenames, _ := readDir("/mp3/", ".mp3") //[]string{"20230224-021127.mp3"} //flag.Args()

	fmt.Printf("Using '%s' analysis to generate fingerprints for %v\n", optAnalyser, filenames)

	fingerprints, err := loadFiles(filenames, analyser, optVerbose)
	if err != nil {
		log.Fatalf("Fatal Error generating fingerprints: %s", err)
	}

	var input pcm.StartReader
	if optInput != "" {
		input, err = pcm.NewFileStream(optInput, fingerprint.SAMPLE_RATE, fingerprint.BLOCK_SIZE)
	} else {
		input, err = pcm.NewMicStream(fingerprint.SAMPLE_RATE, fingerprint.BLOCK_SIZE)
	}
	if err != nil {
		log.Fatalf("Fatal Error opening stream: %s", err)
	}

	matcher := audiomatcher.New(fingerprints, fingerprint.TIME_DELTA_THRESHOLD)

	fmt.Printf("++++++Using '%s' analysis to generate fingerprints for %d\n", optAnalyser, len(filenames))

	err = listen(input, matcher, analyser, optVerbose)
	if err != nil {
		log.Fatalf("Fatal Error listening to stream: %s", err)
	}
	fmt.Println(matcher)
	fmt.Println(matcher.Stats())
}
