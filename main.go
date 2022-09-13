package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

const VERSION string = "1.0.0"

const DESCRATION string = `
https://github.com/mspvirajpatel/Xcode_Developer_Disk_Images/tree/master/Developer Disk Image
download images from github  mspvirajpatel
This website is updated very timely

`

type Resource struct {
	Filename string
	Url      string
	PreDir   string
}

func (r *Resource) CreateFolder(parent string) (string, error) {
	folder := path.Join(parent, r.PreDir)
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		if err = os.MkdirAll(folder, 0755); err != nil {
			return "", err
		}
	}
	return path.Join(folder, r.Filename), nil
}

type DownLoad struct {
	Resources []Resource
	Dir       string
	wg        *sync.WaitGroup
}

func GetImagesURLs(csvfile string) {
	fName := csvfile
	file, err := os.Create(fName)
	if err != nil {
		log.Fatalf("Cannot create file %q: %s\n", fName, err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	// Write CSV header
	writer.Write([]string{"Name", "Image URL", "Signature URL"})

	r, _ := regexp.Compile(`^/mspvirajpatel/.*?/(\d+\..*?)$`)
	// Instantiate default collectorss
	c := colly.NewCollector(
		// Allow requests only to store.xkcd.com
		colly.AllowedDomains("github.com"),
		colly.MaxDepth(1),
	)
	c.OnHTML(`a[href]`, func(e *colly.HTMLElement) {
		// e.Request.Visit(e.Attr("href"))
		if r.MatchString(e.Attr("href")) {
			//r.FindAllStringSubmatch(, -1)
			res := r.FindStringSubmatch(e.Attr("href"))
			if len(res) > 1 {
				//fmt.Println(e.Attr("href"))
				dir, _ := url.PathUnescape(res[1])
				fmt.Println(dir)
				//https://raw.githubusercontent.com/mspvirajpatel/Xcode_Developer_Disk_Images/master/Developer%20Disk%20Image/10.0%20(14A345)/DeveloperDiskImage.dmg.signature
				//https://github.com/mspvirajpatel/Xcode_Developer_Disk_Images/raw/master/Developer%20Disk%20Image/10.0%20(14A345)/DeveloperDiskImage.dmg
				fullurl := "https://github.com/mspvirajpatel/Xcode_Developer_Disk_Images/raw/master/Developer%20Disk%20Image/" + res[1]
				fullurlsign := "https://raw.githubusercontent.com/mspvirajpatel/Xcode_Developer_Disk_Images/master/Developer%20Disk%20Image/" + res[1]
				writer.Write([]string{dir, fullurl + "/DeveloperDiskImage.dmg", fullurlsign + "/DeveloperDiskImage.dmg.signature"})
			}
		}
	})

	//this url update quickly
	c.Visit("https://github.com/mspvirajpatel/Xcode_Developer_Disk_Images/tree/master/Developer%20Disk%20Image")
}

func main() {
	fmt.Print(DESCRATION)
	log.Println(VERSION)
	GetImagesURLs("urls.csv")

	file, err := os.Open("urls.csv")
	if err != nil {
		log.Fatalf("Cannot open file %q: %s\n", file.Name(), err)
		return
	}
	defer file.Close()

	d := NewDownload("./")

	csvReader := csv.NewReader(file)
	for {
		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if rec[0] == "Name" {
			continue
		}
		d.AppendResource(rec[0], "DeveloperDiskImage.dmg", rec[1])
		d.AppendResource(rec[0], "DeveloperDiskImage.dmg.signature", rec[2])
	}

	d.Start()

}

func NewDownload(dir string) *DownLoad {
	return &DownLoad{
		Dir: dir,
		wg:  &sync.WaitGroup{},
	}

}
func (d *DownLoad) AppendResource(predir, filename, url string) {
	d.Resources = append(d.Resources, Resource{
		Filename: filename,
		Url:      url,
		PreDir:   predir,
	})

}
func (d *DownLoad) Start() {
	p := mpb.New(mpb.WithWaitGroup(d.wg),
		mpb.WithWidth(60),
		mpb.WithRefreshRate(180*time.Millisecond))
	for _, v := range d.Resources {
		d.wg.Add(1)
		go d.download(d.Dir, v, p)
	}
	p.Wait()
	d.wg.Wait()

}

func (d *DownLoad) download(parent string, r Resource, p *mpb.Progress) {
	defer d.wg.Done()
	//req, _ := http.Get(r.Url) //.NewRequest("GET", r.Url, nil)
	resp, err := http.Get(r.Url) //http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Get失败", err)
	}
	defer resp.Body.Close()

	fileSize, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
	if fileSize == 0 {
		filename, _ := r.CreateFolder(parent)
		file, err := os.Create(filename)
		if err != nil {
			fmt.Println("创建文件失败", err)
		}
		defer file.Close()
		io.Copy(file, resp.Body)
		return
	}

	var total int64 = int64(fileSize)

	bar := p.New(int64(fileSize),
		mpb.BarStyle().Lbound("╢").Filler("▌").Tip("▌").Padding("░").Rbound("╟"),
		mpb.BarFillerClearOnComplete(),
		mpb.PrependDecorators(
			decor.Name(r.Filename, decor.WC{W: len(r.Filename) + 1, C: decor.DidentRight}),
			decor.CountersKibiByte("% .2f / % .2f"),
		),
		mpb.AppendDecorators(
			decor.EwmaETA(decor.ET_STYLE_GO, 90),
			decor.Name(" ] "),
			decor.EwmaSpeed(decor.UnitKiB, "% .2f", 60),
		),
	)

	filename, _ := r.CreateFolder(parent)
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("创建文件失败", err)
	}
	defer file.Close()
	// create proxy reader
	reader := io.LimitReader(resp.Body, total)
	proxyReader := bar.ProxyReader(reader)
	defer proxyReader.Close()

	// copy from proxyReader, ignoring errors
	_, err = io.Copy(file, proxyReader)
	if err != nil {
		fmt.Println("写入文件失败", err)
	}
}
