package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/olehbozhok/site_block_checker/proxy_util"
	"github.com/olehbozhok/site_block_checker/repo"
)

func startCheck(db *repo.DB) {
	tick := time.Tick(time.Minute * 5)
	once := make(chan struct{}, 1)
	once <- struct{}{}

	for {
		select {
		case <-tick:
		case <-once:
			once = nil
		}

		list, err := db.GetListURLs()
		if err != nil {
			log.Println("error load GetListURLs in worker:", err.Error())
			continue
		}
		if len(list) == 0 {
			log.Println("got empty list urls from db")
			continue
		}

		ruProxyList, err := db.GetProxy("RU")
		if err != nil {
			log.Println("err on get proxy RU,", err)
			continue
		}

		byProxyList, err := db.GetProxy("BY")
		if err != nil {
			log.Println("err on get proxy RU,", err)
			continue
		}

		if len(ruProxyList) == 0 {
			log.Println("ruProxyList is empty")
		}
		if len(byProxyList) == 0 {
			log.Println("byProxyList is empty")
		}

		wg := sync.WaitGroup{}
		wg.Add(2)

		check := func(country string, proxyList []repo.ProxyData) {
			defer wg.Done()

			if len(proxyList) == 0 {
				log.Println("no proxy for country ", country)
				return
			}
			log.Println("start handle urls by country: ", country)
			for _, urlData := range list {
				for i, proxy := range proxyList {
					isLast := (len(proxyList) - 1) == i

					client, err := proxy.GetClient()
					if err != nil {
						log.Println("could not init proxy client for proxy id:", proxy.ID)
						continue
					}
					checkResult, err := checkSite(urlData, client, country)
					if err != nil {
						log.Println("could not check using proxy id:", proxy.ID, " , site id: ", urlData.ID, " , url: ", urlData.URL, " error: ", err)

						if !isLast {
							continue
						}
					}
					log.Printf("done check  url %s lang %s has err:%v", urlData.URL, country, checkResult.ErrorText != "")
					err = db.UpdateCheckURLResult(checkResult)
					if err != nil {
						log.Println("couldUpdateCheckURLResult proxyID id:", proxy.ID, " , site id: ", urlData.ID, " , url: ", urlData.URL, " , error:", err)
					}
					break
				}
			}
			log.Println("done handle urls by country: ", country)
		}

		go check("RU", ruProxyList)
		go check("BY", byProxyList)

		wg.Wait()

		log.Println("done all handlers")
	}
}

type checkResult struct {
	OK         bool
	StatusCode int
	ResultData string
}

func checkSite(urlData repo.CheckURL, client *proxy_util.Client, country string) (result repo.CheckURLResult, err error) {
	result.CheckUrlID = urlData.ID
	result.Country = country

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	// client2 := resty.New()
	client2 := client

	req := client2.R().
		SetContext(ctx).
		SetHeader("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:97.0) Gecko/20100101 Firefox/97.0").
		SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8").
		SetHeader("Accept-Encoding", "gzip, deflate").
		SetHeader("Accept-Language", "ru-RU,ru;q=0.8,en-US;q=0.5,en;q=0.3").
		SetHeader("Sec-Fetch-Dest", "document").
		SetHeader("Sec-Fetch-Mode", "navigate").
		SetHeader("Sec-Fetch-Site", "cross-site")

	resp, err := req.
		Get(strings.TrimSpace(urlData.URL))
	if err != nil {
		result.ErrorText = err.Error()
		return result, err
	}

	result.StatusCode = resp.StatusCode()
	result.ResultData = string(resp.Body())
	result.UpdatedAt = time.Now()
	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusNotModified {
		result.IsActive = true
	}

	return result, nil
}
