package feargreedindex

const PrivateKey = "1318c69b804769a5472a284f80853d4c54914fbf289be1bac2480f82886b43bf"
const ContractAddress = "0x18FB13Fc8e321306Aa24897a2bFcb04830c2CbBd"
const ContractABI = `[
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
						"internalType": "uint256",
						"name": "value",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "value_classification",
						"type": "string"
					}
				],
				"indexed": false,
				"internalType": "struct FearAndGreedIndex.FormData",
				"name": "_formData",
				"type": "tuple"
			}
		],
		"name": "Recorded",
		"type": "event"
	},
	{
		"inputs": [
			{
				"components": [
					{
						"internalType": "uint256",
						"name": "timestamp",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "value",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "value_classification",
						"type": "string"
					}
				],
				"internalType": "struct FearAndGreedIndex.FormData",
				"name": "_formData",
				"type": "tuple"
			}
		],
		"name": "recordIndex",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "timestamp",
				"type": "uint256"
			}
		],
		"name": "getIndexByTime",
		"outputs": [
			{
				"components": [
					{
						"internalType": "uint256",
						"name": "timestamp",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "value",
						"type": "uint256"
					},
					{
						"internalType": "string",
						"name": "value_classification",
						"type": "string"
					}
				],
				"internalType": "struct FearAndGreedIndex.FormData",
				"name": "",
				"type": "tuple"
			}
		],
		"stateMutability": "view",
		"type": "function"
	}
]`
