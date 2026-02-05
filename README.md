# Nexus Orderbook DEX

Off-chain orderbook matching + on-chain settlement DEX.

Users sign EIP-712 orders (gasless). Go backend matches orders. Smart contract settles atomically.

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    Frontend     │────▶│   Go Backend    │────▶│  Smart Contract │
│   (Next.js)     │     │   (Gin + WS)    │     │   (Solidity)    │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │                       │
        │ EIP-712 Sign          │ Match Engine          │ Atomic Swap
        │ REST/WebSocket        │ Settlement Worker     │ Vault System
        │                       │ Event Indexer         │
        ▼                       ▼                       ▼
   ┌─────────┐           ┌───────────┐           ┌───────────┐
   │ Wallet  │           │ PostgreSQL│           │  On-Chain │
   │(MetaMask)│           │   Redis   │           │  Balances │
   └─────────┘           └───────────┘           └───────────┘
```

## Tech Stack

- **Contracts**: Solidity 0.8.24, Foundry, OpenZeppelin
- **Backend**: Go 1.24, Gin, go-ethereum, sqlx, go-redis
- **Frontend**: Next.js 15, React 19, wagmi, viem, Tailwind CSS
- **Infra**: PostgreSQL, Redis, Anvil/Sepolia

## Project Structure

```
nexus-orderbook-dex/
├── contracts/                 # Foundry project
│   ├── src/
│   │   ├── libraries/OrderTypes.sol    # EIP-712 type hash
│   │   ├── OrderValidator.sol          # Signature verification
│   │   ├── NexusOrderBook.sol          # Main contract
│   │   └── mocks/MockERC20.sol         # Test tokens
│   ├── test/NexusOrderBook.t.sol       # Test suite
│   └── script/Deploy.s.sol             # Deployment
├── backend/                   # Go backend
│   ├── cmd/server/main.go              # Entry point
│   ├── internal/
│   │   ├── config/                     # Environment config
│   │   ├── domain/                     # Order, Trade types
│   │   ├── orderbook/                  # Matching engine
│   │   ├── blockchain/                 # Settlement, Indexer
│   │   ├── repository/                 # PostgreSQL, Redis
│   │   ├── service/                    # Order orchestration
│   │   └── handler/                    # REST, WebSocket
│   ├── pkg/eip712/                     # EIP-712 Go impl
│   └── migrations/                     # SQL schema
├── frontend/                  # Next.js app
│   └── src/
│       ├── app/                        # Pages
│       ├── components/                 # UI components
│       ├── hooks/                      # React hooks
│       ├── lib/                        # Utilities
│       └── types/                      # TypeScript types
└── docker-compose.yml         # PostgreSQL, Redis, Anvil
```

## Prerequisites

### Required Software

```bash
# 1. Docker & Docker Compose
docker --version  # Docker version 24.0+
docker-compose --version  # v2.20+

# 2. Foundry (Solidity toolkit)
curl -L https://foundry.paradigm.xyz | bash
foundryup
forge --version  # forge 0.2.0+

# 3. Go
go version  # go1.24+

# 4. Node.js
node --version  # v20+
npm --version   # v10+
```

## Setup & Run

### Step 1: Clone and Setup

```bash
git clone https://github.com/inbeom02/nexus-orderbook-dex.git
cd nexus-orderbook-dex
```

### Step 2: Start Infrastructure

```bash
# Start PostgreSQL (port 5433), Redis (port 6380), Anvil (port 8545)
docker-compose up -d

# Verify all containers are running
docker-compose ps
# NAME                             STATUS
# nexus-orderbook-dex-anvil-1      Up
# nexus-orderbook-dex-postgres-1   Up
# nexus-orderbook-dex-redis-1      Up
```

### Step 3: Deploy Smart Contracts

```bash
cd contracts

# Install dependencies (forge-std, openzeppelin)
forge install

# Build contracts
forge build

# Deploy to local Anvil
PRIVATE_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 \
  forge script script/Deploy.s.sol --rpc-url http://localhost:8545 --broadcast

# Expected output:
# NexusOrderBook deployed at: 0x5FbDB2315678afecb367f032d93F642f64180aa3
# TokenA deployed at: 0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512
# TokenB deployed at: 0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0
```

### Step 4: Configure and Run Backend

```bash
cd ../backend

# Create .env file
cat > .env << 'EOF'
RPC_URL=http://localhost:8545
CHAIN_ID=31337
PRIVATE_KEY=ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
NEXUS_CONTRACT_ADDRESS=0x5FbDB2315678afecb367f032d93F642f64180aa3
TOKEN_A_ADDRESS=0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512
TOKEN_B_ADDRESS=0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0
DATABASE_URL=postgres://nexus:nexus_dev@localhost:5433/nexus_orderbook?sslmode=disable
REDIS_URL=localhost:6380
SERVER_PORT=8080
EOF

