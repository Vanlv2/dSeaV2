package ohlcvConfig

const ContractABIWeek = `[
	{
		"inputs": [
			{
				"components": [
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
					},
					{
						"internalType": "uint256",
						"name": "openTime",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "open",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "high",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "low",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "close",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "volume",
						"type": "string"
					},
					{
						"internalType": "uint256",
						"name": "closeTime",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "quoteAssetVolume",
						"type": "string"
					},
					{
						"internalType": "uint256",
						"name": "numberOfTrades",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "takerBuyBaseVol",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "takerBuyQuoteVol",
						"type": "string"
					}
				],
				"internalType": "struct OHLCVWeek.FormData",
				"name": "formData",
				"type": "tuple"
			}
		],
		"name": "recordData",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"components": [
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
					},
					{
						"internalType": "uint256",
						"name": "openTime",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "open",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "high",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "low",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "close",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "volume",
						"type": "string"
					},
					{
						"internalType": "uint256",
						"name": "closeTime",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "quoteAssetVolume",
						"type": "string"
					},
					{
						"internalType": "uint256",
						"name": "numberOfTrades",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "takerBuyBaseVol",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "takerBuyQuoteVol",
						"type": "string"
					}
				],
				"indexed": false,
				"internalType": "struct OHLCVWeek.FormData",
				"name": "formData",
				"type": "tuple"
			}
		],
		"name": "Recorded",
		"type": "event"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "timestamp",
				"type": "uint256"
			}
		],
		"name": "getAllSymbolByTime",
		"outputs": [
			{
				"components": [
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
					},
					{
						"internalType": "uint256",
						"name": "openTime",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "open",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "high",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "low",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "close",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "volume",
						"type": "string"
					},
					{
						"internalType": "uint256",
						"name": "closeTime",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "quoteAssetVolume",
						"type": "string"
					},
					{
						"internalType": "uint256",
						"name": "numberOfTrades",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "takerBuyBaseVol",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "takerBuyQuoteVol",
						"type": "string"
					}
				],
				"internalType": "struct OHLCVWeek.FormData[]",
				"name": "",
				"type": "tuple[]"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "endTimestamp",
				"type": "uint256"
			},
			{
				"internalType": "string",
				"name": "tokenSymbol",
				"type": "string"
			}
		],
		"name": "getData7ItemsAgo",
		"outputs": [
			{
				"components": [
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
					},
					{
						"internalType": "uint256",
						"name": "openTime",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "open",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "high",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "low",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "close",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "volume",
						"type": "string"
					},
					{
						"internalType": "uint256",
						"name": "closeTime",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "quoteAssetVolume",
						"type": "string"
					},
					{
						"internalType": "uint256",
						"name": "numberOfTrades",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "takerBuyBaseVol",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "takerBuyQuoteVol",
						"type": "string"
					}
				],
				"internalType": "struct OHLCVWeek.FormData[]",
				"name": "",
				"type": "tuple[]"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "startTimestamp",
				"type": "uint256"
			},
			{
				"internalType": "uint256",
				"name": "endTimestamp",
				"type": "uint256"
			},
			{
				"internalType": "string",
				"name": "tokenSymbol",
				"type": "string"
			}
		],
		"name": "getDataInRange",
		"outputs": [
			{
				"components": [
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
					},
					{
						"internalType": "uint256",
						"name": "openTime",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "open",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "high",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "low",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "close",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "volume",
						"type": "string"
					},
					{
						"internalType": "uint256",
						"name": "closeTime",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "quoteAssetVolume",
						"type": "string"
					},
					{
						"internalType": "uint256",
						"name": "numberOfTrades",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "takerBuyBaseVol",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "takerBuyQuoteVol",
						"type": "string"
					}
				],
				"internalType": "struct OHLCVWeek.FormData[]",
				"name": "",
				"type": "tuple[]"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "timestamp",
				"type": "uint256"
			},
			{
				"internalType": "string",
				"name": "symbol",
				"type": "string"
			}
		],
		"name": "getSymbolByTime",
		"outputs": [
			{
				"components": [
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
					},
					{
						"internalType": "uint256",
						"name": "openTime",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "open",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "high",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "low",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "close",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "volume",
						"type": "string"
					},
					{
						"internalType": "uint256",
						"name": "closeTime",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "quoteAssetVolume",
						"type": "string"
					},
					{
						"internalType": "uint256",
						"name": "numberOfTrades",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "takerBuyBaseVol",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "takerBuyQuoteVol",
						"type": "string"
					}
				],
				"internalType": "struct OHLCVWeek.FormData",
				"name": "",
				"type": "tuple"
			}
		],
		"stateMutability": "view",
		"type": "function"
	}
]`
