package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/grafov/m3u8"
)

var (
	subsample = flag.Int("subsample", 5, "What vmaf subsampling factor to use")
	threads   = flag.Int("threads", 5, "How many threads used to run vmaf")
	model     = flag.String("model", "vmaf_v0.6.1.pkl", "vmaf model to use")
	dataFile  = flag.String("datafile", "data.json", "Location of the data file to use for processing")
)

// DataFile represents the current environment data
// Resolutions are represented by *widths* in 16-pixel buckets
// Bandwidths are represented by *kbps* in 100Kbps buckets
type DataFile struct {
	ResolutionPcts []float64 `json:"resolution_pcts"`
	BandwidthPcts  []float64 `json:"bandwidth_pcts"`
}

type FFProbeOutput struct {
	Streams []*FFProbeStream `json:"streams"`
}

type FFProbeStream struct {
	Width  uint64 `json:"width"`
	Height uint64 `json:"height"`
}

const resolutionsLen = 120
const bandwidthsLen = 100

func sumFloat64Array(in []float64) float64 {
	result := float64(0.0)
	for _, val := range in {
		result += val
	}
	return result
}

func probeFile(filename string) (*FFProbeOutput, error) {
	probecmd := exec.Command("ffprobe", "-print_format", "json", "-show_streams", "-select_streams", "v:0", filename)
	stdoutData, err := probecmd.Output()
	fmt.Printf("Probe output: %s\n", string(stdoutData))
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("Error running probe: %s", exitErr.Stderr)
		}
		return nil, fmt.Errorf("Unexpected error running probe: %v", err)
	}

	var probe FFProbeOutput
	err = json.Unmarshal(stdoutData, &probe)
	if err != nil {
		fmt.Printf("Failed to unmarshal probe response: '%v'\n", err)
		return nil, fmt.Errorf("Failed to unmarshal probe response: '%v'", err)
	}
	return &probe, nil
}

func WidthToHeight(width, mezzanineWidth, mezzanineHeight uint64) uint64 {
	return 0
}

func main() {
	flag.Parse()

	if len(flag.Args()) != 2 {
		fmt.Println("Usage: vmaf_analyzer [--subsample n] [--threads n] [--model vmaf_v0.6.1.pkl] [--datafile data.json] mezzanine.mp4 https://example.com/hls_stream.m3u8")
		return
	}

	mezzanineFile := flag.Args()[0]
	manifestURL := flag.Args()[1]

	// Parse the viewer data

	// Load the master manfest
	resp, err := http.Get(manifestURL)
	if err != nil {
		fmt.Printf("Failed to fetch master manfiest (%s): %v\n", manifestURL, err)
		return
	}
	defer resp.Body.Close()

	manifest, manifestType, err := m3u8.DecodeFrom(resp.Body, false)
	if err != nil {
		fmt.Printf("Failed to decode master manifest: %v", err)
		return
	}

	var masterPlaylist *m3u8.MasterPlaylist
	switch manifestType {
	case m3u8.MASTER:
		masterPlaylist = manifest.(*m3u8.MasterPlaylist)
	default:
		fmt.Printf("Invalid manifest format, must be a master manifest")
		return
	}

	fmt.Printf("Master Playlist: %+v\n", masterPlaylist)
	fmt.Printf("Loading mezzanine: %s\n", mezzanineFile)

	for _, variant := range masterPlaylist.Variants {
		fmt.Printf("Here's a variant: %v\n", variant)
	}

	fmt.Printf("Input has %d variants\n", len(masterPlaylist.Variants))

	fileReader, err := os.Open(*dataFile)
	if err != nil {
		fmt.Printf("Failed to load data file: %v", err)
		return
	}
	defer fileReader.Close()

	rawFile, err := ioutil.ReadAll(fileReader)
	if err != nil {
		fmt.Printf("Failed to read data file: %v", err)
		return
	}

	var data DataFile
	if err := json.Unmarshal(rawFile, &data); err != nil {
		fmt.Printf("Failed to unmarshal data: %v", err)
		return
	}

	if len(data.BandwidthPcts) != bandwidthsLen {
		fmt.Printf("Invalid input data; expected %d bandwidth entries but got %d\n", bandwidthsLen, len(data.BandwidthPcts))
		return
	}

	fmt.Printf("Bandwidths len: %d sum: %f\n", len(data.BandwidthPcts), sumFloat64Array(data.BandwidthPcts))
	fmt.Printf("Resolutions len: %d sum: %f\n", len(data.ResolutionPcts), sumFloat64Array(data.ResolutionPcts))

	fileInfo, err := probeFile(mezzanineFile)
	if err != nil {
		fmt.Printf("Failed to probe file: %v\n", err)
		return
	}

	if len(fileInfo.Streams) != 1 {
		fmt.Printf("Input file must have exactly 1 video stream, but had %d streams\n", len(fileInfo.Streams))
		return
	}

	videoStream := fileInfo.Streams[0]
	if videoStream.Width == 0 || videoStream.Height == 0 {
		fmt.Printf("Input file must have a valid width and height, but has %dx%d", videoStream.Width, videoStream.Height)
		return
	}

	fmt.Printf("Input widthxheight: %dx%d\n", videoStream.Width, videoStream.Height)

	// Parse bitrate & resolution combinations

	// For each rendition
	// Load the renditions, and download all renditions to local ts streams

	// For each rendition
	// For each resolution bucket with > 0 viewers
	// Calculate VMAF

	fmt.Println("Done")
}