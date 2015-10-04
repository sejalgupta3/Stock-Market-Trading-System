package main

import (
    "fmt"
    "log"
    "net"
    "net/rpc"
    "net/rpc/jsonrpc"
    "math/rand"
    "io/ioutil"
	"net/http"
	"strings"
	"encoding/json"
	"strconv"
)

type Args struct {
    StockData string
	Balance float32
}

type Reply struct{
	TradeId int
	Stocks string
	UnvestedAmount float32
}

type CheckPortfolioRequestArgs struct {
	TradeId int
}

type CheckPortfolioRequestResult struct{
	Stocks string
	CurrentMarketValue float32
	UnvestedAmount float32
}

type stock struct{
	symbol string
	quantity int
	percentage float32
	newNetAmount float32
	oldNetAmount float32
	leftOverBalance float32
}

var stockMarket map[int] []stock

type StockMarketApp struct{}

func (t *StockMarketApp) HandleBuyStockRequest(args *Args, reply *Reply) error {
	stocks := strings.Split(args.StockData, ",")
	stockArray := make([]stock,len(stocks))
	for i:=0;i<len(stocks);i++{
		stockDetails := strings.Split(stocks[i], ":")
		stockName := stockDetails[0]
		sharePercentage, err := strconv.ParseFloat(strings.Replace(stockDetails[1], "%", "", -1), 64)
		if err != nil {
			log.Fatal(err)
		}
		stockObj := stock{symbol:stockName, percentage:float32(sharePercentage)}
		stockArray[i] = stockObj
	}
    getYahooData(&stockArray)
    var leftOverBalance float32 = 0.0
    for i:=0;i<len(stockArray);i++{
    	stockBalanceShare := ((stockArray[i].percentage / 100.0) * args.Balance)
    	stockArray[i].quantity = int(stockBalanceShare/stockArray[i].newNetAmount)
    	if(reply.Stocks != ""){
    		reply.Stocks = reply.Stocks + "," + stockArray[i].symbol + ":" + strconv.Itoa(stockArray[i].quantity) + ":$" +  fmt.Sprintf("%f", stockArray[i].newNetAmount)
    	}else{
    		reply.Stocks = stockArray[i].symbol + ":" + strconv.Itoa(stockArray[i].quantity) + ":$" + fmt.Sprintf("%f", stockArray[i].newNetAmount)
    	}
    	stockArray[i].leftOverBalance =  stockBalanceShare - (float32(stockArray[i].quantity) * stockArray[i].newNetAmount)
    	leftOverBalance = leftOverBalance + stockArray[i].leftOverBalance
    }
    reply.UnvestedAmount = leftOverBalance
    reply.TradeId = rand.Int()
    stockMarket = make(map[int] []stock)
    stockMarket[reply.TradeId] = stockArray
    return nil
}

func (t * StockMarketApp) HandleGetPortfolioRequest(args *CheckPortfolioRequestArgs, reply *CheckPortfolioRequestResult) error{
	stockArray := stockMarket[args.TradeId]
	gain := ""
	getYahooData(&stockArray)
	for i:=0;i<len(stockArray);i++{
		if(stockArray[i].oldNetAmount < stockArray[i].newNetAmount){
			gain = "+"
		}else if(stockArray[i].oldNetAmount > stockArray[i].newNetAmount){
			gain = "-"
		}
    	if(reply.Stocks != ""){
    		reply.Stocks = reply.Stocks + "," + stockArray[i].symbol + ":" + strconv.Itoa(stockArray[i].quantity) + ":" + gain + ":$" + fmt.Sprintf("%f", stockArray[i].newNetAmount)
    	}else{
    		reply.Stocks = stockArray[i].symbol + ":" + strconv.Itoa(stockArray[i].quantity) + ":" + gain + ":$" + fmt.Sprintf("%f", stockArray[i].newNetAmount)
    	}
    	reply.CurrentMarketValue = reply.CurrentMarketValue + (stockArray[i].newNetAmount * float32(stockArray[i].quantity))
    	reply.UnvestedAmount = reply.UnvestedAmount + stockArray[i].leftOverBalance
    }
	return nil
}