# Install dependencies and run
go mod download
go run cmd/server/main.go

# Expected output:
# [GIN-debug] POST   /api/orders
# [GIN-debug] GET    /api/orders/:address
# [GIN-debug] DELETE /api/orders/:id
# [GIN-debug] GET    /api/orderbook
# [GIN-debug] GET    /api/trades
# [GIN-debug] GET    /ws
# [GIN-debug] Listening and serving HTTP on :8080
```

### Step 5: Run Frontend

```bash
# In a new terminal
cd frontend

# Install dependencies
npm install

# Start development server
npm run dev

# Open http://localhost:3000
```

### Step 6: Setup MetaMask

1. Open MetaMask browser extension
2. Add Network:
   - Network Name: `Localhost`
   - RPC URL: `http://127.0.0.1:8545`
   - Chain ID: `31337`
   - Currency Symbol: `ETH`
3. Import Test Account:
   - Private Key: `0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d`
   - This is Anvil's default account #1 (Buyer)

## Testing

### 1. Smart Contract Tests

```bash
cd contracts

# Run all tests with verbose output
forge test -vvv

# Expected: 15 tests passed
# - test_Deposit
# - test_Withdraw
# - test_SettleMatch
# - test_SettlePartialFill
# - test_CancelOrder
# - ... and more
```

### 2. Backend Unit Tests

```bash
cd backend

# Run orderbook matching engine tests
go test ./internal/orderbook/ -v

# Expected: 6 tests passed
# - TestAddBuyOrder_NoMatch
# - TestAddSellOrder_NoMatch
# - TestMatchBuyAndSell
# - TestPartialMatch
# - TestPriceIncompatible
# - TestCancelOrder

# Run EIP-712 signature tests
go test ./pkg/eip712/ -v

# Expected: 3 tests passed
```

### 3. On-Chain E2E Test (Foundry Script)

```bash
cd contracts

# Run complete on-chain flow test
forge script script/TestFlow.s.sol --rpc-url http://localhost:8545 --broadcast

# This test:
# 1. Mints tokens to buyer and seller
# 2. Deposits tokens to vault
# 3. Creates and signs buy/sell orders
# 4. Settles the match on-chain
# 5. Verifies final balances
# 6. Withdraws tokens

# Expected output:
# === Nexus DEX E2E Test ===
# 1. Minting tokens...
# 2. Depositing to vault...
# 3. Creating orders...
# 4. Settling match...
#    Match settled!
# 5. Final vault balances:
#    Buyer TKA: 100
#    Seller TKB: 200
# === Test Complete! ===
```

### 4. API + Settlement E2E Test

First, prepare test accounts with tokens in vault:

```bash
cd contracts
forge script script/FullE2E.s.sol:FullE2E --rpc-url http://localhost:8545 --broadcast

# Expected:
# Buyer vault TKB: 5000+
# Seller vault TKA: 5000+
```

Then run the API test (with backend running):

```bash
cd backend
go run scripts/test_settlement.go

# This test:
# 1. Submits sell order via REST API
# 2. Submits matching buy order
# 3. Waits for on-chain settlement
# 4. Verifies trade with tx hash

# Expected output:
# === Full Flow with On-Chain Settlement ===
# 1. Seller submits: Sell 100 TKA @ 2 TKB/TKA
# 2. Buyer submits: Buy 100 TKA @ 2 TKB/TKA
# 3. Waiting for on-chain settlement...
# 4. Checking trades...
#    Latest Trade:
#      Base Amount: 100.0000 TKA
#      Quote Amount: 200.0000 TKB
#      Price: 2
#      Settled On-Chain: true
#      Tx Hash: 0x...
# === Test Complete ===
```

### 5. Manual Frontend Test

1. Open http://localhost:3000
2. Click "Connect Wallet" and connect MetaMask
3. In the Vault section:
   - Select TKA, click "mint", enter "1000", click "mint"
   - Click "deposit", enter "500", click "deposit"
4. Repeat for TKB token
5. In the Order Form:
   - Select "Sell"
   - Price: 2, Amount: 100
   - Click "Sell TKA"
   - Sign the EIP-712 message in MetaMask
6. Switch to another account and place a buy order
7. Watch the orderbook update in real-time
8. Check trade history after match

### 6. Verify On-Chain State

```bash
# Check vault balances using cast
cast call 0x5FbDB2315678afecb367f032d93F642f64180aa3 \
  "getBalance(address,address)(uint256)" \
  <USER_ADDRESS> \
  0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512 \
  --rpc-url http://localhost:8545

# Check order fill amount
cast call 0x5FbDB2315678afecb367f032d93F642f64180aa3 \
  "getOrderFill(bytes32)(uint256)" \
  <ORDER_HASH> \
  --rpc-url http://localhost:8545
```

