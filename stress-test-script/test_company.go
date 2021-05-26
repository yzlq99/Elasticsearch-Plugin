package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/wrappers"
	"gitlab.com/momentum-valley/adam/pkg/essentials"
	pb "gitlab.com/momentum-valley/advanced-search/rpc/advanced-search"
)

var qps = flag.Int("q", 50, "qps")
var pageSize = flag.Int("p", 50, "page size")
var searchType = flag.Int("t", 1, "search type 0-6")
var timeLimit = flag.Int("l", 500, "time limit(unit: ms)")
var conditionCount = flag.Int("c", 5, "count of conditions")

var seconds = flag.Int("m", 60, "seconds of benchmark test")

var duration time.Duration
var reqCount int
var resChan = make(chan testResult)
var reqChan = make(chan pb.SearchRequest)

var searchTypes = []pb.SearchType{
	pb.SearchType_NONE,
	pb.SearchType_COMPANY,
	pb.SearchType_PERSON,
	pb.SearchType_FUND,
	pb.SearchType_LP,
	pb.SearchType_INS_INVESTOR,
	pb.SearchType_DEAL,
}

type testResult struct {
	Err   error
	Cost  time.Duration
	Count int
}

func main() {
	flag.Parse()
	log.Printf("search type is %s", pb.SearchType_name[int32(*searchType)])
	log.Printf("qps=%d", *qps)
	log.Printf("pageSize=%d", *pageSize)
	log.Printf("timeLimit=%d ms", *timeLimit)
	log.Printf("conditionCount=%d", *conditionCount)
	log.Printf("test for %d seconds", *seconds)

	duration = time.Duration(*seconds) * time.Second
	reqCount = int(duration/time.Second) * *qps
	fmt.Println("===", reqCount)
	resChan = make(chan testResult, reqCount)
	reqChan = make(chan pb.SearchRequest, reqCount)
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	// runtime.GOMAXPROCS(4) // 最多使用4个核

	benchmarkTest()
}

func benchmarkTest() {
	defer calculate()
	defer printTimeOutReq()
	client := pb.NewAdvancedSearchProtobufClient("http://localhost:8081", &http.Client{})

	investorLines, err := readFileLines(investorsFileName, 0)
	verticalLines, err := readFileLines(verticalsFileName, 0)
	industryLines, err := readFileLines(industriesFileName, 0)
	if err != nil {
		return
	}
	choicedIndustryLines := randChoice(0, industryLines, 0)
	choicedVerticalLines := randChoice(0, verticalLines, 0)
	choicedInvestorLines := randChoice(0, investorLines, 500)
	industries := getVerticals(choicedIndustryLines)
	verticals := getVerticals(choicedVerticalLines)
	verticals = append(verticals, industries...)
	investors := getInvestors(choicedInvestorLines)

	// 定时一分钟
	durationTimer := time.NewTimer(duration)

	for {
		// 每秒发 qps 个请求
		secondTimer := time.NewTimer(1 * time.Second)
		for i := 0; i < *qps; i++ {
			go makeQuery(client, verticals, investors)
		}
		select {
		case <-durationTimer.C:
			fmt.Println("done")
			return
		case <-secondTimer.C:
			log.Println("-1s")
			continue
		}
	}
}

// -q 100 => avg=18.43444s, min=0.00571s, max=125.02240s, failed=1294, successCount=3503, successRatio=58.38%, timeOut=3411, timeOutRatio=56.85%
// 2020/03/06 11:09:16 test_company.go:205: success: avg=31.49749s, min=0.02215s, max=125.02240s, timeOut=3328, timeOutRatio=95.00%

// -p 50 -q 150 => avg=0.35274s, min=0.04353s, max=1.85377s, failed=0
// -p 50 -q 50 => avg=0.17316s, min=0.04431s, max=0.37585s, failed=0
// -p 100 -q 50 => avg=0.42441s, min=0.09380s, max=0.61362s, failed=0
// -p 100 -q 100 => avg=0.76590s, min=0.12139s, max=1.99239s, failed=0
// -p 100 -q 150 => avg=10.05945s, min=0.17849s, max=77.71698s, failed=344
// -p 50 -q 150 => avg=0.55904s, min=0.05103s, max=0.90978s, failed=0
// -p 50 -q 170 => avg=0.95400s, min=0.06595s, max=4.48447s, failed=0
// -p 50 -q 160 => avg=0.61877s, min=0.05361s, max=1.96621s, failed=0

