// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {OrderTypes} from "./libraries/OrderTypes.sol";

abstract contract OrderValidator {
    bytes32 public immutable DOMAIN_SEPARATOR;

    constructor() {
        DOMAIN_SEPARATOR = keccak256(
            abi.encode(
                keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"),
                keccak256("NexusOrderBook"),
                keccak256("1"),
                block.chainid,
                address(this)
            )
        );
    }

    function _hashTypedData(bytes32 structHash) internal view returns (bytes32) {
        return keccak256(abi.encodePacked("\x19\x01", DOMAIN_SEPARATOR, structHash));
    }

    function _validateSignature(
        OrderTypes.Order memory order,
        bytes memory signature
    ) internal view returns (bool) {
        bytes32 digest = _hashTypedData(OrderTypes.hash(order));
        address signer = _recover(digest, signature);
        return signer == order.maker;
    }

    function _recover(bytes32 digest, bytes memory signature) internal pure returns (address) {
        require(signature.length == 65, "Invalid signature length");

        bytes32 r;
        bytes32 s;
        uint8 v;

        assembly {
            r := mload(add(signature, 0x20))
            s := mload(add(signature, 0x40))
            v := byte(0, mload(add(signature, 0x60)))
        }

        if (v < 27) {
            v += 27;
        }

        require(v == 27 || v == 28, "Invalid signature v");
        address signer = ecrecover(digest, v, r, s);
        require(signer != address(0), "Invalid signature");
        return signer;
    }
}
