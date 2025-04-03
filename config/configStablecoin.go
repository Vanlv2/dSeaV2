package config

const PrivateKeyS = "1318c69b804769a5472a284f80853d4c54914fbf289be1bac2480f82886b43bf"

const ContractAddressDateS = "0x9B0CD5b73c2087F2F0242BEC2A0647828EC777E2"
const ContractABIDateS = `[
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "timestamp",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "string",
				"name": "tokenSymbol",
				"type": "string"
			},
			{
				"indexed": false,
				"internalType": "string",
				"name": "exchangeName",
				"type": "string"
			},
			{
				"indexed": false,
				"internalType": "string",
				"name": "incoming",
				"type": "string"
			},
			{
				"indexed": false,
				"internalType": "string",
				"name": "outgoing",
				"type": "string"
			},
			{
				"indexed": false,
				"internalType": "string",
				"name": "balance",
				"type": "string"
			}
		],
		"name": "FlowTotalRecorded",
		"type": "event"
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
				"name": "incoming",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "outgoing",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "balance",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "tokenSymbol",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "exchangeName",
				"type": "string"
			}
		],
		"name": "recordFlow",
		"outputs": [],
		"stateMutability": "nonpayable",
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
			},
			{
				"internalType": "string",
				"name": "exchangeName",
				"type": "string"
			}
		],
		"name": "getFlow7ItemsAgo",
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
						"name": "incoming",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "outgoing",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "balance",
						"type": "string"
					}
				],
				"internalType": "struct FlowTrackerByDay.FlowData[]",
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
				"name": "tokenSymbol",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "exchangeName",
				"type": "string"
			}
		],
		"name": "getFlowbyTime",
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
						"name": "incoming",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "outgoing",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "balance",
						"type": "string"
					}
				],
				"internalType": "struct FlowTrackerByDay.FlowData",
				"name": "",
				"type": "tuple"
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
			},
			{
				"internalType": "string",
				"name": "exchangeName",
				"type": "string"
			}
		],
		"name": "getFlowInRange",
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
						"name": "incoming",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "outgoing",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "balance",
						"type": "string"
					}
				],
				"internalType": "struct FlowTrackerByDay.FlowData[]",
				"name": "",
				"type": "tuple[]"
			}
		],
		"stateMutability": "view",
		"type": "function"
	}
]`
const ContractAddressWeekS = "0x7Fa53BcaCC9Cd1eC721003c5041613bFAbC88e17"
const ContractABIWeekS = `[
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "timestamp",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "string",
				"name": "tokenSymbol",
				"type": "string"
			},
			{
				"indexed": false,
				"internalType": "string",
				"name": "exchangeName",
				"type": "string"
			},
			{
				"indexed": false,
				"internalType": "string",
				"name": "incoming",
				"type": "string"
			},
			{
				"indexed": false,
				"internalType": "string",
				"name": "outgoing",
				"type": "string"
			},
			{
				"indexed": false,
				"internalType": "string",
				"name": "balance",
				"type": "string"
			}
		],
		"name": "FlowTotalRecorded",
		"type": "event"
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
			},
			{
				"internalType": "string",
				"name": "exchangeName",
				"type": "string"
			}
		],
		"name": "getFlow7ItemsAgo",
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
						"name": "incoming",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "outgoing",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "balance",
						"type": "string"
					}
				],
				"internalType": "struct FlowTrackerByWeek.FlowData[]",
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
			},
			{
				"internalType": "string",
				"name": "exchangeName",
				"type": "string"
			}
		],
		"name": "getFlowInRange",
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
						"name": "incoming",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "outgoing",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "balance",
						"type": "string"
					}
				],
				"internalType": "struct FlowTrackerByWeek.FlowData[]",
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
				"name": "tokenSymbol",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "exchangeName",
				"type": "string"
			}
		],
		"name": "getFlowbyTime",
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
						"name": "incoming",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "outgoing",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "balance",
						"type": "string"
					}
				],
				"internalType": "struct FlowTrackerByWeek.FlowData",
				"name": "",
				"type": "tuple"
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
				"name": "incoming",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "outgoing",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "balance",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "tokenSymbol",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "exchangeName",
				"type": "string"
			}
		],
		"name": "recordFlow",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]`
const ContractAddressMonthS = "0x70696Fd4B00ce4a9706b40EDF51ff3459A7950Cf"
const ContractABIMonthS = `[
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "timestamp",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "string",
				"name": "tokenSymbol",
				"type": "string"
			},
			{
				"indexed": false,
				"internalType": "string",
				"name": "exchangeName",
				"type": "string"
			},
			{
				"indexed": false,
				"internalType": "string",
				"name": "incoming",
				"type": "string"
			},
			{
				"indexed": false,
				"internalType": "string",
				"name": "outgoing",
				"type": "string"
			},
			{
				"indexed": false,
				"internalType": "string",
				"name": "balance",
				"type": "string"
			}
		],
		"name": "FlowTotalRecorded",
		"type": "event"
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
				"name": "incoming",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "outgoing",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "balance",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "tokenSymbol",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "exchangeName",
				"type": "string"
			}
		],
		"name": "recordFlow",
		"outputs": [],
		"stateMutability": "nonpayable",
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
			},
			{
				"internalType": "string",
				"name": "exchangeName",
				"type": "string"
			}
		],
		"name": "getFlow7ItemsAgo",
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
						"name": "incoming",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "outgoing",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "balance",
						"type": "string"
					}
				],
				"internalType": "struct FlowTrackerByMonth.FlowData[]",
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
				"name": "tokenSymbol",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "exchangeName",
				"type": "string"
			}
		],
		"name": "getFlowbyTime",
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
						"name": "incoming",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "outgoing",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "balance",
						"type": "string"
					}
				],
				"internalType": "struct FlowTrackerByMonth.FlowData",
				"name": "",
				"type": "tuple"
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
			},
			{
				"internalType": "string",
				"name": "exchangeName",
				"type": "string"
			}
		],
		"name": "getFlowInRange",
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
						"name": "incoming",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "outgoing",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "balance",
						"type": "string"
					}
				],
				"internalType": "struct FlowTrackerByMonth.FlowData[]",
				"name": "",
				"type": "tuple[]"
			}
		],
		"stateMutability": "view",
		"type": "function"
	}
]`