// -q 55 -l 1000 -c 6 -m 1 => avg=2.68161s, min=0.00616s, max=57.35240s, failed=0, successCount=2463, successRatio=74.64%, timeOut=1745, timeOutRatio=52.88%
// 2020/03/05 19:30:36 test_company.go:196: success: avg=3.55947s, min=0.01280s, max=57.35240s, timeOut=1745, timeOutRatio=70.85%

// -q 53 -l 1000 -c 6 -m 1 => avg=0.33311s, min=0.00565s, max=2.71005s, failed=0, successCount=2335, successRatio=73.43%, timeOut=134, timeOutRatio=4.21%
// 2020/03/05 19:31:46 test_company.go:196: success: avg=0.42462s, min=0.01249s, max=2.71005s, timeOut=134, timeOutRatio=5.74%

// -q 54 -l 1000 -c 6 -m 1 => avg=1.38937s, min=0.00582s, max=18.89344s, failed=0, successCount=2384, successRatio=73.58%, timeOut=1200, timeOutRatio=37.04%
// 2020/03/05 19:33:09 test_company.go:196: success: avg=1.85686s, min=0.00903s, max=18.89344s, timeOut=1200, timeOutRatio=50.34%

// =================== full data from db =====================
// -q 200 -p 50 -l 500 -c 5 -m 10 => avg=9.99102s, min=0.39532s, max=39.05917s, failed=1002, successCount=723, successRatio=36.15%, timeOut=997, timeOutRatio=49.85%
// 2020/03/09 15:56:12 test_company.go:209: success: avg=20.79549s, min=0.39532s, max=39.05917s, timeOut=722, timeOutRatio=99.86%

// -q 50 -p 50 -l 500 -c 5 -m 10 => avg=4.59467s, min=0.19662s, max=15.20100s, failed=15, successCount=359, successRatio=71.80%, timeOut=461, timeOutRatio=92.20%
// 2020/03/09 15:49:23 test_company.go:209: success: avg=5.02416s, min=0.36225s, max=14.20501s, timeOut=356, timeOutRatio=99.16%
// avg=3.99284s, min=0.02581s, max=13.68829s, failed=8, successCount=368, successRatio=73.60%, timeOut=479, timeOutRatio=95.80%
// 2020/03/09 15:47:38 test_company.go:209: success: avg=4.29264s, min=0.11268s, max=13.68829s, timeOut=364, timeOutRatio=98.91%

// -q 100 -p 50 -l 500 -c 5 -m 10 => avg=12.98355s, min=0.34135s, max=32.34448s, failed=25, successCount=747, successRatio=74.70%, timeOut=972, timeOutRatio=97.20%
// 2020/03/09 15:46:58 test_company.go:209: success: avg=13.57074s, min=0.58259s, max=32.34448s, timeOut=747, timeOutRatio=100.00%

// ==================== with redis cache =====================
// f: while the redis is empty
// n: next test(redis is not empty)
// ===========================================================
// f: -q 50 => avg=0.34985s, min=0.01851s, max=2.50174s, failed=94, successCount=2098, successRatio=69.93%, timeOut=558, timeOutRatio=18.60%
// 2020/03/10 16:37:50 test_company.go:222: success: avg=0.41293s, min=0.04430s, max=2.50174s, timeOut=537, timeOutRatio=25.60%
// n: => avg=0.29256s, min=0.02216s, max=0.77147s, failed=94, successCount=2117, successRatio=70.57%, timeOut=285, timeOutRatio=9.50%
// 2020/03/10 17:06:58 test_company.go:233: success: avg=0.33601s, min=0.03798s, max=0.77147s, timeOut=285, timeOutRatio=13.46%

// f: -q 100 => avg=6.85364s, min=0.05450s, max=61.18763s, failed=648, successCount=3892, successRatio=64.87%, timeOut=5307, timeOutRatio=88.45%
// 2020/03/10 17:12:48 test_company.go:235: success: avg=7.83628s, min=0.19425s, max=61.18763s, timeOut=3871, timeOutRatio=99.46%
// n: => avg=2.62544s, min=0.05523s, max=31.54572s, failed=257, successCount=4171, successRatio=69.52%, timeOut=5511, timeOutRatio=91.85%
// 2020/03/10 17:14:32 test_company.go:235: success: avg=2.79494s, min=0.06452s, max=31.54572s, timeOut=4054, timeOutRatio=97.19%
// n: -q 100 -l 1000 => avg=0.92567s, min=0.03969s, max=5.04282s, failed=184, successCount=4237, successRatio=70.62%, timeOut=1773, timeOutRatio=29.55%
// 2020/03/10 17:15:58 test_company.go:236: success: avg=1.01939s, min=0.04331s, max=4.96898s, timeOut=1543, timeOutRatio=36.42%

