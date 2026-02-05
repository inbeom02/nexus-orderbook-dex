// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Script, console} from "forge-std/Script.sol";
import {NexusOrderBook} from "../src/NexusOrderBook.sol";
import {MockERC20} from "../src/mocks/MockERC20.sol";
import {OrderTypes} from "../src/libraries/OrderTypes.sol";

contract TestFlow is Script {
    // Anvil default accounts
    uint256 constant DEPLOYER_PK = 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80;
    uint256 constant BUYER_PK = 0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d;
    uint256 constant SELLER_PK = 0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a;

    NexusOrderBook orderbook;
    MockERC20 tokenA;
    MockERC20 tokenB;

    address deployer;
    address buyer;
    address seller;

    function setUp() public {
        deployer = vm.addr(DEPLOYER_PK);
        buyer = vm.addr(BUYER_PK);
        seller = vm.addr(SELLER_PK);

        orderbook = NexusOrderBook(0x5FbDB2315678afecb367f032d93F642f64180aa3);
        tokenA = MockERC20(0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512);
        tokenB = MockERC20(0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0);
    }

    function run() external {
        console.log("=== Nexus DEX E2E Test ===");
        console.log("Deployer:", deployer);
        console.log("Buyer:", buyer);
        console.log("Seller:", seller);
        console.log("");

        // 1. Mint tokens
        console.log("1. Minting tokens...");
        vm.startBroadcast(DEPLOYER_PK);
        tokenA.mint(buyer, 1000 ether);
        tokenA.mint(seller, 1000 ether);
        tokenB.mint(buyer, 1000 ether);
        tokenB.mint(seller, 1000 ether);
        vm.stopBroadcast();
        console.log("   Buyer TKA:", tokenA.balanceOf(buyer) / 1e18);
        console.log("   Buyer TKB:", tokenB.balanceOf(buyer) / 1e18);
        console.log("   Seller TKA:", tokenA.balanceOf(seller) / 1e18);
        console.log("   Seller TKB:", tokenB.balanceOf(seller) / 1e18);
        console.log("");

        // 2. Approve and deposit
        console.log("2. Depositing to vault...");
        vm.startBroadcast(BUYER_PK);
        tokenB.approve(address(orderbook), type(uint256).max);
        orderbook.deposit(address(tokenB), 500 ether);
        vm.stopBroadcast();

        vm.startBroadcast(SELLER_PK);
        tokenA.approve(address(orderbook), type(uint256).max);
        orderbook.deposit(address(tokenA), 500 ether);
        vm.stopBroadcast();

        console.log("   Buyer vault TKB:", orderbook.getBalance(buyer, address(tokenB)) / 1e18);
        console.log("   Seller vault TKA:", orderbook.getBalance(seller, address(tokenA)) / 1e18);
        console.log("");

        // 3. Create and sign orders
        console.log("3. Creating orders...");

        // Buy order: buyer wants 100 TKA, willing to pay 200 TKB (price = 2)
        OrderTypes.Order memory buyOrder = OrderTypes.Order({
            maker: buyer,
            tokenSell: address(tokenB),
            tokenBuy: address(tokenA),
            amountSell: 200 ether,
            amountBuy: 100 ether,
            expiry: block.timestamp + 1 hours,
            nonce: 0,
            salt: 12345
        });

        // Sell order: seller selling 100 TKA, wants 200 TKB (price = 2)
        OrderTypes.Order memory sellOrder = OrderTypes.Order({
            maker: seller,
            tokenSell: address(tokenA),
            tokenBuy: address(tokenB),
            amountSell: 100 ether,
            amountBuy: 200 ether,
            expiry: block.timestamp + 1 hours,
            nonce: 0,
            salt: 67890
        });

        // Sign orders
        bytes memory buySig = _signOrder(buyOrder, BUYER_PK);
        bytes memory sellSig = _signOrder(sellOrder, SELLER_PK);

        console.log("   Buy order: 100 TKA @ 2 TKB/TKA");
        console.log("   Sell order: 100 TKA @ 2 TKB/TKA");
        console.log("");

        // 4. Settle match (as deployer/owner)
        console.log("4. Settling match...");
        vm.startBroadcast(DEPLOYER_PK);
        orderbook.settleMatch(buyOrder, buySig, sellOrder, sellSig, 100 ether);
        vm.stopBroadcast();
        console.log("   Match settled!");
        console.log("");

        // 5. Check final balances
        console.log("5. Final vault balances:");
        console.log("   Buyer TKA:", orderbook.getBalance(buyer, address(tokenA)) / 1e18);
        console.log("   Buyer TKB:", orderbook.getBalance(buyer, address(tokenB)) / 1e18);
        console.log("   Seller TKA:", orderbook.getBalance(seller, address(tokenA)) / 1e18);
        console.log("   Seller TKB:", orderbook.getBalance(seller, address(tokenB)) / 1e18);
        console.log("");

        // 6. Withdraw
        console.log("6. Withdrawing from vault...");
        vm.startBroadcast(BUYER_PK);
        orderbook.withdraw(address(tokenA), 100 ether);
        vm.stopBroadcast();

        vm.startBroadcast(SELLER_PK);
        orderbook.withdraw(address(tokenB), 200 ether);
        vm.stopBroadcast();

        console.log("   Buyer wallet TKA:", tokenA.balanceOf(buyer) / 1e18);
        console.log("   Seller wallet TKB:", tokenB.balanceOf(seller) / 1e18);
        console.log("");
        console.log("=== Test Complete! ===");
    }

    function _signOrder(OrderTypes.Order memory order, uint256 pk) internal view returns (bytes memory) {
        bytes32 structHash = OrderTypes.hash(order);
        bytes32 digest = keccak256(
            abi.encodePacked("\x19\x01", orderbook.DOMAIN_SEPARATOR(), structHash)
        );
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(pk, digest);
        return abi.encodePacked(r, s, v);
    }
}
