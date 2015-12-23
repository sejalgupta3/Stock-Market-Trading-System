its a virtual stock trading system for whoever wants to learn how to invest in stocks.
The system uses real-time pricing via Yahoo finance API and supports USD currency only. 

The system has two features:
1)Buying stocks
Request:
“stockSymbolAndPercentage”: string (E.g. “GOOG:50%,YHOO:50%”)
“budget” : float32

Response:
“tradeId”: number
“stocks”: string (E.g. “GOOG:100:$500.25”, “YHOO:200:$31.40”)
“unvestedAmount”: float32

2)Checking your portfolio (loss/gain)
Request:
“tradeId”: number

Response:
“stocks”: string (E.g. “GOOG:100:+$520.25”, “YHOO:200:-$30.40”)
“currentMarketValue” : float32
“unvestedAmount”: float32

The system has 2 components: client and server.
server: the trading engine has JSON-RPC interface for the above features.
client: the JSON-RPC client takes command line input and send requests the server.
