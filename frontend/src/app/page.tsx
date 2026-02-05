import { ConnectButton } from "@/components/wallet/ConnectButton";
import { DepositWithdraw } from "@/components/deposit/DepositWithdraw";
import { OrderBook } from "@/components/orderbook/OrderBook";
import { OrderForm } from "@/components/order/OrderForm";
import { TradeHistory } from "@/components/trades/TradeHistory";
import { UserOrders } from "@/components/orders/UserOrders";

export default function Home() {
  return (
    <div className="min-h-screen">
      {/* Header */}
      <header className="border-b border-gray-800 px-6 py-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h1 className="text-xl font-bold">Nexus DEX</h1>
          <span className="text-xs text-gray-500 bg-gray-800 px-2 py-0.5 rounded">
            TKA/TKB
          </span>
        </div>
        <ConnectButton />
      </header>

      {/* Main Trading Layout */}
      <main className="max-w-7xl mx-auto p-4 grid grid-cols-1 lg:grid-cols-12 gap-4">
        {/* Left: Order Book */}
        <div className="lg:col-span-3">
          <OrderBook />
        </div>

        {/* Center: Chart placeholder + Trades */}
        <div className="lg:col-span-5 space-y-4">
          <div className="bg-gray-800 rounded-xl p-4 h-48 flex items-center justify-center">
            <span className="text-gray-600 text-sm">
              TKA/TKB - Price Chart (coming soon)
            </span>
          </div>
          <TradeHistory />
          <UserOrders />
        </div>

        {/* Right: Order Form + Vault */}
        <div className="lg:col-span-4 space-y-4">
          <OrderForm />
          <DepositWithdraw />
        </div>
      </main>
    </div>
  );
}
