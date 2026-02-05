// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

library OrderTypes {
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

    bytes32 internal constant ORDER_TYPEHASH = keccak256(
        "Order(address maker,address tokenSell,address tokenBuy,uint256 amountSell,uint256 amountBuy,uint256 expiry,uint256 nonce,uint256 salt)"
    );

    function hash(Order memory order) internal pure returns (bytes32) {
        return keccak256(
            abi.encode(
                ORDER_TYPEHASH,
                order.maker,
                order.tokenSell,
                order.tokenBuy,
                order.amountSell,
                order.amountBuy,
                order.expiry,
                order.nonce,
                order.salt
            )
        );
    }
}
