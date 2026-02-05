import { http, createConfig } from "wagmi";
import { sepolia, hardhat } from "wagmi/chains";

const localhost = {
  ...hardhat,
  id: 31337,
  name: "Localhost",
  rpcUrls: {
    default: { http: ["http://127.0.0.1:8545"] },
  },
} as const;

export const config = createConfig({
  chains: [localhost, sepolia],
  transports: {
    [localhost.id]: http(),
    [sepolia.id]: http(),
  },
});

declare module "wagmi" {
  interface Register {
    config: typeof config;
  }
}