// f: -q 100 -l 1000 -m 10 -c 6 => avg=5.49227s, min=0.03498s, max=16.63220s, failed=37, successCount=687, successRatio=68.70%, timeOut=911, timeOutRatio=91.10%
// 2020/03/10 17:20:18 test_company.go:240: success: avg=5.86417s, min=0.15757s, max=16.63220s, timeOut=663, timeOutRatio=96.51%
// n: => avg=0.81029s, min=0.05833s, max=2.45304s, failed=27, successCount=682, successRatio=68.20%, timeOut=263, timeOutRatio=26.30%
// 2020/03/10 17:21:01 test_company.go:240: success: avg=0.91206s, min=0.10992s, max=2.45304s, timeOut=234, timeOutRatio=34.31%
// n: -q 200 -l 1000 -m 10 -c 6 => avg=4.12843s, min=0.04949s, max=16.27353s, failed=704, successCount=898, successRatio=44.90%, timeOut=1199, timeOutRatio=59.95%
// 2020/03/10 17:25:11 test_company.go:243: success: avg=6.57955s, min=0.09086s, max=16.27353s, timeOut=859, timeOutRatio=95.66%
// n: => avg=3.56645s, min=0.08193s, max=13.96186s, failed=703, successCount=912, successRatio=45.60%, timeOut=1204, timeOutRatio=60.20%
// 2020/03/10 17:26:55 test_company.go:245: success: avg=5.61324s, min=0.19081s, max=13.89521s, timeOut=862, timeOutRatio=94.52%

// f: -q 150 -l 1000 -m 60 -c 6 => avg=5.07436s, min=0.07255s, max=63.78311s, failed=4239, successCount=3349, successRatio=37.21%, timeOut=4469, timeOutRatio=49.66%
// 2020/03/10 17:33:11 test_company.go:249: success: avg=9.82911s, min=0.24473s, max=63.15872s, timeOut=3197, timeOutRatio=95.46%
// n: -q 150 -l 1000 -m 10 -c 6 => avg=3.64592s, min=0.09965s, max=12.70890s, failed=119, successCount=984, successRatio=65.60%, timeOut=1164, timeOutRatio=77.60%
// 2020/03/10 17:34:41 test_company.go:250: success: avg=4.07340s, min=0.12169s, max=12.61857s, timeOut=866, timeOutRatio=88.01%
// ===========================================
// 以下测试一直不清空 redis
// ===========================================
// f: -q 200 -l 1000 -m 60 -c 0 => avg=2.47449s, min=0.36995s, max=28.07103s, failed=2666, successCount=9334, successRatio=77.78%, timeOut=8445, timeOutRatio=70.38%
// 2020/03/10 17:37:16 test_company.go:252: success: avg=3.18126s, min=0.36995s, max=28.07103s, timeOut=8445, timeOutRatio=90.48%
// n: -q 100 -l 1000 -m 60 -c 0 => avg=1.48031s, min=0.23182s, max=13.87515s, failed=0, successCount=6000, successRatio=100.00%, timeOut=1539, timeOutRatio=25.65%
// 2020/03/10 17:38:39 test_company.go:252: success: avg=1.48031s, min=0.23182s, max=13.87515s, timeOut=1539, timeOutRatio=25.65%
// n: -q 100 -l 1000 -m 60 -c 6 => avg=7.08050s, min=0.06034s, max=63.89661s, failed=1728, successCount=3013, successRatio=50.22%, timeOut=4027, timeOutRatio=67.12%
// 2020/03/10 17:41:27 test_company.go:257: success: avg=10.26626s, min=0.22959s, max=63.89661s, timeOut=2895, timeOutRatio=96.08%
// n: -q 100 -l 1000 -m 10 -c 6 => avg=1.75112s, min=0.04753s, max=9.18480s, failed=25, successCount=684, successRatio=68.40%, timeOut=673, timeOutRatio=67.30%
// 2020/03/10 17:42:32 test_company.go:261: success: avg=1.92000s, min=0.20107s, max=9.18480s, timeOut=512, timeOutRatio=74.85%

