// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test, console} from "forge-std/Test.sol";
import {NexusOrderBook} from "../src/NexusOrderBook.sol";
import {MockERC20} from "../src/mocks/MockERC20.sol";
import {OrderTypes} from "../src/libraries/OrderTypes.sol";

contract NexusOrderBookTest is Test {
    NexusOrderBook public orderbook;
    MockERC20 public tokenA;
    MockERC20 public tokenB;

    address public owner;
    uint256 public buyerPk;
    address public buyer;
    uint256 public sellerPk;
    address public seller;

    uint256 constant INITIAL_BALANCE = 1000 ether;

    function setUp() public {
        owner = address(this);
        buyerPk = 0x1;
        buyer = vm.addr(buyerPk);
        sellerPk = 0x2;
        seller = vm.addr(sellerPk);

        orderbook = new NexusOrderBook();
        tokenA = new MockERC20("Token A", "TKA", 18);
        tokenB = new MockERC20("Token B", "TKB", 18);

        // Mint tokens
        tokenA.mint(buyer, INITIAL_BALANCE);
        tokenA.mint(seller, INITIAL_BALANCE);
        tokenB.mint(buyer, INITIAL_BALANCE);
        tokenB.mint(seller, INITIAL_BALANCE);

        // Approve orderbook
        vm.prank(buyer);
        tokenA.approve(address(orderbook), type(uint256).max);
        vm.prank(buyer);
        tokenB.approve(address(orderbook), type(uint256).max);
        vm.prank(seller);
        tokenA.approve(address(orderbook), type(uint256).max);
        vm.prank(seller);
        tokenB.approve(address(orderbook), type(uint256).max);
    }

    // ============ Deposit / Withdraw ============

    function test_Deposit() public {
        vm.prank(buyer);
        orderbook.deposit(address(tokenA), 100 ether);
        assertEq(orderbook.getBalance(buyer, address(tokenA)), 100 ether);
    }

    function test_Withdraw() public {
        vm.prank(buyer);
        orderbook.deposit(address(tokenA), 100 ether);
        vm.prank(buyer);
        orderbook.withdraw(address(tokenA), 50 ether);
        assertEq(orderbook.getBalance(buyer, address(tokenA)), 50 ether);
    }

    function test_WithdrawInsufficientBalance() public {
        vm.prank(buyer);
        orderbook.deposit(address(tokenA), 100 ether);
        vm.prank(buyer);
        vm.expectRevert("Insufficient balance");
        orderbook.withdraw(address(tokenA), 200 ether);
    }

    // ============ Settlement ============

    function _signOrder(
        OrderTypes.Order memory order,
        uint256 privateKey
    ) internal view returns (bytes memory) {
        bytes32 structHash = OrderTypes.hash(order);
        bytes32 digest = keccak256(
            abi.encodePacked("\x19\x01", orderbook.DOMAIN_SEPARATOR(), structHash)
        );
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(privateKey, digest);
        return abi.encodePacked(r, s, v);
    }

    function _createBuyOrder(
        uint256 amountSell,
        uint256 amountBuy
    ) internal view returns (OrderTypes.Order memory) {
        return OrderTypes.Order({
            maker: buyer,
            tokenSell: address(tokenB), // buyer sells tokenB (quote)
            tokenBuy: address(tokenA),  // buyer wants tokenA (base)
            amountSell: amountSell,
            amountBuy: amountBuy,
            expiry: block.timestamp + 1 hours,
            nonce: 0,
            salt: 1
        });
    }

    function _createSellOrder(
        uint256 amountSell,
        uint256 amountBuy
    ) internal view returns (OrderTypes.Order memory) {
        return OrderTypes.Order({
            maker: seller,
            tokenSell: address(tokenA), // seller sells tokenA (base)
            tokenBuy: address(tokenB),  // seller wants tokenB (quote)
            amountSell: amountSell,
            amountBuy: amountBuy,
            expiry: block.timestamp + 1 hours,
            nonce: 0,
            salt: 2
        });
    }

    function test_SettleMatch() public {
        // Deposit
        vm.prank(buyer);
        orderbook.deposit(address(tokenB), 500 ether);
        vm.prank(seller);
        orderbook.deposit(address(tokenA), 500 ether);

        // Buy order: buy 100 TKA for 200 TKB (price = 2 TKB/TKA)
        OrderTypes.Order memory buyOrder = _createBuyOrder(200 ether, 100 ether);
        bytes memory buySig = _signOrder(buyOrder, buyerPk);

        // Sell order: sell 100 TKA for 200 TKB (price = 2 TKB/TKA)
        OrderTypes.Order memory sellOrder = _createSellOrder(100 ether, 200 ether);
        bytes memory sellSig = _signOrder(sellOrder, sellerPk);

        // Settle full fill (100 TKA)
        orderbook.settleMatch(buyOrder, buySig, sellOrder, sellSig, 100 ether);

        // Check balances
        assertEq(orderbook.getBalance(buyer, address(tokenA)), 100 ether);
        assertEq(orderbook.getBalance(buyer, address(tokenB)), 300 ether);
        assertEq(orderbook.getBalance(seller, address(tokenA)), 400 ether);
        assertEq(orderbook.getBalance(seller, address(tokenB)), 200 ether);
    }

    function test_SettlePartialFill() public {
        vm.prank(buyer);
        orderbook.deposit(address(tokenB), 500 ether);
        vm.prank(seller);
        orderbook.deposit(address(tokenA), 500 ether);

        OrderTypes.Order memory buyOrder = _createBuyOrder(200 ether, 100 ether);
        bytes memory buySig = _signOrder(buyOrder, buyerPk);

        OrderTypes.Order memory sellOrder = _createSellOrder(100 ether, 200 ether);
        bytes memory sellSig = _signOrder(sellOrder, sellerPk);

        // Partial fill: 50 TKA
        orderbook.settleMatch(buyOrder, buySig, sellOrder, sellSig, 50 ether);

        assertEq(orderbook.getBalance(buyer, address(tokenA)), 50 ether);
        assertEq(orderbook.getBalance(buyer, address(tokenB)), 400 ether);
        assertEq(orderbook.getBalance(seller, address(tokenA)), 450 ether);
        assertEq(orderbook.getBalance(seller, address(tokenB)), 100 ether);

        // Fill remaining: 50 TKA
        orderbook.settleMatch(buyOrder, buySig, sellOrder, sellSig, 50 ether);

        assertEq(orderbook.getBalance(buyer, address(tokenA)), 100 ether);
        assertEq(orderbook.getBalance(buyer, address(tokenB)), 300 ether);
    }

    function test_SettleMatchOverfill() public {
        vm.prank(buyer);
        orderbook.deposit(address(tokenB), 500 ether);
        vm.prank(seller);
        orderbook.deposit(address(tokenA), 500 ether);

        OrderTypes.Order memory buyOrder = _createBuyOrder(200 ether, 100 ether);
        bytes memory buySig = _signOrder(buyOrder, buyerPk);

        OrderTypes.Order memory sellOrder = _createSellOrder(100 ether, 200 ether);
        bytes memory sellSig = _signOrder(sellOrder, sellerPk);

        // Try overfill
        vm.expectRevert("Sell overfill");
        orderbook.settleMatch(buyOrder, buySig, sellOrder, sellSig, 101 ether);
    }

    function test_SettleExpiredOrder() public {
        vm.prank(buyer);
        orderbook.deposit(address(tokenB), 500 ether);
        vm.prank(seller);
        orderbook.deposit(address(tokenA), 500 ether);

        OrderTypes.Order memory buyOrder = _createBuyOrder(200 ether, 100 ether);
        buyOrder.expiry = block.timestamp - 1;
        bytes memory buySig = _signOrder(buyOrder, buyerPk);

        OrderTypes.Order memory sellOrder = _createSellOrder(100 ether, 200 ether);
        bytes memory sellSig = _signOrder(sellOrder, sellerPk);

        vm.expectRevert("Buy order expired");
        orderbook.settleMatch(buyOrder, buySig, sellOrder, sellSig, 100 ether);
    }

    function test_SettleInvalidSignature() public {
        vm.prank(buyer);
        orderbook.deposit(address(tokenB), 500 ether);
        vm.prank(seller);
        orderbook.deposit(address(tokenA), 500 ether);

        OrderTypes.Order memory buyOrder = _createBuyOrder(200 ether, 100 ether);
        bytes memory buySig = _signOrder(buyOrder, sellerPk); // Wrong key

        OrderTypes.Order memory sellOrder = _createSellOrder(100 ether, 200 ether);
        bytes memory sellSig = _signOrder(sellOrder, sellerPk);

        vm.expectRevert("Invalid buy signature");
        orderbook.settleMatch(buyOrder, buySig, sellOrder, sellSig, 100 ether);
    }

    function test_OnlyOwnerCanSettle() public {
        vm.prank(buyer);
        orderbook.deposit(address(tokenB), 500 ether);
        vm.prank(seller);
        orderbook.deposit(address(tokenA), 500 ether);

        OrderTypes.Order memory buyOrder = _createBuyOrder(200 ether, 100 ether);
        bytes memory buySig = _signOrder(buyOrder, buyerPk);
        OrderTypes.Order memory sellOrder = _createSellOrder(100 ether, 200 ether);
        bytes memory sellSig = _signOrder(sellOrder, sellerPk);

        vm.prank(buyer);
        vm.expectRevert();
        orderbook.settleMatch(buyOrder, buySig, sellOrder, sellSig, 100 ether);
    }

    // ============ Cancel ============

    function test_CancelOrder() public {
        OrderTypes.Order memory order = _createBuyOrder(200 ether, 100 ether);
        vm.prank(buyer);
        orderbook.cancelOrder(order);
        assertTrue(orderbook.orderCancelled(OrderTypes.hash(order)));
    }

    function test_CancelOrderOnlyMaker() public {
        OrderTypes.Order memory order = _createBuyOrder(200 ether, 100 ether);
        vm.prank(seller);
        vm.expectRevert("Not order maker");
        orderbook.cancelOrder(order);
    }

    function test_SettleCancelledOrder() public {
        vm.prank(buyer);
        orderbook.deposit(address(tokenB), 500 ether);
        vm.prank(seller);
        orderbook.deposit(address(tokenA), 500 ether);

        OrderTypes.Order memory buyOrder = _createBuyOrder(200 ether, 100 ether);
        bytes memory buySig = _signOrder(buyOrder, buyerPk);
        OrderTypes.Order memory sellOrder = _createSellOrder(100 ether, 200 ether);
        bytes memory sellSig = _signOrder(sellOrder, sellerPk);

        vm.prank(buyer);
        orderbook.cancelOrder(buyOrder);

        vm.expectRevert("Buy order cancelled");
        orderbook.settleMatch(buyOrder, buySig, sellOrder, sellSig, 100 ether);
    }

    // ============ Min Nonce ============

    function test_IncrementMinNonce() public {
        vm.prank(buyer);
        orderbook.incrementMinNonce(5);
        assertEq(orderbook.minNonce(buyer), 5);
    }

    function test_SettleBelowMinNonce() public {
        vm.prank(buyer);
        orderbook.deposit(address(tokenB), 500 ether);
        vm.prank(seller);
        orderbook.deposit(address(tokenA), 500 ether);

        vm.prank(buyer);
        orderbook.incrementMinNonce(5);

        OrderTypes.Order memory buyOrder = _createBuyOrder(200 ether, 100 ether);
        // buyOrder.nonce is 0, below min nonce of 5
        bytes memory buySig = _signOrder(buyOrder, buyerPk);
        OrderTypes.Order memory sellOrder = _createSellOrder(100 ether, 200 ether);
        bytes memory sellSig = _signOrder(sellOrder, sellerPk);

        vm.expectRevert("Buy nonce too low");
        orderbook.settleMatch(buyOrder, buySig, sellOrder, sellSig, 100 ether);
    }

    // ============ Price Incompatibility ============

    function test_PriceIncompatible() public {
        vm.prank(buyer);
        orderbook.deposit(address(tokenB), 500 ether);
        vm.prank(seller);
        orderbook.deposit(address(tokenA), 500 ether);

        // Buyer willing to pay 1 TKB/TKA
        OrderTypes.Order memory buyOrder = _createBuyOrder(100 ether, 100 ether);
        bytes memory buySig = _signOrder(buyOrder, buyerPk);

        // Seller wants 3 TKB/TKA
        OrderTypes.Order memory sellOrder = _createSellOrder(100 ether, 300 ether);
        bytes memory sellSig = _signOrder(sellOrder, sellerPk);

        vm.expectRevert("Price incompatible");
        orderbook.settleMatch(buyOrder, buySig, sellOrder, sellSig, 100 ether);
    }
}