## Cleanup

```bash
# Stop all services
docker-compose down

# Remove volumes (database data)
docker-compose down -v

# Stop backend: Ctrl+C in terminal
# Stop frontend: Ctrl+C in terminal
```

## Troubleshooting

### Port already in use

```bash
# Check what's using the port
lsof -i :8545  # Anvil
lsof -i :5433  # PostgreSQL
lsof -i :6380  # Redis
lsof -i :8080  # Backend

# Kill the process
kill -9 <PID>
```

### Contract deployment fails

```bash
# Make sure Anvil is running
curl http://localhost:8545 -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# Should return: {"jsonrpc":"2.0","id":1,"result":"0x..."}
```

### Backend can't connect to database

```bash
# Check PostgreSQL is running
docker-compose logs postgres

# Test connection
psql "postgres://nexus:nexus_dev@localhost:5433/nexus_orderbook?sslmode=disable" -c "SELECT 1"
```

### MetaMask transaction stuck

```bash
# Reset Anvil state (this will require redeploying contracts)
docker-compose restart anvil

# In MetaMask: Settings > Advanced > Clear activity tab data
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/orders` | Submit signed order |
| GET | `/api/orders/:address` | Get user's orders |
| DELETE | `/api/orders/:id` | Cancel order |
| GET | `/api/orderbook?pair=TKA-TKB` | Get orderbook snapshot |
| GET | `/api/trades?pair=TKA-TKB` | Get recent trades |
| WS | `/ws?pair=TKA-TKB` | Real-time orderbook updates |

## Order Flow

1. **Sign Order**: User signs EIP-712 typed data (gasless)
2. **Submit**: Frontend POSTs signed order to backend
3. **Verify**: Backend verifies signature matches maker
4. **Match**: Orderbook engine matches against resting orders
5. **Settle**: Settlement worker submits `settleMatch()` tx on-chain
6. **Update**: Contract atomically swaps vault balances
7. **Index**: Event indexer catches `TradeSettled` event
8. **Notify**: WebSocket broadcasts update to subscribers

## Smart Contract

### Key Functions

```solidity
// Deposit tokens to vault
function deposit(address token, uint256 amount) external;

// Withdraw tokens from vault
function withdraw(address token, uint256 amount) external;

// Settle matched orders (owner only)
function settleMatch(
    Order calldata buyOrder,
    bytes calldata buySig,
    Order calldata sellOrder,
    bytes calldata sellSig,
    uint256 fillAmount
) external;

// Cancel specific order
function cancelOrder(Order calldata order) external;

// Bulk cancel by incrementing min nonce
function incrementMinNonce(uint256 newMinNonce) external;
```

### EIP-712 Order Structure

```solidity
struct Order {
    address maker;
    address tokenSell;
    address tokenBuy;
    uint256 amountSell;
    uint256 amountBuy;
    uint256 expiry;
    uint256 nonce;
    uint256 salt;
}
```

## Testing

### Contract Tests

```bash
cd contracts
forge test -vvv
```

### Backend Tests

```bash
cd backend
go test ./...
```

### E2E Test

```bash
cd contracts
# Prepare accounts with vault deposits
forge script script/FullE2E.s.sol:FullE2E --rpc-url http://localhost:8545 --broadcast

cd ../backend
# Run API + settlement test
go run scripts/test_settlement.go
```

## Configuration

### Environment Variables

```env
# Blockchain
RPC_URL=http://localhost:8545
CHAIN_ID=31337
PRIVATE_KEY=<deployer_private_key>
NEXUS_CONTRACT_ADDRESS=<deployed_address>
TOKEN_A_ADDRESS=<token_a_address>
TOKEN_B_ADDRESS=<token_b_address>

# Database
DATABASE_URL=postgres://nexus:nexus_dev@localhost:5433/nexus_orderbook?sslmode=disable

# Redis
REDIS_URL=localhost:6380

# Server
SERVER_PORT=8080
```

## Design Decisions

- **Off-chain matching + On-chain settlement**: Combines CEX-like speed with DEX trustlessness
- **amountSell/amountBuy vs price/quantity**: Avoids floating point on-chain
- **Settlement worker with Go channel**: Serializes nonce management for tx submission
- **EIP-712 domain separator**: Identical across contract, Go backend, and frontend
- **NUMERIC(78,0) in PostgreSQL**: Stores uint256 values without precision loss
- **Redis sorted sets + pub/sub**: Real-time orderbook cache and WebSocket fanout

## License

MIT