// ==========================================================================
// 以下测试模拟 fulldata 数据全部放在 redis 中的性能
// 测试时服务端，客户端（测试脚本）以及 redis 都是运行在本机
// es 数据量为 100W，redis 数据量为 50W+（由于本机内存限制不能将100W都存下）
// 为解决无法将所有数据存放在 redis，导致部分请求需要到数据库请求数据的问题，我将需要到数据库获取数据的这部分请求视为错误请求，延时1s并返回错误
// ==========================================================================
// -q 100 -l 1000 -m 10 -c 2 => avg=0.44239s, min=0.02984s, max=0.85654s, failed=311, successCount=582, successRatio=58.20%, timeOut=0, timeOutRatio=0.00%
// 2020/03/11 14:08:55 test_company.go:263: success: avg=0.66740s, min=0.07385s, max=0.85654s, timeOut=0, timeOutRatio=0.00%
// avg=0.49014s, min=0.05342s, max=0.93513s, failed=281, successCount=614, successRatio=61.40%, timeOut=0, timeOutRatio=0.00%
// 2020/03/11 14:07:08 test_company.go:263: success: avg=0.69709s, min=0.09078s, max=0.93513s, timeOut=0, timeOutRatio=0.00%

// -q 130 -l 1000 -m 10 -c 2 => avg=0.60349s, min=0.04131s, max=1.97735s, failed=372, successCount=775, successRatio=59.62%, timeOut=104, timeOutRatio=8.00%
// 2020/03/11 14:09:34 test_company.go:263: success: avg=0.87828s, min=0.05393s, max=1.97735s, timeOut=97, timeOutRatio=12.52%
// avg=0.68816s, min=0.08945s, max=2.30834s, failed=374, successCount=781, successRatio=60.08%, timeOut=313, timeOutRatio=24.08%
// 2020/03/11 14:06:46 test_company.go:263: success: avg=0.99904s, min=0.11277s, max=2.30834s, timeOut=290, timeOutRatio=37.13%

// -q 135 -l 1000 -m 10 -c 2 => avg=1.02567s, min=0.09680s, max=6.40893s, failed=417, successCount=787, successRatio=58.30%, timeOut=682, timeOutRatio=50.52%
// 2020/03/11 14:06:29 test_company.go:263: success: avg=1.52722s, min=0.17615s, max=6.40893s, timeOut=601, timeOutRatio=76.37%

// -q 140 -l 1000 -m 10 -c 2 => avg=1.18517s, min=0.19001s, max=4.49814s, failed=414, successCount=830, successRatio=59.29%, timeOut=792, timeOutRatio=56.57%
// 2020/03/11 14:06:05 test_company.go:263: success: avg=1.74516s, min=0.35652s, max=4.49814s, timeOut=700, timeOutRatio=84.34%

// -q 150 -l 1000 -m 10 -c 2 => avg=1.69894s, min=0.05533s, max=8.85485s, failed=471, successCount=866, successRatio=57.73%, timeOut=925, timeOutRatio=61.67%
// 2020/03/11 14:05:40 test_company.go:263: success: avg=2.58271s, min=0.32639s, max=8.85485s, timeOut=805, timeOutRatio=92.96%

func makeQuery(client pb.AdvancedSearch, verticals []string, investors []string) {

	cursor := base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(0)))
	// cursor := base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(rand.Intn(1000))))
	var conditions = getSearchConditions(verticals, investors)
	var orderColumns = getOrderColumns()
	var columnIds = getColumnIds()
	var req = pb.SearchRequest{
		SearchType: searchTypes[*searchType],
		First: &wrappers.Int32Value{
			Value: int32(*pageSize),
		},
		After: &wrappers.StringValue{
			Value: cursor,
		},
		Conditions:   conditions,
		OrderColumns: orderColumns,
		ColumnIds:    columnIds,
	}
	start := time.Now()
	result, err := client.Search(context.Background(), &req)
	cost := time.Since(start)
	var count int
	if err == nil {
		count = len(result.Nodes)
	}
	resChan <- testResult{Err: err, Cost: cost, Count: count}
	if cost.Seconds()*1000 > float64(*timeLimit) {
		reqChan <- req
	}
}

