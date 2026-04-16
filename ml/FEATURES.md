# XGBoost Bottom/Top Detection — Feature Specification

## Objective

Predict `future_return_60d` (price 60 days later / today's price - 1) for each instrument on each day. High positive values indicate bottoms, high negative values indicate tops. This is a regression problem — the model outputs a continuous return prediction, not a binary label.

## Model

- **Algorithm**: XGBoost (regression)
- **Input**: 30 features per instrument per day
- **Output**: Predicted 60-day future return
- **Training data**: All instruments, all days — one row per instrument per day

---

## Features (30)

### Price Trend (7)

| # | Feature | Description |
|---|---------|-------------|
| 1 | `slope_20d` | Linear regression slope of price over last 20 days. Negative = falling, positive = rising. |
| 2 | `slope_60d` | Linear regression slope of price over last 60 days. Captures longer-term trend direction. |
| 3 | `curvature_20d` | Slope of the slope over 20 days (second derivative). Positive during decline = fall is decelerating = approaching bottom. |
| 4 | `curvature_60d` | Slope of the slope over 60 days. Captures longer-term trend acceleration/deceleration. |
| 5 | `drawdown_magnitude` | Percentage decline from the most recent high. E.g. -0.35 = stock is down 35% from its high. |
| 6 | `drawdown_duration` | Number of consecutive days the price has been below its recent high. Measures how long the decline has lasted. |
| 7 | `price_vs_200d_ma` | Price divided by its 200-day moving average. Below 1.0 = long-term downtrend. |

### Volume/Trade (3)

| # | Feature | Description |
|---|---------|-------------|
| 8 | `volume_per_trade` | Raw volume per trade value for the day. |
| 9 | `vpt_ratio_20d` | Volume per trade divided by its 20-day moving average. >1 = above normal, 2.0 = double normal activity. |
| 10 | `vpt_ratio_60d` | Volume per trade divided by its 60-day moving average. Captures longer-term volume regime. |

### Delivery (2)

| # | Feature | Description |
|---|---------|-------------|
| 11 | `delivery_pct` | Raw delivery percentage for the day. High delivery = actual buying, not just intraday speculation. |
| 12 | `avg_delivery_pct_20d` | 20-day moving average of delivery percentage. Smooths out daily noise. |

### Shorts (1)

| # | Feature | Description |
|---|---------|-------------|
| 13 | `short_activity` | Short quantity divided by its 60-day moving average. >1 = shorts more active than usual, <1 = less active. |

### Bulk Deals (4)

| # | Feature | Description |
|---|---------|-------------|
| 14 | `bulk_buy_frequency` | Number of bulk buy events in the last 60 days. Measures how actively institutions are buying. |
| 15 | `bulk_buy_total_qty` | Total quantity purchased across all bulk buy deals in last 60 days. Measures magnitude of accumulation. |
| 16 | `bulk_sell_frequency` | Number of bulk sell events in the last 60 days. |
| 17 | `bulk_net` | Total bulk buy quantity minus total bulk sell quantity in last 60 days. Positive = net accumulation. |

### Block Deals (4)

| # | Feature | Description |
|---|---------|-------------|
| 18 | `block_buy_frequency` | Number of block buy events in the last 60 days. |
| 19 | `block_buy_total_qty` | Total quantity purchased across all block deals in last 60 days. |
| 20 | `block_sell_frequency` | Number of block sell events in the last 60 days. |
| 21 | `block_net` | Total block buy quantity minus total block sell quantity in last 60 days. |

### Insider Trades (4)

| # | Feature | Description |
|---|---------|-------------|
| 22 | `insider_buy_frequency` | Number of insider buy events in the last 60 days. Insiders buying during decline = strong conviction signal. |
| 23 | `insider_buy_total_value` | Total rupee value of insider purchases in last 60 days. Measures magnitude of insider conviction. |
| 24 | `insider_sell_frequency` | Number of insider sell events in the last 60 days. |
| 25 | `insider_net_value` | Insider buy value minus sell value in last 60 days. Positive = insiders are net buyers. |

### Market Context (3)

| # | Feature | Description |
|---|---------|-------------|
| 26 | `volatility` | Daily volatility from NSE (cmDailyVolatility). Bottoms often coincide with peak volatility. |
| 27 | `buy_sell_ratio` | End-of-day order book totalBuyQuantity / totalSellQuantity. >1 = more buyers. |
| 28 | `industry` | Basic industry encoded as integer. Allows model to learn industry-specific patterns. |

### Divergence (2)

| # | Feature | Description |
|---|---------|-------------|
| 29 | `divergence_20d` | VPT % change over 20 days minus price % change over 20 days. Positive = volume/trade rising while price falling = accumulation. |
| 30 | `divergence_60d` | Same divergence over 60 days. Captures longer-term accumulation/distribution. |

---

## Label

| Field | Description |
|-------|-------------|
| `future_return_60d` | (price 60 days later / today's price) - 1. Continuous value. E.g. 0.35 = price rose 35%. |

---

## Data Sources

| Source | Fields Used |
|--------|------------|
| Daily price data (existing DB) | adj_close, volume, volume_per_trade, no_of_trades, delivery_percentage |
| Short selling data (existing DB) | short_quantity |
| NSE equity quote API (new) | cmDailyVolatility, totalBuyQuantity, totalSellQuantity, pdSymbolPe, yearHigh, yearLow, basicIndustry |
| NSE bulk deals (new) | buy/sell quantity, frequency |
| NSE block deals (new) | buy/sell quantity, frequency |
| NSE insider trades (new) | buy/sell value, frequency |

---

## Key Concepts

### Why Slope + Curvature

A bottom is a trend reversal — the price stops falling and starts rising. Slope captures direction, curvature captures whether that direction is changing. When slope is negative but curvature is positive, the decline is decelerating — a bottom is forming.

### Why Regression Instead of Classification

Instead of defining arbitrary thresholds for "is this a bottom?" (which led to many edge cases), we predict the actual future return. The model ranks stocks by predicted return — the highest predicted returns are the strongest bottom signals. No threshold decisions needed.

### Why Divergence

When volume/trade rises while price falls, large buyers are accumulating shares from panicking sellers. This divergence between price and volume/trade is one of the strongest bottom signals. The `divergence_20d` and `divergence_60d` features capture this directly.

### Feature Importance

After training, run `xgb.plot_importance(model, importance_type='gain')` and SHAP analysis to determine which features actually matter. Drop features that don't contribute. The feature importance output is valuable even if model accuracy is imperfect — it tells you which signals actually correlate with future returns in your data.
