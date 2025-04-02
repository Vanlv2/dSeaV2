package config

// ngrok config add-authtoken 2uDHKB3y43fFC1G4jX6CZjuWM1A_4ipy24dhQvogqKcHmhg21

const PrivateKey = "1318c69b804769a5472a284f80853d4c54914fbf289be1bac2480f82886b43bf"

const ContractAddressDate = "0x97EFefB2254d4Adc4047319efF6Ce01232D14e92"
const ContractABIDate = `[
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
						"internalType": "uint256",
						"name": "timestamp",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
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
					}
				],
				"indexed": false,
				"internalType": "struct OHLCVDay.FormData",
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
						"internalType": "uint256",
						"name": "timestamp",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
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
					}
				],
				"internalType": "struct OHLCVDay.FormData[]",
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
						"internalType": "uint256",
						"name": "timestamp",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
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
					}
				],
				"internalType": "struct OHLCVDay.FormData[]",
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
						"internalType": "uint256",
						"name": "timestamp",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
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
					}
				],
				"internalType": "struct OHLCVDay.FormData[]",
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
						"internalType": "uint256",
						"name": "timestamp",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
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
					}
				],
				"internalType": "struct OHLCVDay.FormData",
				"name": "",
				"type": "tuple"
			}
		],
		"stateMutability": "view",
		"type": "function"
	}
]`

const ContractAddressWeek = "0x73D6FcDEe2900eFdBDb1c9faBDfe2737BBA7cae0"
const ContractABIWeek = `[
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
						"internalType": "uint256",
						"name": "timestamp",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
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
						"internalType": "uint256",
						"name": "timestamp",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
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
						"internalType": "uint256",
						"name": "timestamp",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
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
						"internalType": "uint256",
						"name": "timestamp",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
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
						"internalType": "uint256",
						"name": "timestamp",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
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

const ContractAddressMonth = "0xa877CC0Aea97Db42b24d02e3eA892949cB50E22f"
const ContractABIMonth = `[
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
						"internalType": "uint256",
						"name": "timestamp",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
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
					}
				],
				"indexed": false,
				"internalType": "struct OHLCVMonth.FormData",
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
						"internalType": "uint256",
						"name": "timestamp",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
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
					}
				],
				"internalType": "struct OHLCVMonth.FormData[]",
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
						"internalType": "uint256",
						"name": "timestamp",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
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
					}
				],
				"internalType": "struct OHLCVMonth.FormData[]",
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
						"internalType": "uint256",
						"name": "timestamp",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
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
					}
				],
				"internalType": "struct OHLCVMonth.FormData[]",
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
						"internalType": "uint256",
						"name": "timestamp",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "symbol",
						"type": "string"
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
					}
				],
				"internalType": "struct OHLCVMonth.FormData",
				"name": "",
				"type": "tuple"
			}
		],
		"stateMutability": "view",
		"type": "function"
	}
]`