func calculate() {
	var failed, timeOut, successCount int
	var minCost, maxCost, avgCost, totalCost, timeOutRatio, successRatio float64
	var successTimeOut int
	var successMinCost, successMaxCost, successAvgCost, successTotalCost, successTimeOutRatio float64
	minCost = 10000000
	successMinCost = 10000000
	for i := 0; i < reqCount; i++ {
		res := <-resChan
		if res.Err != nil {
			failed++
			continue
		}
		seconds := res.Cost.Seconds()
		if res.Count != 0 {
			successTotalCost += seconds
			if seconds < successMinCost {
				successMinCost = seconds
			}
			if seconds > successMaxCost {
				successMaxCost = seconds
			}
			if seconds*1000 > float64(*timeLimit) {
				successTimeOut++
			}
			successCount++
		}
		totalCost += seconds
		if seconds < minCost {
			minCost = seconds
		}
		if seconds > maxCost {
			maxCost = seconds
		}
		if seconds*1000 > float64(*timeLimit) {
			timeOut++
		}
	}
	avgCost = totalCost / float64(reqCount)
	timeOutRatio = float64(timeOut) / float64(reqCount) * 100
	successRatio = float64(successCount) / float64(reqCount) * 100

	successAvgCost = successTotalCost / float64(successCount)
	successTimeOutRatio = float64(successTimeOut) / float64(successCount) * 100
	log.Printf("avg=%.5fs, min=%.5fs, max=%.5fs, failed=%d, successCount=%d, successRatio=%.2f%%, timeOut=%d, timeOutRatio=%.2f%%\n",
		avgCost, minCost, maxCost, failed, successCount, successRatio, timeOut, timeOutRatio)

	log.Printf("success: avg=%.5fs, min=%.5fs, max=%.5fs, timeOut=%d, timeOutRatio=%.2f%%",
		successAvgCost, successMinCost, successMaxCost, successTimeOut, successTimeOutRatio)
}

func printTimeOutReq() {
	for i := 0; i < len(reqChan); i++ {
		<-reqChan
		// req := <-reqChan
		// reqJSON, err := json.Marshal(req)
		// if err != nil {
		// 	fmt.Println(err)
		// 	return
		// }
		// fmt.Println(string(reqJSON))
	}
}

// -p 50 -q 50 => avg=0.19381s, min=0.02836s, max=2.40481s, failed=0
//             => avg=0.14639s, min=0.02872s, max=0.40912s, failed=0
func getSearchConditions(verticals []string, investors []string) []*pb.SearchCondition {
	searchConditions := make([]*pb.SearchCondition, 0)
	searchConditions = append(searchConditions, randSearchCondition("company.founded_at", DateValueType))
	searchConditions = append(searchConditions, randSearchCondition("company.latest_deal_date", DateValueType))
	searchConditions = append(searchConditions, randSearchCondition("company.latest_deal_amount", AmountValueType))
	searchConditions = append(searchConditions, randSearchCondition("company.post_money_valuation", AmountValueType))
	searchConditions = append(searchConditions, randSearchCondition("company.total_deal_amount", AmountValueType))
	searchConditions = append(searchConditions, randSearchCondition("company.deal_count", NumberValueType))
	searchConditions = append(searchConditions, randSearchCondition("company.investment_count", NumberValueType))
	searchConditions = append(searchConditions, randSearchCondition("company.investment_count_last_year", NumberValueType))
	searchConditions = append(searchConditions, randSearchCondition("company.investment_amount_last_year", AmountValueType))
	searchConditions = append(searchConditions, randSearchCondition("company.acquisition_count", NumberValueType))
	searchConditions = append(searchConditions, randSearchCondition("company.acquisition_amount", AmountValueType))

	searchConditions = append(searchConditions, searchCondition("company.ownership_status", pb.Operator_INCLUDES_ANY, randOwnershipStatus(nilPercent), ""))
	searchConditions = append(searchConditions, searchCondition("company.financing_status", pb.Operator_INCLUDES_ANY, randFinancingStatus(nilPercent), ""))
	searchConditions = append(searchConditions, searchCondition("company.headquarter_location", pb.Operator_INCLUDES_ANY, randHeadquarterLocation(nilPercent), ""))
	searchConditions = append(searchConditions, searchCondition("company.latest_deal_type", pb.Operator_INCLUDES_ANY, randDealType(nilPercent), ""))

	searchConditions = append(searchConditions, searchCondition("company.vertical", pb.Operator_INCLUDES_ANY, randChoice(nilPercent, verticals, 10), ""))
	searchConditions = append(searchConditions, searchCondition("company.shareholder", pb.Operator_INCLUDES_ANY, randChoice(nilPercent, investors, 10), ""))
	searchConditions = append(searchConditions, searchCondition("company.lead_investor", pb.Operator_INCLUDES_ANY, randChoice(nilPercent, investors, 10), ""))

	result := make([]*pb.SearchCondition, 0)
	for _, searchCondition := range searchConditions {
		if searchCondition != nil {
			result = append(result, searchCondition)
		}
	}
	return choiceConditions(result, *conditionCount)
}

