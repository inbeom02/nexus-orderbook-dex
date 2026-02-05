// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {SafeERC20} from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import {ReentrancyGuard} from "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {OrderTypes} from "./libraries/OrderTypes.sol";
import {OrderValidator} from "./OrderValidator.sol";

contract NexusOrderBook is OrderValidator, ReentrancyGuard, Ownable {
    using SafeERC20 for IERC20;

    // user => token => balance
    mapping(address => mapping(address => uint256)) public balances;

    // order hash => filled amount (in base token units)
    mapping(bytes32 => uint256) public orderFills;

    // order hash => cancelled
    mapping(bytes32 => bool) public orderCancelled;

    // maker => minimum nonce (bulk cancel)
    mapping(address => uint256) public minNonce;

    event Deposit(address indexed user, address indexed token, uint256 amount);
    event Withdraw(address indexed user, address indexed token, uint256 amount);
    event TradeSettled(
        bytes32 indexed buyOrderHash,
        bytes32 indexed sellOrderHash,
        address buyer,
        address seller,
        uint256 baseAmount,
        uint256 quoteAmount
    );
    event OrderCancelled(bytes32 indexed orderHash, address indexed maker);
    event MinNonceIncremented(address indexed maker, uint256 newMinNonce);

    constructor() Ownable(msg.sender) {}

    function deposit(address token, uint256 amount) external nonReentrant {
        require(amount > 0, "Zero amount");
        IERC20(token).safeTransferFrom(msg.sender, address(this), amount);
        balances[msg.sender][token] += amount;
        emit Deposit(msg.sender, token, amount);
    }

    function withdraw(address token, uint256 amount) external nonReentrant {
        require(amount > 0, "Zero amount");
        require(balances[msg.sender][token] >= amount, "Insufficient balance");
        balances[msg.sender][token] -= amount;
        IERC20(token).safeTransfer(msg.sender, amount);
        emit Withdraw(msg.sender, token, amount);
    }

    function settleMatch(
        OrderTypes.Order calldata buyOrder,
        bytes calldata buySig,
        OrderTypes.Order calldata sellOrder,
        bytes calldata sellSig,
        uint256 fillAmount
    ) external onlyOwner nonReentrant {
        _validateOrders(buyOrder, buySig, sellOrder, sellSig);

        bytes32 buyHash = OrderTypes.hash(buyOrder);
        bytes32 sellHash = OrderTypes.hash(sellOrder);

        require(!orderCancelled[buyHash], "Buy order cancelled");
        require(!orderCancelled[sellHash], "Sell order cancelled");

        uint256 sellerReceives = (fillAmount * sellOrder.amountBuy) / sellOrder.amountSell;
        uint256 buyerPays = (fillAmount * buyOrder.amountSell) / buyOrder.amountBuy;
        require(buyerPays >= sellerReceives, "Price incompatible");

        require(orderFills[sellHash] + fillAmount <= sellOrder.amountSell, "Sell overfill");
        require(orderFills[buyHash] + fillAmount <= buyOrder.amountBuy, "Buy overfill");

        orderFills[sellHash] += fillAmount;
        orderFills[buyHash] += fillAmount;

        _executeTransfers(buyOrder, sellOrder, fillAmount, sellerReceives);

        emit TradeSettled(buyHash, sellHash, buyOrder.maker, sellOrder.maker, fillAmount, sellerReceives);
    }

    function _validateOrders(
        OrderTypes.Order calldata buyOrder,
        bytes calldata buySig,
        OrderTypes.Order calldata sellOrder,
        bytes calldata sellSig
    ) internal view {
        require(buyOrder.tokenBuy == sellOrder.tokenSell, "Token mismatch: buy");
        require(buyOrder.tokenSell == sellOrder.tokenBuy, "Token mismatch: sell");
        require(_validateSignature(buyOrder, buySig), "Invalid buy signature");
        require(_validateSignature(sellOrder, sellSig), "Invalid sell signature");
        require(block.timestamp <= buyOrder.expiry, "Buy order expired");
        require(block.timestamp <= sellOrder.expiry, "Sell order expired");
        require(buyOrder.nonce >= minNonce[buyOrder.maker], "Buy nonce too low");
        require(sellOrder.nonce >= minNonce[sellOrder.maker], "Sell nonce too low");
    }

    function _executeTransfers(
        OrderTypes.Order calldata buyOrder,
        OrderTypes.Order calldata sellOrder,
        uint256 fillAmount,
        uint256 sellerReceives
    ) internal {
        require(balances[buyOrder.maker][buyOrder.tokenSell] >= sellerReceives, "Buyer insufficient balance");
        require(balances[sellOrder.maker][sellOrder.tokenSell] >= fillAmount, "Seller insufficient balance");

        // Buyer sends quote token to seller
        balances[buyOrder.maker][buyOrder.tokenSell] -= sellerReceives;
        balances[sellOrder.maker][sellOrder.tokenBuy] += sellerReceives;

        // Seller sends base token to buyer
        balances[sellOrder.maker][sellOrder.tokenSell] -= fillAmount;
        balances[buyOrder.maker][buyOrder.tokenBuy] += fillAmount;
    }

    function cancelOrder(OrderTypes.Order calldata order) external {
        require(msg.sender == order.maker, "Not order maker");
        bytes32 orderHash = OrderTypes.hash(order);
        require(!orderCancelled[orderHash], "Already cancelled");
        orderCancelled[orderHash] = true;
        emit OrderCancelled(orderHash, msg.sender);
    }

    function incrementMinNonce(uint256 newMinNonce) external {
        require(newMinNonce > minNonce[msg.sender], "Nonce must increase");
        minNonce[msg.sender] = newMinNonce;
        emit MinNonceIncremented(msg.sender, newMinNonce);
    }

    function getBalance(address user, address token) external view returns (uint256) {
        return balances[user][token];
    }

    function getOrderFill(bytes32 orderHash) external view returns (uint256) {
        return orderFills[orderHash];
    }
}
