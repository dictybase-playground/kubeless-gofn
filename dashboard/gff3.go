package kubeless

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

type GFF3Consumer func(Storage, string, string, <-chan string) (<-chan error, error)

var rmap = map[string]GFF3Consumer{
	"chromosome":  GFF3RegionConsumer,
	"supercontig": GFF3RegionConsumer,
}

type chrJsonAPI struct {
	Data []*chrdata `json:"data"`
}

type chrdata struct {
	Type       string      `json:"type"`
	Id         string      `json:"id"`
	Attributes *chromosome `json:"attributes"`
}

type chromosome struct {
	Name   string `json:"name"`
	Id     string `json:"id"`
	Length int    `json:"length"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
}

type featJsonAPI struct {
	Data []*featdata `json:"data"`
}

type featdata struct {
	Type       string   `json:"type"`
	Id         string   `json:"id"`
	Attributes *feature `json:"attributes"`
}

type feature struct {
	SeqId   string `json:"seqid"`
	BlockId string `json:"block_id"`
	Source  string `json:"source"`
	Start   int    `json:"start"`
	End     int    `json:"end"`
	Strand  string `json:"strand"`
}

func storeGFFInforamtion(r io.Reader, st Storage, key string, ftypes ...string) error {
	var errcList []<-chan error
	// Read GFF3 and sends the lines in the channel
	linec, errc, err := GFF3LineProducer(r)
	if err != nil {
		return fmt.Errorf("unable to create gff3 producer %s", err)
	}
	errcList = append(errcList, errc)

	// Read GFF3 line and fan out to multiple consumer channels
	allc, errc, err := GFF3Splitter(linec, len(ftypes))
	if err != nil {
		return fmt.Errorf("unable to create gff3 splitter %s", err)
	}
	errcList = append(errcList, errc)

	// Now the GFF3 consumer receives and extract lines
	for i, t := range ftypes {
		if v, ok := rmap[t]; ok {
			errc, err := v(st, key, t, allc[i])
			if err != nil {
				return fmt.Errorf("unable to create consumer for %s %s", t, err)
			}
			errcList = append(errcList, errc)
		} else {
			errc, err := GFF3GenericConsumer(st, key, t, allc[i])
			if err != nil {
				fmt.Errorf("unable to create consumer for %s %s", t, err)
			}
			errcList = append(errcList, errc)
		}
	}
	return WaitForPipeline(errcList...)
}

func GFF3LineProducer(r io.Reader) (<-chan string, <-chan error, error) {
	out := make(chan string)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := fmt.Sprintf("%s\n", scanner.Text())
			out <- line
		}
		if err := scanner.Err(); err != nil {
			errc <- fmt.Errorf("error in scanning gff3 file %s", err)
			return
		}
	}()
	return out, errc, nil
}

func GFF3Splitter(in <-chan string, len int) ([]chan string, <-chan error, error) {
	out := make([]chan string, len)
	errc := make(chan error, 1)
	for i, _ := range out {
		out[i] = make(chan string)
	}
	go func() {
		defer close(errc)
		for line := range in {
			for _, o := range out {
				o <- line
			}
		}
		for _, o := range out {
			close(o)
		}
	}()
	return out, errc, nil
}

func GFF3RegionConsumer(st Storage, key, t string, in <-chan string) (<-chan error, error) {
	errc := make(chan error, 1)
	go func() {
		defer close(errc)
		seenFasta := false
		var chrData []*chrdata
		for line := range in {
			if seenFasta {
				continue
			}
			if strings.HasPrefix(line, "##") {
				if strings.HasPrefix(line, "###") {
					seenFasta = true
				}
				continue
			}
			s := strings.Split(line, "\t")
			if s[2] != t {
				continue
			}
			start, _ := strconv.Atoi(s[3])
			end, _ := strconv.Atoi(s[4])
			chr := &chromosome{
				Start:  start,
				End:    end,
				Length: end - start,
			}
			fields := strings.Split(s[8], ";")
			chr.Id = strings.Split(fields[0], "=")[1]
			chr.Name = strings.Split(fields[1], "=")[1]
			chrData = append(chrData, &chrdata{
				Type:       fmt.Sprintf("%ss", t),
				Id:         chr.Id,
				Attributes: chr,
			})
		}
		ct, err := json.Marshal(&chrJsonAPI{Data: chrData})
		if err != nil {
			errc <- fmt.Errorf("error in json encoding %s", err)
		}
		if err := st.Set(key, fmt.Sprintf("%ss", t), string(ct)); err != nil {
			errc <- fmt.Errorf("error in storing %s data %s", t, err)
		}
	}()
	return errc, nil
}

func GFF3GenericConsumer(st Storage, key, t string, in <-chan string) (<-chan error, error) {
	errc := make(chan error, 1)
	go func() {
		defer close(errc)
		var featData []*featdata
		seenFasta := false
		for line := range in {
			if seenFasta {
				continue
			}
			if strings.HasPrefix(line, "##") {
				if strings.HasPrefix(line, "###") {
					seenFasta = true
				}
				continue
			}
			s := strings.Split(line, "\t")
			if t != s[2] {
				continue
			}
			fields := strings.Split(s[8], ";")
			Id := strings.Split(fields[0], "=")[1]
			start, _ := strconv.Atoi(s[3])
			end, _ := strconv.Atoi(s[4])
			feat := &feature{
				SeqId:   s[0],
				BlockId: s[0],
				Start:   start,
				End:     end,
				Strand:  s[6],
			}
			if len(s[1]) > 0 {
				feat.Source = s[1]
			}
			featData = append(featData, &featdata{
				Type:       fmt.Sprintf("%ss", t),
				Id:         Id,
				Attributes: feat,
			})
		}
		ct, err := json.Marshal(&featJsonAPI{Data: featData})
		if err != nil {
			errc <- fmt.Errorf("error in json encoding %s", err)
		}
		if err := st.Set(key, fmt.Sprintf("%ss", t), string(ct)); err != nil {
			errc <- fmt.Errorf("error in storing %t data %s", t, err)
		}
	}()
	return errc, nil
}

// WaitForPipeline waits for results from all error channels.
// It returns early on the first error.
func WaitForPipeline(errs ...<-chan error) error {
	errc := MergeErrors(errs...)
	for err := range errc {
		if err != nil {
			return err
		}
	}
	return nil
}

// MergeErrors merges multiple channels of errors.
// Based on https://blog.golang.org/pipelines.
func MergeErrors(cs ...<-chan error) <-chan error {
	var wg sync.WaitGroup
	// We must ensure that the output channel has the capacity to
	// hold as many errors
	// as there are error channels.
	// This will ensure that it never blocks, even
	// if WaitForPipeline returns early.
	out := make(chan error, len(cs))

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls
	// wg.Done.
	output := func(c <-chan error) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines
	// are done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