func choiceConditions(array []*pb.SearchCondition, count int) []*pb.SearchCondition {

	choicedArr := make([]*pb.SearchCondition, 0)
	if count <= 0 || array == nil {
		return choicedArr
	}
	list := perm(len(array), count)
	for _, index := range list {
		choicedArr = append(choicedArr, array[index])
	}
	return choicedArr
}

func randDealType(nilPercent int) []string {
	types := make([]string, 0)
	for _, dealType := range dealTypes {
		types = append(types, strconv.Itoa(int(dealType)))
	}
	return randChoice(nilPercent, types, 0)
}

func randHeadquarterLocation(nilPercent int) []string {
	headquarterLocations := make([]string, 0)
	for _, location := range locations {
		headquarterLocations = append(headquarterLocations, strconv.Itoa(int(location)))
	}
	return randChoice(nilPercent, headquarterLocations, 0)
}

func randFinancingStatus(nilPercent int) []string {
	status := make([]string, 0)
	for _, financialStatus := range financialStatuses {
		status = append(status, string(financialStatus))
	}
	return randChoice(nilPercent, status, 0)
}

func randOwnershipStatus(nilPercent int) []string {
	status := make([]string, 0)
	for _, ownershipStatus := range ownershipStatuses {
		status = append(status, string(ownershipStatus))
	}
	return randChoice(nilPercent, status, 0)
}

func getInvestors(lines []string) []string {
	if lines == nil || len(lines) <= 0 {
		return nil
	}
	investors := make([]string, 0)
	for _, line := range lines {
		arr := strings.Split(line, ",")
		if len(arr) <= 1 {
			continue
		}
		investors = append(investors, strings.TrimSpace(arr[0]))
	}
	return investors
}

func getVerticals(lines []string) []string {
	if lines == nil || len(lines) <= 0 {
		return nil
	}
	verticals := make([]string, 0)
	for _, line := range lines {
		arr := strings.Split(line, ",")
		if len(arr) <= 1 {
			continue
		}
		verticals = append(verticals, strings.TrimSpace(arr[0]))
	}
	return verticals
}

func readFileLines(fileName string, count int64) ([]string, error) {

	lines := make([]string, 0)
	file, err := os.Open(fileName)
	if err != nil {
		log.Printf("Cannot open text file: %s, err: [%v]", fileName, err)
		return nil, err
	}
	defer file.Close()
	if count <= 0 {
		fileInfo, err := file.Stat()
		if err != nil {
			log.Printf("Cannot get file info file name: %s, err: [%v]", fileName, err)
			return nil, err
		}
		// fileInfo.Size() 大于文件行数
		count = fileInfo.Size()
	}

	scanner := bufio.NewScanner(file)
	for i := 0; int64(i) < count && scanner.Scan(); i++ {
		line := scanner.Text()
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Cannot scanner text file: %s, err: [%v]", fileName, err)
		return nil, err
	}
	return lines, nil
}

// nilPercent: 返回 nil 的几率
// maxLen: 返回最大长度,如果传0则最大长度为数组长度
func randChoice(nilPercent int, array []string, maxLen int) []string {
	if array == nil || !randBoolean(nilPercent) {
		return nil
	}
	if maxLen <= 0 {
		maxLen = len(array)
	}
	count := rand.Intn(maxLen) + 1
	return choice(array, count)
}