func getYahooData(stockArr *[]stock){
	var err error
	stockArray := *stockArr
	httpStockList := ""
	for i := 0; i <len(stockArray); i++ { 
		if i == len(stockArray) - 1{
			httpStockList =  httpStockList + "%22" + stockArray[i].symbol+ "%22"
		}else{
			httpStockList =  httpStockList + "%22" + stockArray[i].symbol + "%22%2C"
		}
	}	
	req, err := http.NewRequest("GET", "https://query.yahooapis.com/v1/public/yql?q=select%20symbol%2CBid%20from%20yahoo.finance.quotes%20where%20symbol%20in%20("+httpStockList+")%0A%09%09&format=json&diagnostics=true&env=http%3A%2F%2Fdatatables.org%2Falltables.env&callback=", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.SetBasicAuth("<token>", "x-oauth-basic")

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	var parsed map[string] interface{}
	err = json.Unmarshal(body, &parsed)
	result := parsed["query"].(map[string]interface{})["results"].(map[string]interface{})["quote"]
	for i:=0; i<len(stockArray);i++{
		var sym string;
		var bid string;
		if(len(stockArray) > 1){
			sym = result.([]interface{})[i].(map[string]interface{})["symbol"].(string)
			bid = result.([]interface{})[i].(map[string]interface{})["Bid"].(string)	
		}else{
			sym = result.(map[string]interface{})["symbol"].(string)
			bid = result.(map[string]interface{})["Bid"].(string)
		}
		if stockArray[i].symbol == sym{
			stockArray[i].oldNetAmount = stockArray[i].newNetAmount
			newNetAmount, err := strconv.ParseFloat(strings.Replace(bid, "%", "", -1), 32)
			if err != nil {
				log.Fatal(err)
			}
			stockArray[i].newNetAmount = float32(newNetAmount)
		}
	}
}

func (t *StockMarketApp) Error(args *Args, reply *Reply) error {
    panic("ERROR")
}

func startServer() {
    StockMarketApp := new(StockMarketApp)

    server := rpc.NewServer()
    server.Register(StockMarketApp)

    l, e := net.Listen("tcp", ":8222")
    if e != nil {
        log.Fatal("listen error:", e)
    }

    for {
        conn, err := l.Accept()
        if err != nil {
            log.Fatal(err)
        }
        go server.ServeCodec(jsonrpc.NewServerCodec(conn))
    }
}

func main() {
    go startServer()
	conn, err := net.Dial("tcp", "localhost:8222")
    if err != nil {
        panic(err)
    }
    defer conn.Close()
    c := jsonrpc.NewClient(conn)
    var reply Reply
    var args *Args
    fmt.Printf("Buying Stocks \nRequest:\nstockSymbolAndPercentage:")
	var stockData string
	fmt.Scanf("%s\n",&stockData)

	var unvestedAmount float32
	fmt.Printf("balance:")
	fmt.Scanf("%f\n",&unvestedAmount)
	
	args = &Args{stockData, unvestedAmount}
    err = c.Call("StockMarketApp.HandleBuyStockRequest", args, &reply)
    if err != nil {
        log.Fatal("StockMarketApp error:", err)
    }
    fmt.Printf("\nResponse:\nTrade Id: %d\nStock: %s\nUnvested Amount: %f", reply.TradeId, reply.Stocks, reply.UnvestedAmount)
    
    var checkPortfolioRequestArgs *CheckPortfolioRequestArgs
    var checkPortfolioRequestResult CheckPortfolioRequestResult
    fmt.Printf("\n\nChecking your portfolio (loss/gain) \nRequest:\ntradeId:")
	var tradeId int
	fmt.Scanf("%d\n",&tradeId)
	checkPortfolioRequestArgs = &CheckPortfolioRequestArgs{tradeId}
    err = c.Call("StockMarketApp.HandleGetPortfolioRequest", checkPortfolioRequestArgs, &checkPortfolioRequestResult)
    if err != nil {
        log.Fatal("StockMarketApp error:", err)
    }
    fmt.Printf("\nResponse:\nStocks: %s\nCurrentMarketValue: %f\nUnvestedAmount: %f", checkPortfolioRequestResult.Stocks, checkPortfolioRequestResult.CurrentMarketValue, checkPortfolioRequestResult.UnvestedAmount)   
}