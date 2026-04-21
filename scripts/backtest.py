import os
import pandas as pd
import numpy as np
from sqlalchemy import create_engine

DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://localhost:5432/ninja_trader")

engine = create_engine(DATABASE_URL)

def load_historical(symbol=None):
    """Load historical data into a DataFrame."""
    query = """
        SELECT symbol, date, o, h, l, c, adjusted_close_price, volume,
               no_of_trades, volume_per_trade, delivery_percentage, vwap
        FROM historicaldata
    """
    params = {}
    if symbol:
        query += " WHERE symbol = %(symbol)s"
        params["symbol"] = symbol
    query += " ORDER BY symbol, date"
    
    df = pd.read_sql(query, engine, params=params or None, parse_dates=["date"])
    return df


def load_instruments():
    """Load active instruments."""
    query = """
        SELECT trading_symbol, instrument_token, market_cap, basic_industry,
               "index", tag, watchlist
        FROM instruments
        WHERE active = true
        ORDER BY market_cap DESC
    """
    return pd.read_sql(query, engine)


def compute_signals(df, window='1095D'):
    """Compute VPT MA, z-scores, sigmoid VPT score, and divergence.
    
    Z-scores are computed over a rolling 3-year calendar window (1095 days).
    """
    df = df.set_index('date').sort_index()

    # VPT 20-day Simple Moving Average
    df['vpt_ma20'] = df['volume_per_trade'].rolling(20).mean()

    # Rolling z-score normalization over 3-year calendar window
    price_mean = df['adjusted_close_price'].rolling(window).mean()
    price_std = df['adjusted_close_price'].rolling(window).std()
    df['price_z'] = (df['adjusted_close_price'] - price_mean) / price_std

    vpt_mean = df['vpt_ma20'].rolling(window).mean()
    vpt_std = df['vpt_ma20'].rolling(window).std()
    df['vpt_z'] = (df['vpt_ma20'] - vpt_mean) / vpt_std

    # Sigmoid: z-score -> 0-100
    df['vpt_score'] = 100 / (1 + np.exp(-df['vpt_z']))

    # Divergence: VPT z-score minus price z-score
    df['divergence'] = df['vpt_z'] - df['price_z']

    return df.reset_index()


def backtest_divergence(df, target_pct=1.0, lookback='1095D'):
    """Backtest: buy when divergence hits 3-year max, sell at target % return.
    
    Args:
        df: DataFrame with signals computed (needs 'divergence' and 'adjusted_close_price')
        target_pct: sell when return reaches this fraction (default 1.0 = 100%)
        lookback: rolling window for max divergence (default '1095D' = 3 years)
    
    Returns:
        trades: list of trade dicts with entry/exit details
    """
    df = df.dropna(subset=['divergence']).copy()
    df = df.set_index('date').sort_index()

    # Rolling 3-year max of divergence
    df['div_3y_max'] = df['divergence'].rolling(lookback).max()

    # Signal: divergence equals its 3-year max (new high)
    df['signal'] = df['divergence'] == df['div_3y_max']

    df = df.reset_index()
    prices = df['adjusted_close_price'].values
    divergences = df['divergence'].values
    dates = df['date'].values
    signals = df['signal'].values

    trades = []
    i = 0
    while i < len(df):
        if signals[i]:
            entry_price = prices[i]
            entry_date = dates[i]
            entry_div = divergences[i]
            target_price = entry_price * (1 + target_pct)

            # Scan forward for target hit
            exit_price = None
            exit_date = None
            for j in range(i + 1, len(df)):
                if prices[j] >= target_price:
                    exit_price = prices[j]
                    exit_date = dates[j]
                    break

            # If target never hit, position is still open
            if exit_price is None:
                exit_price = prices[-1]
                exit_date = dates[-1]
                status = 'open'
            else:
                status = 'closed'

            pnl_pct = (exit_price - entry_price) / entry_price * 100
            holding_days = (pd.Timestamp(exit_date) - pd.Timestamp(entry_date)).days

            trades.append({
                'entry_date': pd.Timestamp(entry_date).date(),
                'exit_date': pd.Timestamp(exit_date).date(),
                'entry_price': round(entry_price, 2),
                'exit_price': round(exit_price, 2),
                'entry_divergence': round(entry_div, 2),
                'pnl_pct': round(pnl_pct, 2),
                'holding_days': holding_days,
                'status': status,
            })

            # Skip to exit date to avoid overlapping trades
            if status == 'closed':
                i = j + 1
                continue
            else:
                break
        i += 1

    return trades


def print_backtest_results(symbol, trades):
    """Print summary of backtest results."""
    if not trades:
        print(f"\n{symbol}: No trades triggered")
        return

    df_trades = pd.DataFrame(trades)
    closed = df_trades[df_trades['status'] == 'closed']
    open_trades = df_trades[df_trades['status'] == 'open']

    print(f"\n{'='*70}")
    print(f"{symbol} — Backtest Results (divergence = 3yr max, target 100%)")
    print(f"{'='*70}")
    print(f"Total trades:  {len(df_trades)}")
    print(f"Closed:        {len(closed)}")
    print(f"Still open:    {len(open_trades)}")

    if len(closed) > 0:
        wins = closed[closed['pnl_pct'] >= 100]
        print(f"\nClosed trades:")
        print(f"  Hit target:    {len(wins)}/{len(closed)} ({len(wins)/len(closed)*100:.0f}%)")
        print(f"  Avg holding:   {closed['holding_days'].mean():.0f} days")
        print(f"  Min holding:   {closed['holding_days'].min()} days")
        print(f"  Max holding:   {closed['holding_days'].max()} days")

    if len(open_trades) > 0:
        print(f"\nOpen trades:")
        for _, t in open_trades.iterrows():
            print(f"  Entry {t['entry_date']} @ {t['entry_price']} -> current {t['exit_price']} ({t['pnl_pct']:+.1f}%, {t['holding_days']}d)")

    print(f"\nAll trades:")
    print(df_trades.to_string(index=False))


if __name__ == "__main__":
    instruments = load_instruments()
    print(f"Active instruments: {len(instruments)}")
    
    symbol = instruments.iloc[0]["trading_symbol"] if len(instruments) > 0 else None
    if symbol:
        df = load_historical(symbol)
        df = compute_signals(df)
        print(f"\n{symbol}: {len(df)} days of data")
        print(f"Date range: {df['date'].min().date()} to {df['date'].max().date()}")

        latest = df.dropna(subset=['vpt_score']).iloc[-1]
        print(f"\nLatest signals:")
        print(f"  Price:      {latest['adjusted_close_price']:.2f}")
        print(f"  VPT MA20:   {latest['vpt_ma20']:.0f}")
        print(f"  VPT Score:  {latest['vpt_score']:.1f}")
        print(f"  Divergence: {latest['divergence']:.2f}")

        trades = backtest_divergence(df)
        print_backtest_results(symbol, trades)