func choice(array []string, count int) []string {

	choicedArr := make([]string, 0)
	if count <= 0 || array == nil {
		return choicedArr
	}
	list := perm(len(array), count)
	for _, index := range list {
		choicedArr = append(choicedArr, array[index])
	}
	return choicedArr
}

// n: 数组总个数
// count: 需要生成的随机数个数
func perm(n int, count int) []int {
	var list []int
	var index = 0
	if n > 200 {
		// n > 200 时，Perm生成200以内随机，最终结果为 perm+随机index
		list = rand.Perm(200)
		index = rand.Intn(n - 200)
	} else {
		list = rand.Perm(n)
	}
	result := make([]int, 0)
	for i, m := range list {
		if i >= count {
			break
		}
		result = append(result, m+index)
	}
	return result
}

func randSearchCondition(columnID string, columnValueType ValueType) *pb.SearchCondition {
	if !randBoolean(nilPercent) {
		return nil
	}
	var operator = randOperator(columnValueType)
	switch columnValueType {
	case NumberValueType:
		return searchCondition(columnID, operator, randNumber(operator), "")
	case AmountValueType:
		return searchCondition(columnID, operator, randAmout(operator), currencyCodes[rand.Intn(currencyCodesLen)].Display())
	case DateValueType:
		return searchCondition(columnID, operator, randDate(operator), "")
	// case TextValueType:
	// case TextArrayValueType:
	// case NumberArrayValueType:

	default:
		return &pb.SearchCondition{}
	}
}

func randAmout(operator pb.Operator) []string {
	randAmout := rand.Intn(10000)*200000 + 1
	randAmoutStr := strconv.Itoa(randAmout)
	if operator != pb.Operator_BETWEEN {
		return []string{randAmoutStr}
	}
	return []string{strconv.Itoa((randAmout - rand.Intn(randAmout)) / 2), randAmoutStr}
}

func randNumber(operator pb.Operator) []string {
	randInt := rand.Intn(10) + 1
	randIntStr := strconv.Itoa(randInt)
	if operator != pb.Operator_BETWEEN {
		return []string{randIntStr}
	}
	return []string{strconv.Itoa((randInt - rand.Intn(randInt)) / 2), randIntStr}
}

func randDate(operator pb.Operator) []string {
	randDate := *randomDate(zeroTime)
	if operator != pb.Operator_BETWEEN {
		return []string{randDate.Format(dateFormat)}
	}
	return []string{randDate.Format(dateFormat), (*randomDate(randDate)).Format(dateFormat)}
}

func randomDate(base time.Time) *time.Time {
	var randTime time.Time
	if base.IsZero() {
		randTime = time.Date(
			rand.Intn(40)+1980,
			time.Month(rand.Intn(12)+1),
			rand.Intn(28)+1,
			0, 0, 0, 0, time.UTC,
		)
	} else {
		randTime = base.AddDate(rand.Intn(5), rand.Intn(5), rand.Intn(5))
	}
	return &randTime
}

func randBoolean(falsePercent int) bool {
	if rand.Intn(100) < falsePercent {
		return false
	}
	return true
}

func searchCondition(id string, operator pb.Operator, values []string, currencyCode string) *pb.SearchCondition {
	if id == "" || values == nil {
		return nil
	}
	return &pb.SearchCondition{
		Id:           id,
		Operator:     operator,
		Values:       values,
		CurrencyCode: currencyCode,
	}
}

func randOperator(valueType ValueType) pb.Operator {
	switch valueType {
	case NumberValueType:
		return compareOP[rand.Intn(compareOPLen)]
	case AmountValueType:
		return compareOP[rand.Intn(compareOPLen)]
	case DateValueType:
		return compareOP[rand.Intn(compareOPLen)]
	// case TextValueType:
	// 	return pb.Operator_INCLUDES_ANY
	// case TextArrayValueType:
	// 	return pb.Operator_INCLUDES_ANY
	// case NumberArrayValueType:
	// 	return pb.Operator_INCLUDES_ANY
	default:
		return pb.Operator_INCLUDES_ANY
	}
}

func getOrderColumns() []*pb.OrderColumn {
	return []*pb.OrderColumn{
		&pb.OrderColumn{
			ColumnId: "company_search_result.column.founded_at",
			IsDesc:   true,
		},
	}
}

func getColumnIds() []string {
	return []string{
		"company_search_result.column.short_name",
		"company_search_result.column.full_name",
		"company_search_result.column.founded_at",
	}
}

