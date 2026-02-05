// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Script, console} from "forge-std/Script.sol";
import {NexusOrderBook} from "../src/NexusOrderBook.sol";
import {MockERC20} from "../src/mocks/MockERC20.sol";

contract Deploy is Script {
    function run() external {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        vm.startBroadcast(deployerPrivateKey);

        NexusOrderBook orderbook = new NexusOrderBook();
        console.log("NexusOrderBook deployed at:", address(orderbook));

        MockERC20 tokenA = new MockERC20("Token A", "TKA", 18);
        console.log("TokenA deployed at:", address(tokenA));

        MockERC20 tokenB = new MockERC20("Token B", "TKB", 18);
        console.log("TokenB deployed at:", address(tokenB));

        vm.stopBroadcast();
    }
}
