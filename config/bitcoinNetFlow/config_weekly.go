package smartcontract

import (
	structData "main/services/bitcoinNetFlow/smart_contract"
)

func SetConfigContractWeekly() structData.ConfigContract {
	config.ContractAddress = "0x287189dEF35ab0EE1501979Ea7DE761c04990Ea8"
	config.ContractABI = `[
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
	}
]`
	return config
}