type ValueType int

const (
	NumberValueType ValueType = iota
	AmountValueType
	DateValueType
	// TextValueType
	// TextArrayValueType
	// NumberArrayValueType
)

var compareOP = []pb.Operator{
	pb.Operator_AFTER,
	pb.Operator_BEFORE,
	pb.Operator_BETWEEN,
}
var compareOPLen = len(compareOP)

const (
	dateFormat = "2006-01-02"
	nilPercent = 0
)

var zeroTime time.Time

var locations = []essentials.Location{
	essentials.LocationCityBeijing,
	essentials.LocationCityShanghai,
	essentials.LocationCityShenzhen,
	essentials.LocationCityGuangzhou,
	essentials.LocationCityHangzhou,
	essentials.LocationCityWuhan,
	essentials.LocationCityTaiyuan,
	essentials.LocationProvinceGuangdong,
	essentials.LocationProvinceZhejiang,
	essentials.LocationProvinceShandong,
	essentials.LocationAreaLongRiverDelta,
	essentials.LocationAreaZhuRiverDelta,
	essentials.LocationCountryChina,
	essentials.LocationCountryUSA,
	essentials.LocationCountryUK,
	essentials.LocationContinentEU,
}
var locationsLen = len(locations)

var financialStatuses = []essentials.FinancialStatus{
	essentials.FinancingStatusFundingAngelBacking,
	essentials.FinancingStatusFundingVCBacking,
	essentials.FinancingStatusFundingPEBacking,
	essentials.FinancingStatusNotSeekingFund}
var financialStatusesLen = len(financialStatuses)

var ownershipStatuses = []essentials.OwnershipStatus{
	essentials.OwnershipStatusPrivateStartup,
	essentials.OwnershipStatusPrivateMature,
	essentials.OwnershipStatusPrivateAcquired,
	essentials.OwnershipStatusPublic}
var ownershipStatusesLen = len(ownershipStatuses)

var dealTypes = []essentials.DealType{
	essentials.DealTypePEVC,
	essentials.DealTypePEVCSeed,
	essentials.DealTypePEVCAngel,
	essentials.DealTypePEVCPreA,
	essentials.DealTypePEVCSeriesA,
	essentials.DealTypePEVCSeriesAPlus,
	essentials.DealTypePEVCPreB,
	essentials.DealTypePEVCSeriesB,
	essentials.DealTypePEVCSeriesBPlus,
	essentials.DealTypePEVCSeriesC,
	essentials.DealTypePEVCSeriesD,
	essentials.DealTypePEVCPreIPO,
	essentials.DealTypePEVCUnknown,

	// Directed Additional
	essentials.DealTypeDirectedAdditional,
	essentials.DealTypeAMarketPrivatePlacement,
	essentials.DealTypeNEEQPrivatePlacement,

	// Strategic Investment
	essentials.DealTypeStrategicInvestment,

	// Debt Investment
	essentials.DealTypeDebtInvestment,
	essentials.DealTypeMortgageLoans,
	essentials.DealTypeCreditLoans,
	essentials.DealTypeConvertible,
	essentials.DealTypeABS,

	// Exit Deal Types
	essentials.DealTypeIPO,
	essentials.DealTypeNEEQ,
	essentials.DealTypeREEQ,
	essentials.DealTypeMergerAndAcquisition,
	essentials.DealTypeLBO,
	essentials.DealTypeMBO,
	essentials.DealTypeStrategicAcquisition,
	essentials.DealTypeEquityTransfer,
	essentials.DealTypeLiquidation,
	essentials.DealTypeICO,

	// Others
	essentials.DealTypePIPE,
	essentials.DealTypeRealEstate,
	essentials.DealTypeInfrastructure,
	essentials.DealTypeNaturalResource,
	essentials.DealTypeOthers,
}
var dealTypesLen = len(dealTypes)

var currencyCodes = []essentials.CurrencyCode{
	essentials.CurrencyCodeCNY,
	essentials.CurrencyCodeUSD,
	essentials.CurrencyCodeEUR,
	essentials.CurrencyCodeGBP,
	essentials.CurrencyCodeJPY,
}
var currencyCodesLen = len(currencyCodes)

var investorsFileName = "../mock_data/investors.csv"
var verticalsFileName = "../mock_data/verticals.csv"
var industriesFileName = "../mock_data/industries.csv"
