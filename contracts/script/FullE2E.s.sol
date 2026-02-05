// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Script, console} from "forge-std/Script.sol";
import {NexusOrderBook} from "../src/NexusOrderBook.sol";
import {MockERC20} from "../src/mocks/MockERC20.sol";

contract FullE2E is Script {
    // Anvil accounts 1 and 2
    uint256 constant BUYER_PK = 0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d;
    uint256 constant SELLER_PK = 0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a;

    NexusOrderBook orderbook = NexusOrderBook(0x5FbDB2315678afecb367f032d93F642f64180aa3);
    MockERC20 tokenA = MockERC20(0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512);
    MockERC20 tokenB = MockERC20(0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0);

    function run() external {
        address buyer = vm.addr(BUYER_PK);
        address seller = vm.addr(SELLER_PK);

        console.log("Preparing accounts for API testing...");
        console.log("Buyer:", buyer);
        console.log("Seller:", seller);

        // Mint and deposit for both accounts
        vm.startBroadcast(BUYER_PK);
        tokenB.mint(buyer, 10000 ether);
        tokenB.approve(address(orderbook), type(uint256).max);
        orderbook.deposit(address(tokenB), 5000 ether);
        vm.stopBroadcast();

        vm.startBroadcast(SELLER_PK);
        tokenA.mint(seller, 10000 ether);
        tokenA.approve(address(orderbook), type(uint256).max);
        orderbook.deposit(address(tokenA), 5000 ether);
        vm.stopBroadcast();

        console.log("Buyer vault TKB:", orderbook.getBalance(buyer, address(tokenB)) / 1e18);
        console.log("Seller vault TKA:", orderbook.getBalance(seller, address(tokenA)) / 1e18);
        console.log("Ready for API testing!");
    }
}
