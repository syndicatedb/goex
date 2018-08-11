# Trading (Private API)

In every exchange `TradingProvider()` is responsible for trading.

## Trades

`Trades(Trade) []Trade, error`

To load trades you can pass last trade to load trades.
Why last trade?

Exchanges have different set of params to filter trades.
We using last trade to gather this params.

For example:

`exchange.TradingProvider().Trades(Trade{})` - will load first N trades.

```

exchange.TradingProvider().Trades(Trade{
  ID: 123,
  Timestamp: 1532248147,
  ...
})

```

will load trades from ID > 123 or Timestamp > 1532248147 (depending on exchange)